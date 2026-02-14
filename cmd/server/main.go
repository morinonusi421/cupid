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

	// === 環境変数の読み込み ===
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	channelToken := os.Getenv("LINE_CHANNEL_TOKEN")
	userLiffChannelID := os.Getenv("LINE_LIFF_USER_CHANNEL_ID")
	crushLiffChannelID := os.Getenv("LINE_LIFF_CRUSH_CHANNEL_ID")
	userLiffURL := os.Getenv("LINE_LIFF_USER_URL")
	crushLiffURL := os.Getenv("LINE_LIFF_CRUSH_URL")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 必須環境変数のチェック
	if channelSecret == "" || channelToken == "" {
		log.Fatal("LINE_CHANNEL_SECRET and LINE_CHANNEL_TOKEN must be set")
	}
	if userLiffChannelID == "" {
		log.Fatal("LINE_LIFF_USER_CHANNEL_ID must be set")
	}
	if crushLiffChannelID == "" {
		log.Fatal("LINE_LIFF_CRUSH_CHANNEL_ID must be set")
	}
	if userLiffURL == "" {
		log.Fatal("LINE_LIFF_USER_URL must be set")
	}
	if crushLiffURL == "" {
		log.Fatal("LINE_LIFF_CRUSH_URL must be set")
	}

	// === 外部リソースの初期化 ===
	// LINE Messaging APIクライアント
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

	// === Repository層 ===
	userRepo := repository.NewUserRepository(db)

	// === LIFF Verifier ===
	userLiffVerifier := liff.NewVerifier(userLiffChannelID)
	crushLiffVerifier := liff.NewVerifier(crushLiffChannelID)

	// === Service層 ===
	lineBotClient := linebot.NewClient(botAPI)
	matchingService := service.NewMatchingService(userRepo)
	userService := service.NewUserService(userRepo, userLiffURL, crushLiffURL, matchingService, lineBotClient)

	// === Handler層 ===
	webhookHandler := handler.NewWebhookHandler(channelSecret, lineBotClient, userService)
	registrationAPIHandler := handler.NewRegistrationAPIHandler(userService, userLiffVerifier)
	crushRegistrationAPIHandler := handler.NewCrushRegistrationAPIHandler(userService, crushLiffVerifier, userLiffURL)

	// === ルーティング設定 ===
	// ヘルスチェック
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Cupid LINE Bot is running")
	})

	// LINE Webhook
	http.HandleFunc("/webhook", webhookHandler.Handle)

	// Registration API
	http.HandleFunc("/api/register", registrationAPIHandler.Register)
	http.HandleFunc("/api/register-crush", crushRegistrationAPIHandler.RegisterCrush)

	// 静的ファイル配信（/user/, /crush/）はNginxで直接処理されるため、ここでは設定しない
	// 詳細: nginx/cupid.conf を参照

	// === サーバー起動 ===
	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
