package handler

import (
	"log"
	"net/http"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"github.com/morinonusi421/cupid/internal/service"
)

// WebhookHandler はLINE Webhookを処理するハンドラー
type WebhookHandler struct {
	channelSecret string
	bot           *messaging_api.MessagingApiAPI
	userService   service.UserService
}

// NewWebhookHandler は WebhookHandler の新しいインスタンスを作成する
func NewWebhookHandler(
	channelSecret string,
	bot *messaging_api.MessagingApiAPI,
	userService service.UserService,
) *WebhookHandler {
	return &WebhookHandler{
		channelSecret: channelSecret,
		bot:           bot,
		userService:   userService,
	}
}

// Handle はLINE Webhookのリクエストを処理する
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Webhookイベントをパース
	callbackRequest, err := webhook.ParseRequest(h.channelSecret, r)
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
				// オウム返し（後でユーザー登録フローに変更予定）
				replyText := message.Text

				_, err = h.bot.ReplyMessage(
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
