package service

import (
	"context"
	"errors"
	"testing"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
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

func (m *MockUserRepository) FindMatchingUser(ctx context.Context, user *model.User) (*model.User, error) {
	// モックの実装: 既存のテストで使用されないため、nilを返す
	return nil, nil
}

// MockMatchingService は MatchingService の mock
type MockMatchingService struct {
	mock.Mock
}

func (m *MockMatchingService) CheckAndUpdateMatch(ctx context.Context, currentUser *model.User) (matched bool, matchedUser *model.User, err error) {
	args := m.Called(ctx, currentUser)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	return args.Bool(0), args.Get(1).(*model.User), args.Error(2)
}

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

// ProcessTextMessage tests

func TestUserService_ProcessTextMessage_Step0_InvalidState(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockLineBotClient)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U123",
		Name:             "",
		Birthday:         "",
		RegistrationStep: 0,
	}

	// FindByLineID が既存ユーザーを返す（異常な状態：DB登録済みなのに registration_step が 0）
	mockRepo.On("FindByLineID", ctx, "U123").Return(user, nil)

	replyText, err := service.ProcessTextMessage(ctx, "U123", "こんにちは")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid state")
	assert.Contains(t, err.Error(), "registration_step is 0")
	assert.Empty(t, replyText)
	mockRepo.AssertExpectations(t)
}

func TestUserService_ProcessTextMessage_Step1_CrushRegistration(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	crushLiffURL := "https://miniapp.line.me/2009070891-iIdvFKtI"
	service := NewUserService(mockRepo, "", crushLiffURL, mockMatchingService, mockLineBotClient)
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
	assert.Contains(t, replyText, "https://miniapp.line.me/2009070891-iIdvFKtI")
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUserService_ProcessTextMessage_Step2_CrushReregistration(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	crushLiffURL := "https://miniapp.line.me/2009070891-iIdvFKtI"
	service := NewUserService(mockRepo, "", crushLiffURL, mockMatchingService, mockLineBotClient)
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
	assert.Contains(t, replyText, "https://miniapp.line.me/2009070891-iIdvFKtI")
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUserService_ProcessTextMessage_GetUserError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockLineBotClient)
	ctx := context.Background()

	mockRepo.On("FindByLineID", ctx, "U123").Return(nil, errors.New("db error"))

	replyText, err := service.ProcessTextMessage(ctx, "U123", "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find user")
	assert.Empty(t, replyText)
}

func TestUserService_ProcessTextMessage_UnregisteredUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	userLiffURL := "https://miniapp.line.me/2009059076-kBsUXYIC"
	service := NewUserService(mockRepo, userLiffURL, "", mockMatchingService, mockLineBotClient)
	ctx := context.Background()

	// FindByLineID が nil を返す（ユーザー未登録）
	mockRepo.On("FindByLineID", ctx, "U-new-user").Return(nil, nil)

	replyText, err := service.ProcessTextMessage(ctx, "U-new-user", "こんにちは")

	assert.NoError(t, err)
	assert.Contains(t, replyText, "初めまして")
	assert.Contains(t, replyText, "下のリンクから登録してね")
	assert.Contains(t, replyText, userLiffURL)
	mockRepo.AssertExpectations(t)
}

func TestUserService_RegisterFromLIFF_NewUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	crushLiffURL := "https://miniapp.line.me/2009070891-iIdvFKtI"
	service := NewUserService(mockRepo, "", crushLiffURL, mockMatchingService, mockLineBotClient)
	ctx := context.Background()

	// FindByLineID が nil を返す（ユーザー未登録）
	mockRepo.On("FindByLineID", ctx, "U-new-user").Return(nil, nil)

	// Create が呼ばれることを期待（RegistrationStep=1で作成）
	mockRepo.On("Create", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U-new-user" &&
			u.Name == "テストタロウ" &&
			u.Birthday == "2000-01-15" &&
			u.RegistrationStep == 1
	})).Return(nil)

	// PushMessage が好きな人登録を促すメッセージで呼ばれることを期待
	mockLineBotClient.On("PushMessage", mock.MatchedBy(func(r *messaging_api.PushMessageRequest) bool {
		return r.To == "U-new-user" && len(r.Messages) == 1
	})).Return(&messaging_api.PushMessageResponse{}, nil)

	err := service.RegisterFromLIFF(ctx, "U-new-user", "テストタロウ", "2000-01-15", false)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockLineBotClient.AssertExpectations(t)
}

