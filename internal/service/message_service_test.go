package service

import (
	"context"
	"errors"
	"testing"

	"github.com/morinonusi421/cupid/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserService は UserService の mock (message_service用)
type MockUserServiceForMessage struct {
	mock.Mock
}

func (m *MockUserServiceForMessage) RegisterUser(ctx context.Context, lineID, displayName string) error {
	args := m.Called(ctx, lineID, displayName)
	return args.Error(0)
}

func (m *MockUserServiceForMessage) GetOrCreateUser(ctx context.Context, lineID, displayName string) (*model.User, error) {
	args := m.Called(ctx, lineID, displayName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserServiceForMessage) UpdateUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func TestMessageService_ProcessTextMessage_Step0_NameInput(t *testing.T) {
	// Setup
	mockUserService := new(MockUserServiceForMessage)
	service := NewMessageService(mockUserService)
	ctx := context.Background()

	// ユーザー（step 0: 名前入力待ち）
	user := &model.User{
		LineID:           "U123",
		Name:             "",
		Birthday:         "",
		RegistrationStep: 0,
	}

	// Mock: GetOrCreateUser が既存ユーザーを返す
	mockUserService.On("GetOrCreateUser", ctx, "U123", "").Return(user, nil)

	// Mock: UpdateUser が呼ばれることを期待
	mockUserService.On("UpdateUser", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U123" && u.Name == "テスト太郎" && u.RegistrationStep == 1
	})).Return(nil)

	// Execute
	replyText, err := service.ProcessTextMessage(ctx, "U123", "テスト太郎")

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, replyText, "テスト太郎さん、よろしくね")
	assert.Contains(t, replyText, "誕生日を教えて")
	mockUserService.AssertExpectations(t)
}

func TestMessageService_ProcessTextMessage_Step0_EmptyName(t *testing.T) {
	// Setup
	mockUserService := new(MockUserServiceForMessage)
	service := NewMessageService(mockUserService)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U123",
		Name:             "",
		RegistrationStep: 0,
	}

	mockUserService.On("GetOrCreateUser", ctx, "U123", "").Return(user, nil)

	// Execute
	replyText, err := service.ProcessTextMessage(ctx, "U123", "  ")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "名前を入力してください。", replyText)
	mockUserService.AssertNotCalled(t, "UpdateUser")
}

func TestMessageService_ProcessTextMessage_Step1_BirthdayInput(t *testing.T) {
	// Setup
	mockUserService := new(MockUserServiceForMessage)
	service := NewMessageService(mockUserService)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U123",
		Name:             "テスト太郎",
		Birthday:         "",
		RegistrationStep: 1,
	}

	mockUserService.On("GetOrCreateUser", ctx, "U123", "").Return(user, nil)

	mockUserService.On("UpdateUser", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U123" && u.Birthday == "2000-01-15" && u.RegistrationStep == 2
	})).Return(nil)

	// Execute
	replyText, err := service.ProcessTextMessage(ctx, "U123", "2000-01-15")

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, replyText, "登録完了")
	mockUserService.AssertExpectations(t)
}

func TestMessageService_ProcessTextMessage_Step1_InvalidBirthday(t *testing.T) {
	// Setup
	mockUserService := new(MockUserServiceForMessage)
	service := NewMessageService(mockUserService)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U123",
		Name:             "テスト太郎",
		RegistrationStep: 1,
	}

	mockUserService.On("GetOrCreateUser", ctx, "U123", "").Return(user, nil)

	// Execute
	replyText, err := service.ProcessTextMessage(ctx, "U123", "2000/01/15")

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, replyText, "YYYY-MM-DD形式")
	mockUserService.AssertNotCalled(t, "UpdateUser")
}

func TestMessageService_ProcessTextMessage_Step2_Completed(t *testing.T) {
	// Setup
	mockUserService := new(MockUserServiceForMessage)
	service := NewMessageService(mockUserService)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U123",
		Name:             "テスト太郎",
		Birthday:         "2000-01-15",
		RegistrationStep: 2,
	}

	mockUserService.On("GetOrCreateUser", ctx, "U123", "").Return(user, nil)

	// Execute
	replyText, err := service.ProcessTextMessage(ctx, "U123", "こんにちは")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "こんにちは", replyText) // オウム返し
	mockUserService.AssertNotCalled(t, "UpdateUser")
}

func TestMessageService_ProcessTextMessage_GetUserError(t *testing.T) {
	// Setup
	mockUserService := new(MockUserServiceForMessage)
	service := NewMessageService(mockUserService)
	ctx := context.Background()

	mockUserService.On("GetOrCreateUser", ctx, "U123", "").Return(nil, errors.New("db error"))

	// Execute
	replyText, err := service.ProcessTextMessage(ctx, "U123", "test")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get or create user")
	assert.Empty(t, replyText)
}
