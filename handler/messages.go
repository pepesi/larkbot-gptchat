package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/spf13/viper"
)

const (
	BotName               = "BotName"
	LarkAppId             = "Lark.AppID"
	LarkAppSecret         = "Lark.AppSecret"
	MessageExtractFailed  = "Message.ExtractFailed"
	MessageUpstreamFailed = "Message.UpstreamFailed"
)

func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		BotName:  viper.GetString(BotName),
		client:   NewChatGPTClient(),
		lark:     lark.NewClient(viper.GetString(LarkAppId), viper.GetString(LarkAppSecret)),
		filter:   NewFilter(),
		sessions: make(map[string]*Session),
		locker:   &sync.Mutex{},
	}
}

type MessageHandler struct {
	BotName  string
	client   *ChatGPTClient
	lark     *lark.Client
	filter   *KeywordsFilter
	sessions map[string]*Session
	locker   sync.Locker
}

func (h *MessageHandler) Handle(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	h.Enqueue(event)
	return nil
}

func (h *MessageHandler) Enqueue(event *larkim.P2MessageReceiveV1) {
	if !h.needHandle(event) {
		return
	}
	session, isNew := h.getOrCreateSession(event)
	if isNew {
		go session.StartConversation()
	}
	session.Inputs <- event
}

func (h *MessageHandler) getOrCreateSession(event *larkim.P2MessageReceiveV1) (*Session, bool) {
	sid := sessionId(event)
	ss, exist := h.sessions[sid]
	if exist {
		return ss, false
	}

	ss = &Session{
		Id:             sid,
		Inputs:         make(chan *larkim.P2MessageReceiveV1, 1000),
		conversationID: "",
		lastMessageID:  "",
		h:              h,
	}
	h.locker.Lock()
	defer h.locker.Unlock()
	h.sessions[sid] = ss
	return ss, true
}

func (h *MessageHandler) needHandle(event *larkim.P2MessageReceiveV1) bool {
	switch *event.Event.Message.ChatType {
	case "p2p":
		return true
	case "group":
		if len(event.Event.Message.Mentions) != 1 {
			return false
		}
		return *event.Event.Message.Mentions[0].Name == h.BotName
	default:
		return false
	}
}

type Session struct {
	Id             string
	Inputs         chan *larkim.P2MessageReceiveV1
	conversationID string
	lastMessageID  string
	h              *MessageHandler
}

func (s *Session) StartConversation() {
	for {
		event := <-s.Inputs
		<-time.After(time.Second * 1)
		textContent, err := getTextContent(event.Event)
		if err != nil {
			log.Printf("extract text failed %v", err.Error())
			s.Send(viper.GetString(MessageExtractFailed), event)
			continue
		}
		chatResponse, err := s.h.client.Ask(textContent, s.conversationID, s.lastMessageID)
		if err != nil {
			s.Send(viper.GetString(MessageUpstreamFailed), event)
			log.Printf("get reply failed %v", err.Error())
			continue
		}
		respJson, _ := json.Marshal(chatResponse)
		log.Println("anwser >>: ", string(respJson), err)
		if chatResponse.Error != "" {
			s.Send(viper.GetString(MessageUpstreamFailed), event)
			continue
		}
		s.lastMessageID = chatResponse.ResponseId
		s.conversationID = chatResponse.ConversationId
		reply := s.h.filter.Filter(chatResponse.Content)

		s.Send(reply, event)
	}
}

func (s *Session) Send(text string, event *larkim.P2MessageReceiveV1) {
	req := wrapMessage(text, event)
	createResp, err := s.h.lark.Im.Message.Reply(context.Background(), req)
	if err != nil {
		log.Printf("send reply failed %v; %v", err.Error(), createResp.Error())
	}
}

type Text struct {
	Text string `json:"text,omitempty"`
}

func (t *Text) GetText() string {
	s := strings.ReplaceAll(t.Text, "@_user_1", "")
	return strings.TrimSpace(s)
}

func getTextContent(event *larkim.P2MessageReceiveV1Data) (string, error) {
	text := &Text{}
	err := json.Unmarshal([]byte(*event.Message.Content), text)
	if err != nil {
		return "", err
	}
	return text.GetText(), nil
}

func sessionId(event *larkim.P2MessageReceiveV1) string {
	return fmt.Sprintf("%s_%s", *event.Event.Message.ChatId, *event.Event.Sender.SenderId.UnionId)
}

func wrapMessage(message string, event *larkim.P2MessageReceiveV1) *larkim.ReplyMessageReq {
	var (
		text string
	)
	if *event.Event.Message.ChatType == "group" {
		uid := *event.Event.Sender.SenderId.UnionId
		text = fmt.Sprintf("<at user_id=\"%s\">Tom </at> %s", uid, message)
	} else {
		text = message
	}
	tmp := struct {
		Text string `json:"text"`
	}{
		Text: text,
	}
	textContent, err := json.Marshal(tmp)
	if err != nil {
		log.Printf("JSON Marshal message error: %v\n", err)
	}
	return larkim.NewReplyMessageReqBuilder().MessageId(
		*event.Event.Message.MessageId,
	).Body(
		larkim.NewReplyMessageReqBodyBuilder().MsgType(
			"text",
		).Uuid(
			uuid.New().String(),
		).Content(
			string(textContent),
		).Build(),
	).Build()
}
