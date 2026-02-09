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
	liffChannelID := os.Getenv("LINE_LIFF_CHANNEL_ID")
	crushLiffChannelID := os.Getenv("LINE_LIFF_CRUSH_CHANNEL_ID")
	liffURL := os.Getenv("LINE_MINIAPP_LIFF_URL")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 必須環境変数のチェック
	if channelSecret == "" || channelToken == "" {
		log.Fatal("LINE_CHANNEL_SECRET and LINE_CHANNEL_TOKEN must be set")
	}
	if liffChannelID == "" {
		log.Fatal("LINE_LIFF_CHANNEL_ID must be set")
	}
	if crushLiffChannelID == "" {
		log.Fatal("LINE_LIFF_CRUSH_CHANNEL_ID must be set")
	}
	if liffURL == "" {
		log.Fatal("LINE_MINIAPP_LIFF_URL must be set")
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
	likeRepo := repository.NewLikeRepository(db)

	// === LIFF Verifier ===
	userLiffVerifier := liff.NewVerifier(liffChannelID)
	crushLiffVerifier := liff.NewVerifier(crushLiffChannelID)

	// === Service層 ===
	lineBotClient := linebot.NewClient(botAPI)
	matchingService := service.NewMatchingService(userRepo, likeRepo)
	userService := service.NewUserService(userRepo, likeRepo, liffURL, matchingService, lineBotClient)

	// === Handler層 ===
	webhookHandler := handler.NewWebhookHandler(channelSecret, lineBotClient, userService)
	registrationAPIHandler := handler.NewRegistrationAPIHandler(userService, userLiffVerifier)
	crushRegistrationAPIHandler := handler.NewCrushRegistrationAPIHandler(userService, crushLiffVerifier)

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

	// 静的ファイル配信
	http.Handle("/crush/", http.StripPrefix("/crush/", http.FileServer(http.Dir("static/crush"))))

	// === サーバー起動 ===
	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
