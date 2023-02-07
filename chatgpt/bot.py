import os
from datetime import date
import sys
from typing import List
import uuid

import tiktoken
import redis
import openai
from flask import (
    Flask,
    Response,
    request,
    stream_with_context,
    jsonify,
)


openai.proxy = os.environ.get("OPENAI_PROXY")
BOTNAME = os.environ.get("BOTNAME", "ChatGPT")
EXTRA_PROMPT = os.environ.get("EXTRA_PROMPT")
API_KEY = os.environ.get("OPENAI_API_KEY")
ENGINE = os.environ.get("GPT_ENGINE", "text-chat-davinci-002-20221122")
ENCODER = tiktoken.get_encoding("gpt2")
SYMBOL_END = "<|im_end|>"
SYMBOL_STOP = "\n\n\n"
SYMBOL_USER = "User:"
SYMBOL_CHAT = f"{BOTNAME}:"
MAX_TOKENS = 4096
DEFAULT_BASE_PROMPT  = f"你的名字是{BOTNAME}。{EXTRA_PROMPT} 请使用对话式回答。 现在的时间是: {date.today()}\n\n"  # noqa


class ChatGPTAPI:
    def __init__(self, api_key: str) -> None:
        openai.api_key = api_key
        redis_cli = redis.Redis(
            host=os.environ.get("REDIS_HOST", "localhost"),
            port=int(os.environ.get("REDIS_PORT", 6379)),
            username=os.environ.get("REDIS_USERNAME", ""),
            password=os.environ.get("REDIS_PASSWORD", ""),
        )
        self.promptor = Promptor(cli=redis_cli)

    def _process_completion(self, chat_id: str, user_request: str, completion: dict) -> dict:
        if not completion.get("choices"):
            raise Exception("ChatGPT API returned no choices")
        if not completion["choices"][0].get("text"):
            raise Exception("ChatGPT API returned no text")
        self.promptor.add_history(
            chat_id,
            f"{SYMBOL_USER} {user_request}{SYMBOL_STOP}{SYMBOL_CHAT} {completion['choices'][0]['text']}\n"
        )
        return completion

    def _process_completion_stream(self, chat_id: str, user_request: str, completion: dict) -> str:
        full_response = ""
        for sep in completion:
            if not sep["choices"]:
                raise Exception("ChatGPT API returned no choices")
            if sep["choices"][0].get("finish_details") is not None:
                break
            if sep["choices"][0].get("text") is None:
                raise Exception("ChatGPT API returned no text")
            if sep["choices"][0]["text"] == SYMBOL_END:
                break
            yield sep["choices"][0]["text"]
            full_response += sep["choices"][0]["text"]

        if full_response.endswith(SYMBOL_END):
            full_response = full_response[:-len(SYMBOL_END)]
        self.promptor.add_history(
            chat_id,
            f"{SYMBOL_USER} {user_request}{SYMBOL_STOP}{SYMBOL_CHAT} {full_response}{SYMBOL_END}\n"
        )

    def set_base_prompt(self, chat_id: str, content: str):
        self.promptor.set_base_prompt(chat_id, content)

    def reset_chat(self, chat_id: str):
        self.promptor.clean(chat_id)

    def ask(self, chat_id: str, input_text: str, temperature: float = 0.5, stream: bool = False) -> dict:
        gen_prompt, max_tokens = self.promptor.construct_prompt(chat_id, input_text)
        completion = openai.Completion.create(
            engine=ENGINE,
            prompt=gen_prompt,
            temperature=temperature,
            max_tokens=max_tokens,
            stop=[SYMBOL_STOP],
            stream=stream,
        )
        if stream:
            return self._process_completion_stream(chat_id, user_request=input_text, completion=completion)
        else:
            return self._process_completion(chat_id, input_text, completion)


