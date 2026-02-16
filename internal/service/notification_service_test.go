package service

import (
	"context"
	"errors"
	"testing"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/morinonusi421/cupid/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLineBotClient は linebot.Client の手動mock
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

// ========================================
// SendMatchNotification のテスト
// ========================================

func TestNotificationService_SendMatchNotification(t *testing.T) {
	tests := []struct {
		name             string
		toUserLineID     string
		matchedUserName  string
		mockSetup        func(*MockLineBotClient)
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:            "正常系 - マッチ通知送信成功",
			toUserLineID:    "U-alice",
			matchedUserName: "ボブ",
			mockSetup: func(m *MockLineBotClient) {
				m.On("PushMessage", mock.MatchedBy(func(req *messaging_api.PushMessageRequest) bool {
					return req.To == "U-alice" &&
						len(req.Messages) == 1 &&
						!req.NotificationDisabled
				})).Return(&messaging_api.PushMessageResponse{}, nil)
			},
			expectedError: false,
		},
		{
			name:            "異常系 - Push API呼び出し失敗",
			toUserLineID:    "U-alice",
			matchedUserName: "ボブ",
			mockSetup: func(m *MockLineBotClient) {
				m.On("PushMessage", mock.Anything).Return(nil, errors.New("api error"))
			},
			expectedError:    true,
			expectedErrorMsg: "api error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockLineBotClient)
			tt.mockSetup(mockClient)

			service := NewNotificationService(mockClient)
			err := service.SendMatchNotification(context.Background(), tt.toUserLineID, tt.matchedUserName)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// ========================================
// SendCrushRegistrationPrompt のテスト
// ========================================

func TestNotificationService_SendCrushRegistrationPrompt(t *testing.T) {
	tests := []struct {
		name             string
		toUserLineID     string
		crushLiffURL     string
		mockSetup        func(*MockLineBotClient)
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:         "正常系 - 好きな人登録促進メッセージ送信成功",
			toUserLineID: "U-alice",
			crushLiffURL: "https://liff.example.com/crush",
			mockSetup: func(m *MockLineBotClient) {
				m.On("PushMessage", mock.MatchedBy(func(req *messaging_api.PushMessageRequest) bool {
					if req.To != "U-alice" || len(req.Messages) != 1 {
						return false
					}
					textMsg, ok := req.Messages[0].(messaging_api.TextMessage)
					if !ok {
						return false
					}
					return textMsg.Text == message.UserRegistrationComplete &&
						textMsg.QuickReply != nil &&
						len(textMsg.QuickReply.Items) == 1
				})).Return(&messaging_api.PushMessageResponse{}, nil)
			},
			expectedError: false,
		},
		{
			name:         "異常系 - Push API呼び出し失敗",
			toUserLineID: "U-alice",
			crushLiffURL: "https://liff.example.com/crush",
			mockSetup: func(m *MockLineBotClient) {
				m.On("PushMessage", mock.Anything).Return(nil, errors.New("api error"))
			},
			expectedError:    true,
			expectedErrorMsg: "api error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockLineBotClient)
			tt.mockSetup(mockClient)

			service := NewNotificationService(mockClient)
			err := service.SendCrushRegistrationPrompt(context.Background(), tt.toUserLineID, tt.crushLiffURL)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// ========================================
// SendUserInfoUpdateConfirmation のテスト
// ========================================

func TestNotificationService_SendUserInfoUpdateConfirmation(t *testing.T) {
	tests := []struct {
		name             string
		toUserLineID     string
		mockSetup        func(*MockLineBotClient)
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:         "正常系 - 更新完了メッセージ送信成功",
			toUserLineID: "U-alice",
			mockSetup: func(m *MockLineBotClient) {
				m.On("PushMessage", mock.MatchedBy(func(req *messaging_api.PushMessageRequest) bool {
					if req.To != "U-alice" || len(req.Messages) != 1 {
						return false
					}
					textMsg, ok := req.Messages[0].(messaging_api.TextMessage)
					return ok && textMsg.Text == message.UserInfoUpdateConfirmation
				})).Return(&messaging_api.PushMessageResponse{}, nil)
			},
			expectedError: false,
		},
		{
			name:         "異常系 - Push API呼び出し失敗",
			toUserLineID: "U-alice",
			mockSetup: func(m *MockLineBotClient) {
				m.On("PushMessage", mock.Anything).Return(nil, errors.New("api error"))
			},
			expectedError:    true,
			expectedErrorMsg: "api error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockLineBotClient)
			tt.mockSetup(mockClient)

			service := NewNotificationService(mockClient)
			err := service.SendUserInfoUpdateConfirmation(context.Background(), tt.toUserLineID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// ========================================
// SendCrushRegistrationComplete のテスト
// ========================================

func TestNotificationService_SendCrushRegistrationComplete(t *testing.T) {
	tests := []struct {
		name              string
		toUserLineID      string
		isFirstRegistration bool
		expectedMessage   string
		mockSetup         func(*MockLineBotClient, string)
		expectedError     bool
		expectedErrorMsg  string
	}{
		{
			name:                "正常系 - 初回登録完了メッセージ",
			toUserLineID:        "U-alice",
			isFirstRegistration: true,
			expectedMessage:     message.CrushRegistrationCompleteFirst,
			mockSetup: func(m *MockLineBotClient, expectedMsg string) {
				m.On("PushMessage", mock.MatchedBy(func(req *messaging_api.PushMessageRequest) bool {
					if req.To != "U-alice" || len(req.Messages) != 1 {
						return false
					}
					textMsg, ok := req.Messages[0].(messaging_api.TextMessage)
					return ok && textMsg.Text == expectedMsg
				})).Return(&messaging_api.PushMessageResponse{}, nil)
			},
			expectedError: false,
		},
		{
			name:                "正常系 - 再登録完了メッセージ",
			toUserLineID:        "U-alice",
			isFirstRegistration: false,
			expectedMessage:     message.CrushRegistrationCompleteUpdate,
			mockSetup: func(m *MockLineBotClient, expectedMsg string) {
				m.On("PushMessage", mock.MatchedBy(func(req *messaging_api.PushMessageRequest) bool {
					if req.To != "U-alice" || len(req.Messages) != 1 {
						return false
					}
					textMsg, ok := req.Messages[0].(messaging_api.TextMessage)
					return ok && textMsg.Text == expectedMsg
				})).Return(&messaging_api.PushMessageResponse{}, nil)
			},
			expectedError: false,
		},
		{
			name:                "異常系 - Push API呼び出し失敗",
			toUserLineID:        "U-alice",
			isFirstRegistration: true,
			expectedMessage:     message.CrushRegistrationCompleteFirst,
			mockSetup: func(m *MockLineBotClient, expectedMsg string) {
				m.On("PushMessage", mock.Anything).Return(nil, errors.New("api error"))
			},
			expectedError:    true,
			expectedErrorMsg: "api error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockLineBotClient)
			tt.mockSetup(mockClient, tt.expectedMessage)

			service := NewNotificationService(mockClient)
			err := service.SendCrushRegistrationComplete(context.Background(), tt.toUserLineID, tt.isFirstRegistration)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// ========================================
// SendUnmatchNotification のテスト
// ========================================

func TestNotificationService_SendUnmatchNotification(t *testing.T) {
	tests := []struct {
		name             string
		toUserLineID     string
		partnerUserName  string
		isInitiator      bool
		mockSetup        func(*MockLineBotClient, string)
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:            "正常系 - 解除開始者へのメッセージ",
			toUserLineID:    "U-alice",
			partnerUserName: "ボブ",
			isInitiator:     true,
			mockSetup: func(m *MockLineBotClient, expectedMsg string) {
				m.On("PushMessage", mock.MatchedBy(func(req *messaging_api.PushMessageRequest) bool {
					if req.To != "U-alice" || len(req.Messages) != 1 {
						return false
					}
					textMsg, ok := req.Messages[0].(messaging_api.TextMessage)
					return ok && textMsg.Text == expectedMsg
				})).Return(&messaging_api.PushMessageResponse{}, nil)
			},
			expectedError: false,
		},
		{
			name:            "正常系 - 解除される側へのメッセージ",
			toUserLineID:    "U-bob",
			partnerUserName: "アリス",
			isInitiator:     false,
			mockSetup: func(m *MockLineBotClient, expectedMsg string) {
				m.On("PushMessage", mock.MatchedBy(func(req *messaging_api.PushMessageRequest) bool {
					if req.To != "U-bob" || len(req.Messages) != 1 {
						return false
					}
					textMsg, ok := req.Messages[0].(messaging_api.TextMessage)
					return ok && textMsg.Text == expectedMsg
				})).Return(&messaging_api.PushMessageResponse{}, nil)
			},
			expectedError: false,
		},
		{
			name:            "異常系 - Push API呼び出し失敗",
			toUserLineID:    "U-alice",
			partnerUserName: "ボブ",
			isInitiator:     true,
			mockSetup: func(m *MockLineBotClient, expectedMsg string) {
				m.On("PushMessage", mock.Anything).Return(nil, errors.New("api error"))
			},
			expectedError:    true,
			expectedErrorMsg: "api error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockLineBotClient)

			var expectedMsg string
			if tt.isInitiator {
				expectedMsg = message.UnmatchNotificationInitiator(tt.partnerUserName)
			} else {
				expectedMsg = message.UnmatchNotificationPartner(tt.partnerUserName)
			}

			tt.mockSetup(mockClient, expectedMsg)

			service := NewNotificationService(mockClient)
			err := service.SendUnmatchNotification(context.Background(), tt.toUserLineID, tt.partnerUserName, tt.isInitiator)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// ========================================
// SendFollowGreeting のテスト
// ========================================

func TestNotificationService_SendFollowGreeting(t *testing.T) {
	tests := []struct {
		name             string
		replyToken       string
		userLiffURL      string
		mockSetup        func(*MockLineBotClient)
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:        "正常系 - Follow挨拶メッセージ送信成功",
			replyToken:  "reply-token-123",
			userLiffURL: "https://liff.example.com/user",
			mockSetup: func(m *MockLineBotClient) {
				m.On("ReplyMessage", mock.MatchedBy(func(req *messaging_api.ReplyMessageRequest) bool {
					if req.ReplyToken != "reply-token-123" || len(req.Messages) != 1 {
						return false
					}
					textMsg, ok := req.Messages[0].(messaging_api.TextMessage)
					if !ok {
						return false
					}
					return textMsg.Text == message.FollowGreeting &&
						textMsg.QuickReply != nil &&
						len(textMsg.QuickReply.Items) == 1
				})).Return(&messaging_api.ReplyMessageResponse{}, nil)
			},
			expectedError: false,
		},
		{
			name:        "異常系 - Reply API呼び出し失敗",
			replyToken:  "reply-token-123",
			userLiffURL: "https://liff.example.com/user",
			mockSetup: func(m *MockLineBotClient) {
				m.On("ReplyMessage", mock.Anything).Return(nil, errors.New("api error"))
			},
			expectedError:    true,
			expectedErrorMsg: "api error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockLineBotClient)
			tt.mockSetup(mockClient)

			service := NewNotificationService(mockClient)
			err := service.SendFollowGreeting(context.Background(), tt.replyToken, tt.userLiffURL)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// ========================================
// SendJoinGroupGreeting のテスト
// ========================================

func TestNotificationService_SendJoinGroupGreeting(t *testing.T) {
	tests := []struct {
		name             string
		replyToken       string
		mockSetup        func(*MockLineBotClient)
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:       "正常系 - Join挨拶メッセージ送信成功",
			replyToken: "reply-token-456",
			mockSetup: func(m *MockLineBotClient) {
				m.On("ReplyMessage", mock.MatchedBy(func(req *messaging_api.ReplyMessageRequest) bool {
					if req.ReplyToken != "reply-token-456" || len(req.Messages) != 1 {
						return false
					}
					textMsg, ok := req.Messages[0].(messaging_api.TextMessage)
					return ok && textMsg.Text == message.JoinGroupGreeting
				})).Return(&messaging_api.ReplyMessageResponse{}, nil)
			},
			expectedError: false,
		},
		{
			name:       "異常系 - Reply API呼び出し失敗",
			replyToken: "reply-token-456",
			mockSetup: func(m *MockLineBotClient) {
				m.On("ReplyMessage", mock.Anything).Return(nil, errors.New("api error"))
			},
			expectedError:    true,
			expectedErrorMsg: "api error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockLineBotClient)
			tt.mockSetup(mockClient)

			service := NewNotificationService(mockClient)
			err := service.SendJoinGroupGreeting(context.Background(), tt.replyToken)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