func TestUserService_RegisterFromLIFF_UpdateExisting(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockLineBotClient)
	ctx := context.Background()

	existingUser := &model.User{
		LineID:           "U-existing",
		Name:             "旧ナマエ",
		Birthday:         "2000-01-01",
		RegistrationStep: 1,
		RegisteredAt:     "2024-01-01 00:00:00",
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U-existing").Return(existingUser, nil)

	// Update が呼ばれることを期待（Name と Birthday が更新される）
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U-existing" &&
			u.Name == "アタラシイナマエ" &&
			u.Birthday == "2000-12-25" &&
			u.RegisteredAt == "2024-01-01 00:00:00" // RegisteredAt は保持される
	})).Return(nil)

	// PushMessage が更新完了メッセージで呼ばれることを期待
	mockLineBotClient.On("PushMessage", mock.MatchedBy(func(r *messaging_api.PushMessageRequest) bool {
		return r.To == "U-existing" && len(r.Messages) == 1
	})).Return(&messaging_api.PushMessageResponse{}, nil)

	err := service.RegisterFromLIFF(ctx, "U-existing", "アタラシイナマエ", "2000-12-25", false)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockLineBotClient.AssertExpectations(t)
}

func TestUserService_RegisterFromLIFF_InvalidName(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockLineBotClient)
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
	err := service.RegisterFromLIFF(ctx, "U123", "山田太郎", "2000-01-15", false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "名前は全角カタカナ")
	// Update は呼ばれないはず（バリデーションで弾かれる）
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUserService_RegisterCrush_NoMatch(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockLineBotClient)
	ctx := context.Background()

	currentUser := &model.User{
		LineID:           "U_A",
		Name:             "ヤマダタロウ",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U_A").Return(currentUser, nil)

	// User.Update が呼ばれることを期待（CrushName, CrushBirthday, RegistrationStep=2）
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_A" &&
			u.CrushName == "サトウハナコ" &&
			u.CrushBirthday == "1992-02-02" &&
			u.RegistrationStep == 2
	})).Return(nil)

	// MatchingService.CheckAndUpdateMatch がマッチなしを返す
	mockMatchingService.On("CheckAndUpdateMatch", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_A" &&
			u.CrushName == "サトウハナコ" &&
			u.CrushBirthday == "1992-02-02"
	})).Return(false, nil, nil)

	// PushMessage が登録完了メッセージで呼ばれることを期待
	mockLineBotClient.On("PushMessage", mock.MatchedBy(func(r *messaging_api.PushMessageRequest) bool {
		return r.To == "U_A" && len(r.Messages) == 1
	})).Return(&messaging_api.PushMessageResponse{}, nil)

	matched, matchedName, err := service.RegisterCrush(ctx, "U_A", "サトウハナコ", "1992-02-02", false)

	assert.NoError(t, err)
	assert.False(t, matched)
	assert.Empty(t, matchedName)
	mockRepo.AssertExpectations(t)
	mockMatchingService.AssertExpectations(t)
	mockLineBotClient.AssertExpectations(t)
}

func TestUserService_RegisterCrush_SelfRegistrationError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockLineBotClient)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U_SELF",
		Name:             "ヤマダタロウ",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U_SELF").Return(user, nil)

	// 自分自身を登録しようとする
	_, _, err := service.RegisterCrush(ctx, "U_SELF", "ヤマダタロウ", "1990-01-01", false)

	assert.Error(t, err)
	assert.Equal(t, "cannot register yourself", err.Error())
	mockRepo.AssertExpectations(t)
	// MatchingService は呼ばれないはず
	mockMatchingService.AssertNotCalled(t, "CheckAndUpdateMatch")
}

