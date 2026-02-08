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
	likeRepo := repository.NewLikeRepository(db)

	// LIFF verifier（トークン検証に使用）
	// ユーザー登録用
	liffChannelID := os.Getenv("LINE_LIFF_CHANNEL_ID")
	if liffChannelID == "" {
		log.Fatal("LINE_LIFF_CHANNEL_ID must be set")
	}
	userLiffVerifier := liff.NewVerifier(liffChannelID)

	// Crush registration用
	crushLiffChannelID := os.Getenv("LINE_LIFF_CRUSH_CHANNEL_ID")
	if crushLiffChannelID == "" {
		log.Fatal("LINE_LIFF_CRUSH_CHANNEL_ID must be set")
	}
	crushLiffVerifier := liff.NewVerifier(crushLiffChannelID)

	// LINEミニアプリ LIFF URL (環境変数から取得)
	// 例: https://miniapp.line.me/2009059074-aX6pc41R
	liffURL := os.Getenv("LINE_MINIAPP_LIFF_URL")
	if liffURL == "" {
		log.Fatal("LINE_MINIAPP_LIFF_URL must be set")
	}

	// Service層
	matchingService := service.NewMatchingService(userRepo, likeRepo)
	userService := service.NewUserService(userRepo, likeRepo, liffURL, matchingService, lineBotClient)
	webhookHandler := handler.NewWebhookHandler(channelSecret, lineBotClient, userService)

	// Registration API handler
	registrationAPIHandler := handler.NewRegistrationAPIHandler(userService, userLiffVerifier)
	crushRegistrationAPIHandler := handler.NewCrushRegistrationAPIHandler(userService, crushLiffVerifier)

	// ヘルスチェック用エンドポイント
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Cupid LINE Bot is running")
	})

	// LINE Webhook エンドポイント
	http.HandleFunc("/webhook", webhookHandler.Handle)

	// Registration API endpoints
	http.HandleFunc("/api/register", registrationAPIHandler.Register)
	http.HandleFunc("/api/register-crush", crushRegistrationAPIHandler.RegisterCrush)

	// 静的ファイル配信
	http.Handle("/crush/", http.StripPrefix("/crush/", http.FileServer(http.Dir("static/crush"))))

	// サーバー起動
	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
