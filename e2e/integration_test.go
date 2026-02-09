package e2e

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/morinonusi421/cupid/internal/handler"
	"github.com/morinonusi421/cupid/internal/linebot"
	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/internal/repository"
	"github.com/morinonusi421/cupid/internal/service"
	"github.com/morinonusi421/cupid/pkg/database"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDBFile = "cupid_test.db"
)

var (
	channelSecret string
	channelToken  string
	registerURL   string
)

func TestMain(m *testing.M) {
	// Load .env from project root (same as production code)
	// Try current directory first, then parent directory
	if err := godotenv.Load(); err != nil {
		// If not found in current directory, try parent directory
		if err := godotenv.Load("../.env"); err != nil {
			log.Println("Warning: .env file not found (tests may be skipped)")
		}
	}

	channelSecret = os.Getenv("LINE_CHANNEL_SECRET")
	channelToken = os.Getenv("LINE_CHANNEL_TOKEN")
	registerURL = os.Getenv("REGISTER_URL")

	// Clean up test DB before and after
	os.Remove(testDBFile)
	code := m.Run()
	os.Remove(testDBFile)
	os.Exit(code)
}

// mockLineBotClient is a mock implementation for CI/CD
type mockLineBotClient struct{}

func (m *mockLineBotClient) ReplyMessage(request *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error) {
	return &messaging_api.ReplyMessageResponse{}, nil
}

func (m *mockLineBotClient) PushMessage(request *messaging_api.PushMessageRequest) (*messaging_api.PushMessageResponse, error) {
	return &messaging_api.PushMessageResponse{}, nil
}

// mockLIFFVerifier is a mock implementation for testing
type mockLIFFVerifier struct{}

func (m *mockLIFFVerifier) VerifyAccessToken(accessToken string) (string, error) {
	// Not used in current implementation
	return "", nil
}

func (m *mockLIFFVerifier) VerifyIDToken(idToken string) (string, error) {
	// Accept tokens in format "test-token-{userID}"
	// Example: "test-token-U123" returns "U123"
	if len(idToken) > 11 && idToken[:11] == "test-token-" {
		return idToken[11:], nil
	}
	return "", nil
}

func setupTestEnvironment(t *testing.T) (*handler.WebhookHandler, *handler.RegistrationAPIHandler, *handler.CrushRegistrationAPIHandler, *sql.DB) {
	// Initialize real database
	db, err := database.InitDB(testDBFile)
	require.NoError(t, err)

	// Run migrations
	migrations := &migrate.FileMigrationSource{
		Dir: "../db/migrations",
	}
	n, err := migrate.Exec(db, "sqlite3", migrations, migrate.Up)
	require.NoError(t, err)
	t.Logf("Applied %d migrations", n)

	// Initialize LINE Bot client (real or mock)
	var lineBotClient linebot.Client
	if channelToken != "" && os.Getenv("SKIP_LINE_API") != "true" {
		botAPI, err := messaging_api.NewMessagingApiAPI(channelToken)
		require.NoError(t, err)
		lineBotClient = linebot.NewClient(botAPI)
	} else {
		lineBotClient = &mockLineBotClient{}
	}

	// Initialize real repositories
	userRepo := repository.NewUserRepository(db)
	likeRepo := repository.NewLikeRepository(db)

	// Initialize mock LIFF verifier for integration tests
	// This allows us to test LIFF authentication without real LINE API calls
	mockVerifier := &mockLIFFVerifier{}

	// Initialize real services
	matchingService := service.NewMatchingService(userRepo, likeRepo)
	// Use registerURL for both user and crush LIFF URLs in tests
	userService := service.NewUserService(userRepo, likeRepo, registerURL, registerURL, matchingService, lineBotClient)

	// Initialize real handlers
	webhookHandler := handler.NewWebhookHandler(channelSecret, lineBotClient, userService)
	registrationAPIHandler := handler.NewRegistrationAPIHandler(userService, mockVerifier)
	crushRegistrationAPIHandler := handler.NewCrushRegistrationAPIHandler(userService, mockVerifier)

	return webhookHandler, registrationAPIHandler, crushRegistrationAPIHandler, db
}

