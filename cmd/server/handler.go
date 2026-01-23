package main

import (
	"log"
	"net/http"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// handleWebhook はLINE Webhookのリクエストを処理する
func handleWebhook(channelSecret string, bot *messaging_api.MessagingApiAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Webhookイベントをパース
		callbackRequest, err := webhook.ParseRequest(channelSecret, r)
		if err != nil {
			log.Println("Failed to parse request:", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 各イベントを処理
		for _, event := range callbackRequest.Events {
			switch e := event.(type) {
			case webhook.MessageEvent:
				// テキストメッセージの場合
				switch message := e.Message.(type) {
				case webhook.TextMessageContent:
					// オウム返し
					replyText := message.Text

					_, err = bot.ReplyMessage(
						&messaging_api.ReplyMessageRequest{
							ReplyToken: e.ReplyToken,
							Messages: []messaging_api.MessageInterface{
								messaging_api.TextMessage{
									Text: replyText,
								},
							},
						},
					)
					if err != nil {
						log.Println("Failed to reply message:", err)
					} else {
						log.Printf("Replied: %s", replyText)
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}
