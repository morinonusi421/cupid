package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/morinonusi421/cupid/internal/handler"
	"github.com/morinonusi421/cupid/internal/liff"
	"github.com/morinonusi421/cupid/internal/linebot"
	"github.com/morinonusi421/cupid/internal/repository"
	"github.com/morinonusi421/cupid/internal/service"
	"github.com/morinonusi421/cupid/pkg/database"
)

func main() {
	// .envファイルを読み込む
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// 環境変数を読み込む
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	channelToken := os.Getenv("LINE_CHANNEL_TOKEN")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if channelSecret == "" || channelToken == "" {
		log.Fatal("LINE_CHANNEL_SECRET and LINE_CHANNEL_TOKEN must be set")
	}

	// LINE Messaging APIクライアントを作成
	botAPI, err := messaging_api.NewMessagingApiAPI(channelToken)
	if err != nil {
		log.Fatal(err)
	}

	// データベース接続
	db, err := database.InitDB("cupid.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 依存関係の組み立て (DI)
	lineBotClient := linebot.NewClient(botAPI)
	userRepo := repository.NewUserRepository(db)

	// LIFF verifier (現在は未使用だが、インターフェース互換性のためnilで渡す)
	// TODO: VerifyLIFFTokenメソッドをUserServiceインターフェースから削除したら、これも削除
	var liffVerifier *liff.Verifier
	if liffChannelID := os.Getenv("LINE_LIFF_CHANNEL_ID"); liffChannelID != "" {
		liffVerifier = liff.NewVerifier(liffChannelID)
	}

	// Web registration URL (環境変数から取得)
	// 例: https://cupid-linebot.click/liff/register.html
	registerURL := os.Getenv("REGISTER_URL")

	userService := service.NewUserService(userRepo, liffVerifier, registerURL)
	webhookHandler := handler.NewWebhookHandler(channelSecret, lineBotClient, userService)

	// Registration API handler
	registrationAPIHandler := handler.NewRegistrationAPIHandler(userService)

	// ヘルスチェック用エンドポイント
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Cupid LINE Bot is running")
	})

	// LINE Webhook エンドポイント
	http.HandleFunc("/webhook", webhookHandler.Handle)

	// Registration API endpoint
	http.HandleFunc("/api/register", registrationAPIHandler.Register)

	// サーバー起動
	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
