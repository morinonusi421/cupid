package handler

import (
	"log"
	"net/http"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"github.com/morinonusi421/cupid/internal/linebot"
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
			// 友達登録時の挨拶メッセージ（クイックリプライ付き）
			greetingText := "友達追加ありがとう！\nCupidは相思相愛を見つけるお手伝いをするよ。\n\nまずは下のボタンから登録してね。"

			// LINE APIで返信（クイックリプライで登録を促す）
			_, err = h.bot.ReplyMessage(
				&messaging_api.ReplyMessageRequest{
					ReplyToken: e.ReplyToken,
					Messages: []messaging_api.MessageInterface{
						messaging_api.TextMessage{
							Text: greetingText,
							QuickReply: &messaging_api.QuickReply{
								Items: []messaging_api.QuickReplyItem{
									{
										Type: "action",
										Action: &messaging_api.UriAction{
											Label: "登録する",
											Uri:   "https://miniapp.line.me/2009059074-aX6pc41R",
										},
									},
								},
							},
						},
					},
				},
			)
			if err != nil {
				log.Println("Failed to send greeting message:", err)
			} else {
				log.Printf("Sent greeting message to new follower")
			}

		case webhook.MessageEvent:
			// テキストメッセージの場合
			switch message := e.Message.(type) {
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
				replyText, err := h.userService.ProcessTextMessage(r.Context(), userID, message.Text)
				if err != nil {
					log.Printf("Failed to process message: %v", err)
					replyText = "エラーが発生しました。もう一度試してください。"
				}

				// LINE APIで返信
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
