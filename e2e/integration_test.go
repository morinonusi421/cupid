package e2e

// Integration tests for Cupid LINE Bot backend.
//
// Test environment:
//   - Uses real components (Handler → Service → Repository → SQLite)
//   - LINE API: real by default, mock when SKIP_LINE_API=true
//   - Test DB: cupid_test.db (auto-cleanup before/after tests)
//
// Run tests:
//   go test ./e2e -v
//   SKIP_LINE_API=true go test ./e2e -v  # for CI/CD

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
	"github.com/morinonusi421/cupid/internal/middleware"
	"github.com/morinonusi421/cupid/internal/repository"
	"github.com/morinonusi421/cupid/internal/service"
	"github.com/morinonusi421/cupid/pkg/testutil"
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

func setupTestEnvironment(t *testing.T) (*handler.WebhookHandler, *handler.UserRegistrationAPIHandler, *handler.CrushRegistrationAPIHandler, *sql.DB) {
	// Initialize test database with schema
	db := testutil.SetupTestDB(t, testDBFile, "../db/schema.sql")

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

	// Initialize real services
	notificationService := service.NewNotificationService(lineBotClient)
	matchingService := service.NewMatchingService(userRepo)
	// Use registerURL for both user and crush LIFF URLs in tests
	userService := service.NewUserService(userRepo, registerURL, registerURL, matchingService, notificationService)

	// Initialize real handlers
	webhookHandler := handler.NewWebhookHandler(channelSecret, lineBotClient, userService)
	userRegistrationAPIHandler := handler.NewUserRegistrationAPIHandler(userService)
	crushRegistrationAPIHandler := handler.NewCrushRegistrationAPIHandler(userService, registerURL)

	return webhookHandler, userRegistrationAPIHandler, crushRegistrationAPIHandler, db
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

// registerUserViaAPI registers a user via API (true E2E)
func registerUserViaAPI(t *testing.T, handler *handler.UserRegistrationAPIHandler, userID, name, birthday string) {
	reqBody := map[string]interface{}{
		"name":     name,
		"birthday": birthday,
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/register-user", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Set user_id in context (authentication middleware would do this)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.Register(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "User registration should succeed")
}

// registerCrushViaAPI registers a crush via API (true E2E) and returns the response
func registerCrushViaAPI(t *testing.T, handler *handler.CrushRegistrationAPIHandler, userID, crushName, crushBirthday string) map[string]interface{} {
	reqBody := map[string]interface{}{
		"crush_name":     crushName,
		"crush_birthday": crushBirthday,
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/register-user-crush", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Set user_id in context (authentication middleware would do this)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.RegisterCrush(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "Crush registration should succeed")

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	return response
}

func TestIntegration_UserRegistrationFlow(t *testing.T) {
	if channelSecret == "" {
		t.Skip("LINE_CHANNEL_SECRET not set, skipping integration test")
	}

	webhookHandler, registrationAPIHandler, _, db := setupTestEnvironment(t)
	defer db.Close()

	ctx := context.Background()
	userID := "test-user-registration"

	// Step 1: Send message event via webhook
	messageEvent := map[string]interface{}{
		"type": "message",
		"source": map[string]interface{}{
			"type":   "user",
			"userId": userID,
		},
		"replyToken": "test-reply-token-registration",
		"message": map[string]interface{}{
			"type": "text",
			"text": "hello",
		},
	}

	rec := sendWebhook(t, webhookHandler, []interface{}{messageEvent})
	assert.Equal(t, http.StatusOK, rec.Code)

	// Step 2: Register user via LIFF API
	registerUserViaAPI(t, registrationAPIHandler, userID, "ヤマダタロウ", "1990-01-01")

	// Step 3: Verify user is saved in DB
	userRepo := repository.NewUserRepository(db)
	user, err := userRepo.FindByLineID(ctx, userID)
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "ヤマダタロウ", user.Name)
	assert.Equal(t, "1990-01-01", user.Birthday)
}

func TestIntegration_CrushRegistrationNoMatch(t *testing.T) {
	if channelSecret == "" {
		t.Skip("LINE_CHANNEL_SECRET not set, skipping integration test")
	}

	_, registrationAPIHandler, crushHandler, db := setupTestEnvironment(t)
	defer db.Close()

	userID := "test-user-no-match"

	// Step 1: Register user via API
	registerUserViaAPI(t, registrationAPIHandler, userID, "タナカハナコ", "1995-05-05")

	// Step 2: Register crush (no matching user exists) via API
	response := registerCrushViaAPI(t, crushHandler, userID, "サトウケンタ", "1992-03-15")

	// Step 3: Verify no match occurred
	assert.False(t, response["matched"].(bool), "Should not match when no matching user exists")
}

func TestIntegration_CrushRegistrationMatch(t *testing.T) {
	if channelSecret == "" {
		t.Skip("LINE_CHANNEL_SECRET not set, skipping integration test")
	}

	_, registrationAPIHandler, crushHandler, db := setupTestEnvironment(t)
	defer db.Close()

	ctx := context.Background()
	userRepo := repository.NewUserRepository(db)

	userAID := "test-user-match-a"
	userBID := "test-user-match-b"

	// Step 1: User A registers via API
	registerUserViaAPI(t, registrationAPIHandler, userAID, "スズキイチロウ", "1988-08-08")

	// Step 2: User A registers crush (User B) via API
	responseA := registerCrushViaAPI(t, crushHandler, userAID, "コバヤシミキ", "1990-12-25")
	assert.False(t, responseA["matched"].(bool), "User A should not match yet (User B not registered)")

	// Step 3: User B registers via API
	registerUserViaAPI(t, registrationAPIHandler, userBID, "コバヤシミキ", "1990-12-25")

	// Step 4: User B registers crush (User A) via API - should trigger match
	responseB := registerCrushViaAPI(t, crushHandler, userBID, "スズキイチロウ", "1988-08-08")
	assert.True(t, responseB["matched"].(bool), "User B should match with User A")

	// Step 5: Verify both users have matched_with_user_id set in DB
	userA, err := userRepo.FindByLineID(ctx, userAID)
	require.NoError(t, err)
	assert.True(t, userA.MatchedWithUserID.Valid, "User A should have matched_with_user_id set")
	assert.Equal(t, userBID, userA.MatchedWithUserID.String, "User A should be matched with User B")

	userB, err := userRepo.FindByLineID(ctx, userBID)
	require.NoError(t, err)
	assert.True(t, userB.MatchedWithUserID.Valid, "User B should have matched_with_user_id set")
	assert.Equal(t, userAID, userB.MatchedWithUserID.String, "User B should be matched with User A")
}

func TestIntegration_MatchedUserExistsError(t *testing.T) {
	if channelSecret == "" {
		t.Skip("LINE_CHANNEL_SECRET not set, skipping integration test")
	}

	_, registrationAPIHandler, crushHandler, db := setupTestEnvironment(t)
	defer db.Close()

	userAID := "test-user-matched-update-a"
	userBID := "test-user-matched-update-b"

	// Step 1: Create matched users
	registerUserViaAPI(t, registrationAPIHandler, userAID, "ワタナベサトシ", "1987-07-07")
	registerCrushViaAPI(t, crushHandler, userAID, "イトウアキコ", "1989-09-09")
	registerUserViaAPI(t, registrationAPIHandler, userBID, "イトウアキコ", "1989-09-09")
	responseB := registerCrushViaAPI(t, crushHandler, userBID, "ワタナベサトシ", "1987-07-07")
	assert.True(t, responseB["matched"].(bool), "Users should be matched")

	// Step 2: Try to update User A's info without confirmUnmatch
	reqBody := map[string]interface{}{
		"name":            "ワタナベマサル",
		"birthday":        "1987-07-08",
		"confirm_unmatch": false,
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/register-user", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userAID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	registrationAPIHandler.Register(rec, req)

	// Step 3: Verify error response
	assert.Equal(t, http.StatusConflict, rec.Code)
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "matched_user_exists", response["error"])
}

func TestIntegration_DuplicateUserError(t *testing.T) {
	if channelSecret == "" {
		t.Skip("LINE_CHANNEL_SECRET not set, skipping integration test")
	}

	_, registrationAPIHandler, _, db := setupTestEnvironment(t)
	defer db.Close()

	userAID := "test-user-duplicate-a"
	userBID := "test-user-duplicate-b"

	// Step 1: Register User A
	registerUserViaAPI(t, registrationAPIHandler, userAID, "タカハシユウキ", "1991-11-11")

	// Step 2: Try to register User B with same name/birthday
	reqBody := map[string]interface{}{
		"name":     "タカハシユウキ",
		"birthday": "1991-11-11",
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/register-user", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userBID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	registrationAPIHandler.Register(rec, req)

	// Step 3: Verify error response
	assert.Equal(t, http.StatusConflict, rec.Code)
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "duplicate_user", response["error"])
}

func TestIntegration_UnmatchFlow(t *testing.T) {
	if channelSecret == "" {
		t.Skip("LINE_CHANNEL_SECRET not set, skipping integration test")
	}

	_, registrationAPIHandler, crushHandler, db := setupTestEnvironment(t)
	defer db.Close()

	ctx := context.Background()
	userRepo := repository.NewUserRepository(db)

	userAID := "test-user-unmatch-a"
	userBID := "test-user-unmatch-b"

	// Step 1: Create matched users
	registerUserViaAPI(t, registrationAPIHandler, userAID, "ヨシダマサヒロ", "1986-06-06")
	registerCrushViaAPI(t, crushHandler, userAID, "サイトウユカ", "1988-08-08")
	registerUserViaAPI(t, registrationAPIHandler, userBID, "サイトウユカ", "1988-08-08")
	responseB := registerCrushViaAPI(t, crushHandler, userBID, "ヨシダマサヒロ", "1986-06-06")
	assert.True(t, responseB["matched"].(bool), "Users should be matched")

	// Step 2: Update User A's info with confirmUnmatch=true
	reqBody := map[string]interface{}{
		"name":            "ヨシダタカシ",
		"birthday":        "1986-06-07",
		"confirm_unmatch": true,
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/register-user", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	reqCtx := context.WithValue(req.Context(), middleware.UserIDKey, userAID)
	req = req.WithContext(reqCtx)

	rec := httptest.NewRecorder()
	registrationAPIHandler.Register(rec, req)

	// Step 3: Verify successful update
	assert.Equal(t, http.StatusOK, rec.Code)

	// Step 4: Verify both users are unmatched in DB
	userA, err := userRepo.FindByLineID(ctx, userAID)
	require.NoError(t, err)
	assert.False(t, userA.MatchedWithUserID.Valid, "User A should be unmatched")
	assert.Equal(t, "ヨシダタカシ", userA.Name, "User A's name should be updated")
	assert.Equal(t, "1986-06-07", userA.Birthday, "User A's birthday should be updated")

	userB, err := userRepo.FindByLineID(ctx, userBID)
	require.NoError(t, err)
	assert.False(t, userB.MatchedWithUserID.Valid, "User B should be unmatched")
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

			req := httptest.NewRequest(http.MethodPost, "/api/register-user", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Set user_id in context (authentication middleware would do this)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, "test-user")
			req = req.WithContext(ctx)

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
