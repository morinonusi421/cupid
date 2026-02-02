package service

import (
	"context"
	"errors"
	"testing"

	"github.com/morinonusi421/cupid/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository は UserRepository の mock
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindByLineID(ctx context.Context, lineID string) (*model.User, error) {
	args := m.Called(ctx, lineID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByNameAndBirthday(ctx context.Context, name, birthday string) (*model.User, error) {
	// モックの実装: 既存のテストで使用されないため、nilを返す
	return nil, nil
}

func TestUserService_RegisterUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo, nil, "")
	ctx := context.Background()

	// Create が呼ばれることを期待
	mockRepo.On("Create", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U123456789" && u.Name == "Test User"
	})).Return(nil)

	err := service.RegisterUser(ctx, "U123456789", "Test User")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_RegisterUser_Error(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo, nil, "")
	ctx := context.Background()

	// Create がエラーを返すことを期待
	mockRepo.On("Create", ctx, mock.Anything).Return(errors.New("db error"))

	err := service.RegisterUser(ctx, "U123456789", "Test User")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user")
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetOrCreateUser_ExistingUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo, nil, "")
	ctx := context.Background()

	existingUser := &model.User{
		LineID:           "U123456789",
		Name:             "Existing User",
		RegistrationStep: 2,
	}

	// FindByLineID が既存ユーザーを返すことを期待
	mockRepo.On("FindByLineID", ctx, "U123456789").Return(existingUser, nil)

	user, err := service.GetOrCreateUser(ctx, "U123456789", "Test User")
	assert.NoError(t, err)
	assert.Equal(t, "Existing User", user.Name)
	assert.Equal(t, 2, user.RegistrationStep)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetOrCreateUser_NewUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo, nil, "")
	ctx := context.Background()

	newUser := &model.User{
		LineID:           "U987654321",
		Name:             "New User",
		RegistrationStep: 0,
	}

	// FindByLineID が nil を返す（ユーザーが存在しない）
	mockRepo.On("FindByLineID", ctx, "U987654321").Return(nil, nil).Once()

	// Create が呼ばれることを期待
	mockRepo.On("Create", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U987654321" && u.Name == "New User"
	})).Return(nil)

	// FindByLineID が作成したユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U987654321").Return(newUser, nil).Once()

	user, err := service.GetOrCreateUser(ctx, "U987654321", "New User")
	assert.NoError(t, err)
	assert.Equal(t, "New User", user.Name)
	assert.Equal(t, 0, user.RegistrationStep)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetOrCreateUser_FindError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo, nil, "")
	ctx := context.Background()

	// FindByLineID がエラーを返すことを期待
	mockRepo.On("FindByLineID", ctx, "U123456789").Return(nil, errors.New("db error"))

	user, err := service.GetOrCreateUser(ctx, "U123456789", "Test User")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to find user")
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetOrCreateUser_CreateError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo, nil, "")
	ctx := context.Background()

	// FindByLineID が nil を返す（ユーザーが存在しない）
	mockRepo.On("FindByLineID", ctx, "U987654321").Return(nil, nil)

	// Create がエラーを返すことを期待
	mockRepo.On("Create", ctx, mock.Anything).Return(errors.New("db error"))

	user, err := service.GetOrCreateUser(ctx, "U987654321", "New User")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to register user")
	mockRepo.AssertExpectations(t)
}

// ProcessTextMessage tests

func TestUserService_ProcessTextMessage_Step0_InitialMessage(t *testing.T) {
	mockRepo := new(MockUserRepository)
	registerURL := "https://cupid-linebot.click/liff/register.html"
	service := NewUserService(mockRepo, nil, registerURL)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U123",
		Name:             "",
		Birthday:         "",
		RegistrationStep: 0,
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U123").Return(user, nil)

	replyText, err := service.ProcessTextMessage(ctx, "U123", "こんにちは")

	assert.NoError(t, err)
	assert.Contains(t, replyText, "初めまして")
	assert.Contains(t, replyText, "下のリンクから登録してね")
	assert.Contains(t, replyText, registerURL+"?user_id=U123")
	mockRepo.AssertExpectations(t)
}

func TestUserService_ProcessTextMessage_Step1_EchoBack(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo, nil, "")
	ctx := context.Background()

	user := &model.User{
		LineID:           "U123",
		Name:             "テスト太郎",
		Birthday:         "2000-01-15",
		RegistrationStep: 1,
	}

	mockRepo.On("FindByLineID", ctx, "U123").Return(user, nil)

	replyText, err := service.ProcessTextMessage(ctx, "U123", "こんにちは")

	assert.NoError(t, err)
	assert.Equal(t, "こんにちは", replyText) // オウム返し
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUserService_ProcessTextMessage_GetUserError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo, nil, "")
	ctx := context.Background()

	mockRepo.On("FindByLineID", ctx, "U123").Return(nil, errors.New("db error"))

	replyText, err := service.ProcessTextMessage(ctx, "U123", "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get or create user")
	assert.Empty(t, replyText)
}