class Promptor:
    def __init__(self, cli: redis.Redis):
        self.cli = cli

    def _base_prompt_key(self, chat_id: str):
        return f"base_prompt_{chat_id}"

    def _history_key(self, chat_id: str):
        return f"chat_history_{chat_id}"

    def _history_start_key(self, chat_id: str):
        return f"chat_history_start_{chat_id}"

    def get_history_start(self, chat_id: str) -> int:
        idx = self.cli.get(self._history_start_key(chat_id))
        if not idx:
            return 0
        return int(idx)

    def incr_history_start(self, chat_id: str):
        self.cli.incr(self._history_start_key(chat_id))

    def add_history(self, chat_id: str, data: str) -> None:
        self.cli.rpush(self._history_key(chat_id), data)

    def history(self, chat_id: str, start_idx: int = 0) -> List[str]:
        return [el.decode() for el in self.cli.lrange(self._history_key(chat_id), start_idx, -1)]

    def get_base_prompt(self, chat_id: str) -> str:
        content = self.cli.get(self._base_prompt_key(chat_id))
        if not content:
            return DEFAULT_BASE_PROMPT
        return content.decode()

    def set_base_prompt(self, chat_id: str, content: str):
        if not content:
            return
        self.cli.set(self._base_prompt_key(chat_id), content + SYMBOL_STOP)

    def construct_prompt(self, chat_id: str, input_text: str) -> str:
        base_prompt = self.get_base_prompt(chat_id)
        history_start_idx = self.get_history_start(chat_id)
        history = "\n".join(self.history(chat_id, history_start_idx))
        new_prompt = f"{base_prompt}{history}{SYMBOL_USER} {input_text}\n{SYMBOL_CHAT}"
        length = len(ENCODER.encode(new_prompt))
        while length > MAX_TOKENS:
            self.incr_history_start(chat_id)
            history = "\n".join(self.history(chat_id, history_start_idx))
            new_prompt = f"{base_prompt}{history}{SYMBOL_USER} {input_text}\n{SYMBOL_CHAT}"
            length = len(ENCODER.encode(new_prompt))
        return new_prompt, MAX_TOKENS - length

    def clean(self, chat_id: str):
        self.cli.delete(self._base_prompt_key(chat_id))
        self.cli.delete(self._history_key(chat_id))
        self.cli.delete(self._history_start_key(chat_id))


bot = ChatGPTAPI(api_key=API_KEY)

if os.environ.get("CHATAS_HTTP_SERVER"):
    app = Flask(__name__)

    @app.route("/delete", methods=["POST"])
    def reset():
        chat_id = request.json.get("chatid")
        if not chat_id:
            return jsonify({"code": 1, "msg": "error"})
        bot.reset_chat(chat_id)
        return jsonify({"code": 0, "msg": "ok"})

    @app.route("/set_base_prompt", methods=["POST"])
    def set_prompt():
        chat_id = request.json.get("chatid")
        base_prompt = request.json.get("base_prompt")
        if not chat_id or not base_prompt:
            return jsonify({"code": 1, "msg": "error"})
        bot.set_base_prompt(chat_id, base_prompt)
        return jsonify({"code": 0, "msg": "ok"})

    @app.route("/chat", methods=["POST"])
    def query():
        question = request.json.get("question")
        chat_id = request.json.get("chatid")

        def generate():
            for part in bot.ask(chat_id, question, stream=True):
                yield part

        return Response(stream_with_context(generate()), headers={"chat_id": chat_id})

    if __name__ == "__main__":
        app.run()
else:
    uid = str(uuid.uuid4())
    chatid = f"chat_{uid}"
    bot.set_base_prompt(chatid, input("set base prompt: "))
    while True:
        try:
            content = input("\nUser:\n")
        except KeyboardInterrupt:
            print("\nExiting...")
            sys.exit()
        if not content:
            continue
        response = bot.ask(chatid, content, stream=True)
        print("Reply: ")
        sys.stdout.flush()
        for resp in response:
            print(resp, end="")
            sys.stdout.flush()
        print()
