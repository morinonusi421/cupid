package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/morinonusi421/cupid/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLineBotClient は linebot.Client の mock
type MockLineBotClient struct {
	mock.Mock
}

func (m *MockLineBotClient) ReplyMessage(request *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error) {
	args := m.Called(request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messaging_api.ReplyMessageResponse), args.Error(1)
}

func (m *MockLineBotClient) PushMessage(request *messaging_api.PushMessageRequest) (*messaging_api.PushMessageResponse, error) {
	args := m.Called(request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messaging_api.PushMessageResponse), args.Error(1)
}

// MockUserService は service.UserService の mock (webhook_test用)
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) RegisterUser(ctx context.Context, lineID, displayName string) error {
	args := m.Called(ctx, lineID, displayName)
	return args.Error(0)
}

func (m *MockUserService) GetOrCreateUser(ctx context.Context, lineID, displayName string) (*model.User, error) {
	args := m.Called(ctx, lineID, displayName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserService) VerifyLIFFToken(accessToken string) (string, error) {
	args := m.Called(accessToken)
	return args.String(0), args.Error(1)
}

func (m *MockUserService) ProcessTextMessage(ctx context.Context, userID, text string) (string, error) {
	args := m.Called(ctx, userID, text)
	return args.String(0), args.Error(1)
}

func (m *MockUserService) RegisterFromLIFF(ctx context.Context, userID, name, birthday string) error {
	args := m.Called(ctx, userID, name, birthday)
	return args.Error(0)
}

func (m *MockUserService) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string) (matched bool, matchedUserName string, err error) {
	args := m.Called(ctx, userID, crushName, crushBirthday)
	return args.Bool(0), args.String(1), args.Error(2)
}

// generateSignature はLINE Webhookの署名を生成する
func generateSignature(channelSecret, body string) string {
	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write([]byte(body))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestWebhookHandler_Handle_TextMessage(t *testing.T) {
	// Setup
	channelSecret := "test-channel-secret"
	mockBot := new(MockLineBotClient)
	mockUserService := new(MockUserService)
	handler := NewWebhookHandler(channelSecret, mockBot, mockUserService)

	// テスト用のWebhookイベント（JSONフォーマット）
	webhookBodyJSON := `{
		"destination": "U1234567890",
		"events": [
			{
				"type": "message",
				"replyToken": "reply-token-123",
				"source": {
					"type": "user",
					"userId": "U-test-user"
				},
				"timestamp": 1234567890123,
				"mode": "active",
				"message": {
					"type": "text",
					"id": "msg-id-123",
					"text": "こんにちは"
				}
			}
		]
	}`

	bodyBytes := []byte(webhookBodyJSON)
	signature := generateSignature(channelSecret, webhookBodyJSON)

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Line-Signature", signature)

	// Mock: ProcessTextMessage が呼ばれて返信テキストを返すことを期待
	mockUserService.On("ProcessTextMessage", mock.Anything, "U-test-user", "こんにちは").
		Return("こんにちは", nil)

	// Mock: ReplyMessage が呼ばれることを期待
	mockBot.On("ReplyMessage", mock.MatchedBy(func(r *messaging_api.ReplyMessageRequest) bool {
		return r.ReplyToken == "reply-token-123" &&
			len(r.Messages) == 1
	})).Return(&messaging_api.ReplyMessageResponse{}, nil)

	// Execute
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	mockBot.AssertExpectations(t)
	mockUserService.AssertExpectations(t)
}

func TestWebhookHandler_Handle_InvalidSignature(t *testing.T) {
	// Setup
	channelSecret := "test-channel-secret"
	mockBot := new(MockLineBotClient)
	mockUserService := new(MockUserService)
	handler := NewWebhookHandler(channelSecret, mockBot, mockUserService)

	// 不正な署名でリクエスト
	bodyBytes := []byte(`{"events":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Line-Signature", "invalid-signature")

	// Execute
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	// Assert: 署名が不正なので BadRequest が返る
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockBot.AssertNotCalled(t, "ReplyMessage")
	mockUserService.AssertNotCalled(t, "ProcessTextMessage")
}
