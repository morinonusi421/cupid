package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

func main() {
	// .envファイルを読み込む
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	channelToken := os.Getenv("LINE_CHANNEL_TOKEN")

	if channelSecret == "" || channelToken == "" {
		log.Fatal("LINE_CHANNEL_SECRET and LINE_CHANNEL_TOKEN must be set")
	}

	// LINE Messaging APIクライアントを作成
	bot, err := messaging_api.NewMessagingApiAPI(channelToken)
	if err != nil {
		log.Fatal(err)
	}

	// ヘルスチェック用エンドポイント
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Cupid LINE Bot is running")
	})

	// LINE Webhook エンドポイント
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
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
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