func generateSignature(body []byte) string {
	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func sendWebhook(t *testing.T, handler *handler.WebhookHandler, events []interface{}) *httptest.ResponseRecorder {
	body, err := json.Marshal(map[string]interface{}{
		"events": events,
	})
	require.NoError(t, err)

	signature := generateSignature(body)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	rec := httptest.NewRecorder()
	handler.Handle(rec, req)

	return rec
}

func TestIntegration_UserRegistrationFlow(t *testing.T) {
	if channelSecret == "" {
		t.Skip("LINE_CHANNEL_SECRET not set, skipping integration test")
	}

	webhookHandler, registrationAPIHandler, _, db := setupTestEnvironment(t)
	defer db.Close()

	ctx := context.Background()
	userID := "test-user-001"

	// Step 1: Send message event via webhook
	messageEvent := map[string]interface{}{
		"type": "message",
		"source": map[string]interface{}{
			"type":   "user",
			"userId": userID,
		},
		"replyToken": "test-reply-token-001",
		"message": map[string]interface{}{
			"type": "text",
			"text": "hello",
		},
	}

	rec := sendWebhook(t, webhookHandler, []interface{}{messageEvent})
	assert.Equal(t, http.StatusOK, rec.Code)

	// Step 2: Register user via LIFF API with ID token
	registrationReq := map[string]interface{}{
		"name":     "ヤマダタロウ",
		"birthday": "1990-01-01",
	}
	body, err := json.Marshal(registrationReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token-"+userID) // Mock ID token

	rec = httptest.NewRecorder()
	registrationAPIHandler.Register(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Step 3: Verify user is saved in DB
	userRepo := repository.NewUserRepository(db)
	user, err := userRepo.FindByLineID(ctx, userID)
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "ヤマダタロウ", user.Name)
	assert.Equal(t, "1990-01-01", user.Birthday)
	assert.Equal(t, 1, user.RegistrationStep)
}

func TestIntegration_CrushRegistrationNoMatch(t *testing.T) {
	if channelSecret == "" {
		t.Skip("LINE_CHANNEL_SECRET not set, skipping integration test")
	}

	_, _, crushHandler, db := setupTestEnvironment(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test user
	userRepo := repository.NewUserRepository(db)
	user := &model.User{
		LineID:           "test-user-002",
		Name:             "タナカハナコ",
		Birthday:         "1995-05-05",
		RegistrationStep: 0,
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Update to step 1
	user, _ = userRepo.FindByLineID(ctx, "test-user-002")
	user.CompleteUserRegistration()
	err = userRepo.Update(ctx, user)
	require.NoError(t, err)

	// Register crush (no matching) with ID token
	crushReq := map[string]interface{}{
		"crush_name":     "サトウケンタ",
		"crush_birthday": "1992-03-15",
	}
	body, err := json.Marshal(crushReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/crush/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token-test-user-002") // Mock ID token

	rec := httptest.NewRecorder()
	crushHandler.RegisterCrush(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["matched"].(bool))
}

func TestIntegration_CrushRegistrationMatch(t *testing.T) {
	if channelSecret == "" {
		t.Skip("LINE_CHANNEL_SECRET not set, skipping integration test")
	}

	_, _, crushHandler, db := setupTestEnvironment(t)
	defer db.Close()

	ctx := context.Background()
	userRepo := repository.NewUserRepository(db)
	likeRepo := repository.NewLikeRepository(db)

	// Create User A
	userA := &model.User{
		LineID:           "test-user-a",
		Name:             "スズキイチロウ",
		Birthday:         "1988-08-08",
		RegistrationStep: 0,
	}
	err := userRepo.Create(ctx, userA)
	require.NoError(t, err)
	userA, _ = userRepo.FindByLineID(ctx, "test-user-a")
	userA.CompleteUserRegistration()
	err = userRepo.Update(ctx, userA)
	require.NoError(t, err)

	// Create User B
	userB := &model.User{
		LineID:           "test-user-b",
		Name:             "コバヤシミキ",
		Birthday:         "1990-12-25",
		RegistrationStep: 0,
	}
	err = userRepo.Create(ctx, userB)
	require.NoError(t, err)
	userB, _ = userRepo.FindByLineID(ctx, "test-user-b")
	userB.CompleteUserRegistration()
	err = userRepo.Update(ctx, userB)
	require.NoError(t, err)

	// User A registers User B as crush
	likeA := model.NewLike("test-user-a", "コバヤシミキ", "1990-12-25")
	err = likeRepo.Create(ctx, likeA)
	require.NoError(t, err)

	// User B registers User A as crush (should trigger match) with ID token
	crushReq := map[string]interface{}{
		"crush_name":     "スズキイチロウ",
		"crush_birthday": "1988-08-08",
	}
	body, err := json.Marshal(crushReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/crush/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token-test-user-b") // Mock ID token

	rec := httptest.NewRecorder()
	crushHandler.RegisterCrush(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["matched"].(bool))
}

func TestIntegration_ValidationError(t *testing.T) {
	if channelSecret == "" {
		t.Skip("LINE_CHANNEL_SECRET not set, skipping integration test")
	}

	_, registrationAPIHandler, _, db := setupTestEnvironment(t)
	defer db.Close()

	tests := []struct {
		name        string
		requestBody map[string]interface{}
		expectedMsg string
	}{
		{
			name: "漢字を含む名前",
			requestBody: map[string]interface{}{
				"name":     "山田太郎",
				"birthday": "1990-01-01",
			},
			expectedMsg: "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name: "ひらがなを含む名前",
			requestBody: map[string]interface{}{
				"name":     "やまだたろう",
				"birthday": "1990-01-01",
			},
			expectedMsg: "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token-test-user") // Mock ID token

			rec := httptest.NewRecorder()
			registrationAPIHandler.Register(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response["error"], tt.expectedMsg)
		})
	}
}
