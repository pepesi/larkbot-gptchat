package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

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

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

type ChatGptRequest struct {
	ChatID   string `json:"chatid"`
	Question string `json:"question"`
}

func (c *ChatGPTClient) Chat(inputText, chatID string) (io.ReadCloser, error) {
	chatRequest := ChatGptRequest{
		ChatID:   chatID,
		Question: inputText,
	}
	bts, err := json.Marshal(chatRequest)
	if err != nil {
		return nil, err
	}
	body := bytes.NewBuffer(bts)
	req, err := http.NewRequest(http.MethodPost, c.host+"/chat", body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", c.token)
	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, err
}

func (c *ChatGPTClient) SetBasePrompt(chatID, basePrompt string) error {
	data, _ := json.Marshal(map[string]interface{}{
		"chatid":      chatID,
		"base_prompt": basePrompt,
	})
	body := bytes.NewBuffer(data)
	req, err := http.NewRequest(http.MethodPost, c.host+"/set_base_prompt", body)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", c.token)
	resp, err := c.cli.Do(req)
	if err != nil {
		return err
	}
	return handleResp(resp.Body)
}

func (c *ChatGPTClient) Delete(chatID string) error {
	data, _ := json.Marshal(map[string]interface{}{
		"chatid": chatID,
	})
	body := bytes.NewBuffer(data)
	req, err := http.NewRequest(http.MethodPost, c.host+"/delete", body)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", c.token)
	resp, err := c.cli.Do(req)
	if err != nil {
		return err
	}
	return handleResp(resp.Body)
}

func handleResp(r io.ReadCloser) error {
	decoder := json.NewDecoder(r)
	rep := &Response{}
	if err := decoder.Decode(rep); err != nil {
		return err
	}
	if rep.Code != 0 {
		return errors.New(rep.Message)
	}
	return nil
}
