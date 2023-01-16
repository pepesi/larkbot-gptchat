package main

import (
	"log"
	"net/http"

	"github.com/spf13/viper"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/core/httpserverext"
	larkevent "github.com/larksuite/oapi-sdk-go/v3/event"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/pepesi/larkbot/handler"
)

const (
	LarkVerifyToken = "Lark.VerifyToken"
	LarkEncryptKey  = "Lark.EncryptKey"
)

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}

func main() {
	handler := handler.NewMessageHandler()
	h := dispatcher.
		NewEventDispatcher(
			viper.GetString(LarkVerifyToken),
			viper.GetString(LarkEncryptKey),
		).
		OnP2MessageReceiveV1(handler.Handle)

	logreq := func(f func(w http.ResponseWriter, req *http.Request)) func(w http.ResponseWriter, req *http.Request) {
		return func(w http.ResponseWriter, req *http.Request) {
			log.Println(req.URL.String())
			f(w, req)
		}
	}
	http.HandleFunc(
		"/webhook/event",
		logreq(
			httpserverext.NewEventHandlerFunc(
				h,
				larkevent.WithLogLevel(larkcore.LogLevelInfo),
			)),
	)

	err := http.ListenAndServe(":9999", nil)
	if err != nil {
		panic(err)
	}
}
