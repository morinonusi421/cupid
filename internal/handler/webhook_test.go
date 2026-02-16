package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	servicemocks "github.com/morinonusi421/cupid/internal/service/mocks"
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

// generateSignature はLINE Webhookの署名を生成する
func generateSignature(channelSecret, body string) string {
	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write([]byte(body))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestWebhookHandler_Handle(t *testing.T) {
	channelSecret := "test-channel-secret"

	tests := []struct {
		name               string
		webhookBodyJSON    string
		signature          string
		mockSetup          func(*MockLineBotClient, *servicemocks.MockUserService)
		expectedStatusCode int
	}{
		{
			name: "正常系 - テキストメッセージ",
			webhookBodyJSON: `{
				"destination": "U1234567890",
				"events": [{
					"type": "message",
					"replyToken": "reply-token-123",
					"source": {"type": "user", "userId": "U-test-user"},
					"timestamp": 1234567890123,
					"mode": "active",
					"message": {"type": "text", "id": "msg-id-123", "text": "こんにちは"}
				}]
			}`,
			mockSetup: func(mockBot *MockLineBotClient, mockUserService *servicemocks.MockUserService) {
				mockUserService.EXPECT().ProcessTextMessage(mock.Anything, "U-test-user").
					Return("こんにちは", "", "", nil)
				mockBot.On("ReplyMessage", mock.MatchedBy(func(r *messaging_api.ReplyMessageRequest) bool {
					return r.ReplyToken == "reply-token-123" && len(r.Messages) == 1
				})).Return(&messaging_api.ReplyMessageResponse{}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "正常系 - フォローイベント",
			webhookBodyJSON: `{
				"destination": "U1234567890",
				"events": [{
					"type": "follow",
					"replyToken": "reply-token-456",
					"source": {"type": "user", "userId": "U-new-user"},
					"timestamp": 1234567890123,
					"mode": "active"
				}]
			}`,
			mockSetup: func(mockBot *MockLineBotClient, mockUserService *servicemocks.MockUserService) {
				mockUserService.EXPECT().ProcessFollowEvent(mock.Anything, "reply-token-456").Return(nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "正常系 - グループ参加イベント",
			webhookBodyJSON: `{
				"destination": "U1234567890",
				"events": [{
					"type": "join",
					"replyToken": "reply-token-789",
					"source": {"type": "group", "groupId": "G-group-123"},
					"timestamp": 1234567890123,
					"mode": "active"
				}]
			}`,
			mockSetup: func(mockBot *MockLineBotClient, mockUserService *servicemocks.MockUserService) {
				mockUserService.EXPECT().ProcessJoinEvent(mock.Anything, "reply-token-789").Return(nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:            "異常系 - 不正な署名",
			webhookBodyJSON: `{"events":[]}`,
			signature:       "invalid-signature",
			mockSetup: func(mockBot *MockLineBotClient, mockUserService *servicemocks.MockUserService) {
				// 署名が不正なので、mockは呼ばれない
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "異常系 - 不正なJSON",
			webhookBodyJSON: `{invalid json}`,
			mockSetup: func(mockBot *MockLineBotClient, mockUserService *servicemocks.MockUserService) {
				// JSONが不正なので、mockは呼ばれない
			},
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBot := new(MockLineBotClient)
			mockUserService := servicemocks.NewMockUserService(t)
			tt.mockSetup(mockBot, mockUserService)
			handler := NewWebhookHandler(channelSecret, mockBot, mockUserService)

			bodyBytes := []byte(tt.webhookBodyJSON)
			signature := tt.signature
			if signature == "" {
				signature = generateSignature(channelSecret, tt.webhookBodyJSON)
			}

			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Line-Signature", signature)

			rr := httptest.NewRecorder()
			handler.Handle(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)
			mockBot.AssertExpectations(t)
			mockUserService.AssertExpectations(t)
		})
	}
}
