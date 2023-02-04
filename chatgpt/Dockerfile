FROM python:3.10-alpine

RUN  apk add gcc musl-dev python3-dev libffi-dev openssl-dev cargo pkgconfig
COPY req.txt req.txt
RUN pip install -U pip && pip install -r req.txt -i https://mirrors.cloud.tencent.com/pypi/simple
COPY bot.py bot.py
CMD ["gunicorn", "bot:app"]