func TestUserService_RegisterCrush_InvalidCrushName(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockLineBotClient)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U_INVALID",
		Name:             "ヤマダタロウ",
		Birthday:         "2000-01-15",
		RegistrationStep: 1,
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U_INVALID").Return(user, nil)

	// ひらがなの無効な名前で登録を試みる
	_, _, err := service.RegisterCrush(ctx, "U_INVALID", "やまだはなこ", "1995-05-20", false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "名前は全角カタカナ")
	mockRepo.AssertExpectations(t)
	// MatchingService は呼ばれないはず（バリデーションで弾かれる）
	mockMatchingService.AssertNotCalled(t, "CheckAndUpdateMatch")
}

func TestUserService_RegisterCrush_Matched(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockLineBotClient)
	ctx := context.Background()

	currentUser := &model.User{
		LineID:           "U_B",
		Name:             "サトウハナコ",
		Birthday:         "1992-02-02",
		RegistrationStep: 1,
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U_B").Return(currentUser, nil)

	// User.Update が呼ばれることを期待（CrushName, CrushBirthday, RegistrationStep=2）
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_B" &&
			u.CrushName == "ヤマダタロウ" &&
			u.CrushBirthday == "1990-01-01" &&
			u.RegistrationStep == 2
	})).Return(nil)

	// MatchingService.CheckAndUpdateMatch がマッチを返す
	matchedUser := &model.User{
		LineID:   "U_A",
		Name:     "ヤマダタロウ",
		Birthday: "1990-01-01",
	}
	mockMatchingService.On("CheckAndUpdateMatch", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_B" &&
			u.CrushName == "ヤマダタロウ" &&
			u.CrushBirthday == "1990-01-01"
	})).Return(true, matchedUser, nil)

	// PushMessage が2回呼ばれることを期待（現在のユーザーと相手ユーザー）
	mockLineBotClient.On("PushMessage", mock.MatchedBy(func(req *messaging_api.PushMessageRequest) bool {
		return req.To == "U_B" || req.To == "U_A"
	})).Return(&messaging_api.PushMessageResponse{}, nil).Times(2)

	matched, matchedName, err := service.RegisterCrush(ctx, "U_B", "ヤマダタロウ", "1990-01-01", false)

	assert.NoError(t, err)
	assert.True(t, matched)
	assert.Equal(t, "ヤマダタロウ", matchedName)
	mockRepo.AssertExpectations(t)
	mockMatchingService.AssertExpectations(t)
	mockLineBotClient.AssertExpectations(t)
}

func TestUserService_RegisterCrush_Matched_NotificationFails(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := new(MockMatchingService)
	mockLineBotClient := new(MockLineBotClient)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockLineBotClient)
	ctx := context.Background()

	currentUser := &model.User{
		LineID:           "U_B",
		Name:             "サトウハナコ",
		Birthday:         "1992-02-02",
		RegistrationStep: 1,
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U_B").Return(currentUser, nil)

	// User.Update が呼ばれることを期待（CrushName, CrushBirthday, RegistrationStep=2）
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_B" &&
			u.CrushName == "ヤマダタロウ" &&
			u.CrushBirthday == "1990-01-01" &&
			u.RegistrationStep == 2
	})).Return(nil)

	// MatchingService.CheckAndUpdateMatch がマッチを返す
	matchedUser := &model.User{
		LineID:   "U_A",
		Name:     "ヤマダタロウ",
		Birthday: "1990-01-01",
	}
	mockMatchingService.On("CheckAndUpdateMatch", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_B" &&
			u.CrushName == "ヤマダタロウ" &&
			u.CrushBirthday == "1990-01-01"
	})).Return(true, matchedUser, nil)

	// PushMessage が2回呼ばれるがエラーを返す
	mockLineBotClient.On("PushMessage", mock.Anything).Return(nil, errors.New("notification failed")).Times(2)

	matched, matchedName, err := service.RegisterCrush(ctx, "U_B", "ヤマダタロウ", "1990-01-01", false)

	// 通知失敗してもマッチ成立は正常に返される
	assert.NoError(t, err)
	assert.True(t, matched)
	assert.Equal(t, "ヤマダタロウ", matchedName)
	mockRepo.AssertExpectations(t)
	mockMatchingService.AssertExpectations(t)
	mockLineBotClient.AssertExpectations(t)
}
