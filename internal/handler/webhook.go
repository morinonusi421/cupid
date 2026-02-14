package handler

import (
	"log"
	"net/http"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"github.com/morinonusi421/cupid/internal/linebot"
	"github.com/morinonusi421/cupid/internal/message"
	"github.com/morinonusi421/cupid/internal/service"
)

// WebhookHandler はLINE Webhookを処理するハンドラー
type WebhookHandler struct {
	channelSecret string
	bot           linebot.Client
	userService   service.UserService
}

// NewWebhookHandler は WebhookHandler の新しいインスタンスを作成する
func NewWebhookHandler(
	channelSecret string,
	bot linebot.Client,
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
		case webhook.FollowEvent:
			// UserServiceで挨拶メッセージを送信
			err = h.userService.ProcessFollowEvent(r.Context(), e.ReplyToken)
			if err != nil {
				log.Println("Failed to handle follow event:", err)
			} else {
				log.Printf("Sent greeting message to new follower")
			}

		case webhook.JoinEvent:
			// グループに招待された時の挨拶メッセージを送信
			_, err = h.bot.ReplyMessage(
				&messaging_api.ReplyMessageRequest{
					ReplyToken: e.ReplyToken,
					Messages: []messaging_api.MessageInterface{
						messaging_api.TextMessage{
							Text: message.JoinGroupGreeting,
						},
					},
				},
			)
			if err != nil {
				log.Println("Failed to reply join message:", err)
			} else {
				log.Printf("Sent join message to group")
			}

		case webhook.MessageEvent:
			// テキストメッセージの場合
			switch e.Message.(type) {
			case webhook.TextMessageContent:
				// userIDを取得
				var userID string
				switch source := e.Source.(type) {
				case webhook.UserSource:
					userID = source.UserId
				default:
					log.Println("Unsupported source type")
					continue
				}

				// UserServiceで処理
				replyText, quickReplyURL, quickReplyLabel, err := h.userService.ProcessTextMessage(r.Context(), userID)
				if err != nil {
					log.Printf("Failed to process message: %v", err)
					replyText = "エラーが発生しました。もう一度試してください。"
					quickReplyURL = ""
					quickReplyLabel = ""
				}

				// LINE APIで返信
				textMessage := messaging_api.TextMessage{
					Text: replyText,
				}

				// QuickReplyがある場合は追加
				if quickReplyURL != "" && quickReplyLabel != "" {
					textMessage.QuickReply = &messaging_api.QuickReply{
						Items: []messaging_api.QuickReplyItem{
							{
								Type: "action",
								Action: &messaging_api.UriAction{
									Label: quickReplyLabel,
									Uri:   quickReplyURL,
								},
							},
						},
					}
				}

				_, err = h.bot.ReplyMessage(
					&messaging_api.ReplyMessageRequest{
						ReplyToken: e.ReplyToken,
						Messages: []messaging_api.MessageInterface{
							textMessage,
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
