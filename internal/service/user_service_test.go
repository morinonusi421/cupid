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

// MockLikeRepository は LikeRepository の mock
type MockLikeRepository struct {
	mock.Mock
}

func (m *MockLikeRepository) Create(ctx context.Context, like *model.Like) error {
	args := m.Called(ctx, like)
	return args.Error(0)
}

func (m *MockLikeRepository) FindByFromUserID(ctx context.Context, fromUserID string) (*model.Like, error) {
	args := m.Called(ctx, fromUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Like), args.Error(1)
}

func (m *MockLikeRepository) FindMatchingLike(ctx context.Context, fromUserID, toName, toBirthday string) (*model.Like, error) {
	args := m.Called(ctx, fromUserID, toName, toBirthday)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Like), args.Error(1)
}

func (m *MockLikeRepository) UpdateMatched(ctx context.Context, likeID int64, matched bool) error {
	args := m.Called(ctx, likeID, matched)
	return args.Error(0)
}

// MockMatchingService は MatchingService の mock
type MockMatchingService struct {
	mock.Mock
}

func (m *MockMatchingService) CheckAndUpdateMatch(ctx context.Context, currentUser *model.User, currentLike *model.Like) (matched bool, matchedUserName string, err error) {
	args := m.Called(ctx, currentUser, currentLike)
	return args.Bool(0), args.String(1), args.Error(2)
}

func TestUserService_RegisterUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
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
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
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
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
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
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
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
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
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
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
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
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	registerURL := "https://cupid-linebot.click/liff/register.html"
	service := NewUserService(mockRepo, mockLikeRepo, nil, registerURL, mockMatchingService)
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

func TestUserService_ProcessTextMessage_Step1_CrushRegistration(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
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
	assert.Contains(t, replyText, "次に、好きな人を登録してください")
	assert.Contains(t, replyText, "https://cupid-linebot.click/crush/register.html?user_id=U123")
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUserService_ProcessTextMessage_Step2_CrushReregistration(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U123",
		Name:             "テスト太郎",
		Birthday:         "2000-01-15",
		RegistrationStep: 2,
	}

	mockRepo.On("FindByLineID", ctx, "U123").Return(user, nil)

	replyText, err := service.ProcessTextMessage(ctx, "U123", "こんにちは")

	assert.NoError(t, err)
	assert.Contains(t, replyText, "登録済みです")
	assert.Contains(t, replyText, "好きな人を変更する場合は")
	assert.Contains(t, replyText, "https://cupid-linebot.click/crush/register.html?user_id=U123")
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUserService_ProcessTextMessage_GetUserError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
	ctx := context.Background()

	mockRepo.On("FindByLineID", ctx, "U123").Return(nil, errors.New("db error"))

	replyText, err := service.ProcessTextMessage(ctx, "U123", "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get or create user")
	assert.Empty(t, replyText)
}

func TestUserService_RegisterFromLIFF(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U123",
		Name:             "",
		Birthday:         "",
		RegistrationStep: 0,
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U123").Return(user, nil)

	// Update が呼ばれることを期待（CompleteUserRegistration後の状態）
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U123" &&
			u.Name == "テストタロウ" &&
			u.Birthday == "2000-01-15" &&
			u.RegistrationStep == 1
	})).Return(nil)

	err := service.RegisterFromLIFF(ctx, "U123", "テストタロウ", "2000-01-15")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_RegisterFromLIFF_InvalidName(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U123",
		Name:             "",
		Birthday:         "",
		RegistrationStep: 0,
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U123").Return(user, nil)

	// 漢字を含む無効な名前で登録を試みる
	err := service.RegisterFromLIFF(ctx, "U123", "山田太郎", "2000-01-15")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid name")
	// Update は呼ばれないはず（バリデーションで弾かれる）
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUserService_RegisterCrush_NoMatch(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
	ctx := context.Background()

	currentUser := &model.User{
		LineID:           "U_A",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U_A").Return(currentUser, nil)

	// Like.Create が呼ばれることを期待
	mockLikeRepo.On("Create", ctx, mock.MatchedBy(func(like *model.Like) bool {
		return like.FromUserID == "U_A" &&
			like.ToName == "佐藤花子" &&
			like.ToBirthday == "1992-02-02"
	})).Return(nil)

	// User.Update が呼ばれることを期待（CompleteCrushRegistration後）
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_A" && u.RegistrationStep == 2
	})).Return(nil)

	// MatchingService.CheckAndUpdateMatch がマッチなしを返す
	mockMatchingService.On("CheckAndUpdateMatch", ctx, currentUser, mock.MatchedBy(func(like *model.Like) bool {
		return like.FromUserID == "U_A" &&
			like.ToName == "佐藤花子" &&
			like.ToBirthday == "1992-02-02"
	})).Return(false, "", nil)

	matched, matchedName, err := service.RegisterCrush(ctx, "U_A", "佐藤花子", "1992-02-02")

	assert.NoError(t, err)
	assert.False(t, matched)
	assert.Empty(t, matchedName)
	mockRepo.AssertExpectations(t)
	mockLikeRepo.AssertExpectations(t)
	mockMatchingService.AssertExpectations(t)
}

func TestUserService_RegisterCrush_SelfRegistrationError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U_SELF",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U_SELF").Return(user, nil)

	// 自分自身を登録しようとする
	_, _, err := service.RegisterCrush(ctx, "U_SELF", "山田太郎", "1990-01-01")

	assert.Error(t, err)
	assert.Equal(t, "cannot register yourself", err.Error())
	mockRepo.AssertExpectations(t)
	// Like.Create や MatchingService は呼ばれないはず
	mockLikeRepo.AssertNotCalled(t, "Create")
	mockMatchingService.AssertNotCalled(t, "CheckAndUpdateMatch")
}

func TestUserService_RegisterCrush_Matched(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockLikeRepo := new(MockLikeRepository)
	mockMatchingService := new(MockMatchingService)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "", mockMatchingService)
	ctx := context.Background()

	currentUser := &model.User{
		LineID:           "U_B",
		Name:             "佐藤花子",
		Birthday:         "1992-02-02",
		RegistrationStep: 1,
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U_B").Return(currentUser, nil)

	// Like.Create が呼ばれることを期待
	mockLikeRepo.On("Create", ctx, mock.MatchedBy(func(like *model.Like) bool {
		return like.FromUserID == "U_B" &&
			like.ToName == "山田太郎" &&
			like.ToBirthday == "1990-01-01"
	})).Return(nil)

	// User.Update が呼ばれることを期待（CompleteCrushRegistration後）
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_B" && u.RegistrationStep == 2
	})).Return(nil)

	// MatchingService.CheckAndUpdateMatch がマッチを返す
	mockMatchingService.On("CheckAndUpdateMatch", ctx, currentUser, mock.MatchedBy(func(like *model.Like) bool {
		return like.FromUserID == "U_B" &&
			like.ToName == "山田太郎" &&
			like.ToBirthday == "1990-01-01"
	})).Return(true, "山田太郎", nil)

	matched, matchedName, err := service.RegisterCrush(ctx, "U_B", "山田太郎", "1990-01-01")

	assert.NoError(t, err)
	assert.True(t, matched)
	assert.Equal(t, "山田太郎", matchedName)
	mockRepo.AssertExpectations(t)
	mockLikeRepo.AssertExpectations(t)
	mockMatchingService.AssertExpectations(t)
}
