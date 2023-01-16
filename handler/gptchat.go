package handler

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ChatGPT-Hackers/ChatGPT-API-server/types"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

const (
	ChatGPTHost    = "ChatGPT.Host"
	ChatGPTToken   = "ChatGPT.Token"
	ChatGPTTimeout = "ChatGPT.Timeout"
)

func NewChatGPTClient() *ChatGPTClient {
	return &ChatGPTClient{
		host:  viper.GetString(ChatGPTHost),
		token: viper.GetString(ChatGPTToken),
		cli: &http.Client{
			Timeout: viper.GetDuration(ChatGPTTimeout),
		},
	}
}

type ChatGPTClient struct {
	host  string
	token string
	cli   *http.Client
}

func (c *ChatGPTClient) Ask(inputText, conversationId, lastMessageId string) (*types.ChatGptResponse, error) {
	chatResponse := &types.ChatGptResponse{}
	chatRequest := types.ChatGptRequest{
		MessageId:      uuid.NewString(),
		ConversationId: conversationId,
		ParentId:       lastMessageId,
		Content:        inputText,
	}
	bts, err := json.Marshal(chatRequest)
	if err != nil {
		return nil, err
	}
	log.Println("ask >>: ", string(bts), err)
	body := bytes.NewBuffer(bts)
	req, err := http.NewRequest(http.MethodPost, c.host+"/api/ask", body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", c.token)
	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(resp.Body).Decode(chatResponse)
	return chatResponse, err
}
