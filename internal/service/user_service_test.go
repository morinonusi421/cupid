package service

import (
	"context"
	"errors"
	"testing"

	"github.com/aarondl/null/v8"
	"github.com/morinonusi421/cupid/internal/message"
	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/internal/service/mocks"
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

// ProcessTextMessage tests

func TestUserService_ProcessTextMessage_Step1_CrushRegistration(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	crushLiffURL := "https://miniapp.line.me/2009070891-iIdvFKtI"
	service := NewUserService(mockRepo, "", crushLiffURL, mockMatchingService, mockNotificationService)
	ctx := context.Background()

	user := &model.User{
		LineID:   "U123",
		Name:     "テスト太郎",
		Birthday: "2000-01-15",
		// CrushName未設定 = 好きな人未登録
	}

	mockRepo.On("FindByLineID", ctx, "U123").Return(user, nil)

	replyText, quickReplyURL, quickReplyLabel, err := service.ProcessTextMessage(ctx, "U123")

	assert.NoError(t, err)
	assert.Equal(t, message.RegistrationStep1Prompt, replyText)
	assert.Equal(t, crushLiffURL, quickReplyURL)
	assert.Equal(t, "好きな人を登録", quickReplyLabel)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUserService_ProcessTextMessage_Step2_CrushReregistration(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	crushLiffURL := "https://miniapp.line.me/2009070891-iIdvFKtI"
	service := NewUserService(mockRepo, "", crushLiffURL, mockMatchingService, mockNotificationService)
	ctx := context.Background()

	user := &model.User{
		LineID:        "U123",
		Name:          "テスト太郎",
		Birthday:      "2000-01-15",
		CrushName:     null.StringFrom("テスト花子"),
		CrushBirthday: null.StringFrom("2001-05-20"),
		// CrushName設定済み = 好きな人登録済み
	}

	mockRepo.On("FindByLineID", ctx, "U123").Return(user, nil)

	replyText, quickReplyURL, quickReplyLabel, err := service.ProcessTextMessage(ctx, "U123")

	assert.NoError(t, err)
	assert.Equal(t, message.AlreadyRegisteredMessage, replyText)
	assert.Equal(t, "", quickReplyURL) // 全て登録済みの場合はQuickReplyなし
	assert.Equal(t, "", quickReplyLabel)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUserService_ProcessTextMessage_GetUserError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockNotificationService)
	ctx := context.Background()

	mockRepo.On("FindByLineID", ctx, "U123").Return(nil, errors.New("db error"))

	replyText, quickReplyURL, quickReplyLabel, err := service.ProcessTextMessage(ctx, "U123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find user")
	assert.Empty(t, replyText)
	assert.Empty(t, quickReplyURL)
	assert.Empty(t, quickReplyLabel)
}

func TestUserService_ProcessTextMessage_UnregisteredUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	userLiffURL := "https://miniapp.line.me/2009059076-kBsUXYIC"
	service := NewUserService(mockRepo, userLiffURL, "", mockMatchingService, mockNotificationService)
	ctx := context.Background()

	// FindByLineID が nil を返す（ユーザー未登録）
	mockRepo.On("FindByLineID", ctx, "U-new-user").Return(nil, nil)

	replyText, quickReplyURL, quickReplyLabel, err := service.ProcessTextMessage(ctx, "U-new-user")

	assert.NoError(t, err)
	assert.Equal(t, message.UnregisteredUserPrompt, replyText)
	assert.Equal(t, userLiffURL, quickReplyURL)
	assert.Equal(t, "登録する", quickReplyLabel)
	mockRepo.AssertExpectations(t)
}

func TestUserService_RegisterFromLIFF_NewUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	crushLiffURL := "https://miniapp.line.me/2009070891-iIdvFKtI"
	service := NewUserService(mockRepo, "", crushLiffURL, mockMatchingService, mockNotificationService)
	ctx := context.Background()

	// FindByLineID が nil を返す（ユーザー未登録）
	mockRepo.On("FindByLineID", ctx, "U-new-user").Return(nil, nil)

	// Create が呼ばれることを期待
	mockRepo.On("Create", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U-new-user" &&
			u.Name == "テストタロウ" &&
			u.Birthday == "2000-01-15"
	})).Return(nil)

	// SendCrushRegistrationPrompt が呼ばれることを期待
	mockNotificationService.On("SendCrushRegistrationPrompt", ctx, "U-new-user", crushLiffURL).Return(nil)

	_, err := service.RegisterUser(ctx, "U-new-user", "テストタロウ", "2000-01-15", false)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotificationService.AssertExpectations(t)
}

func TestUserService_RegisterFromLIFF_UpdateExisting(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockNotificationService)
	ctx := context.Background()

	existingUser := &model.User{
		LineID:       "U-existing",
		Name:         "旧ナマエ",
		Birthday:     "2000-01-01",
		RegisteredAt: "2024-01-01 00:00:00",
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

	// SendUserInfoUpdateConfirmation が呼ばれることを期待
	mockNotificationService.On("SendUserInfoUpdateConfirmation", ctx, "U-existing").Return(nil)

	_, err := service.RegisterUser(ctx, "U-existing", "アタラシイナマエ", "2000-12-25", false)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotificationService.AssertExpectations(t)
}

func TestUserService_RegisterFromLIFF_InvalidName(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockNotificationService)
	ctx := context.Background()

	user := &model.User{
		LineID:   "U123",
		Name:     "",
		Birthday: "",
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U123").Return(user, nil)

	// 漢字を含む無効な名前で登録を試みる
	_, err := service.RegisterUser(ctx, "U123", "山田太郎", "2000-01-15", false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "名前は全角カタカナ")
	// Update は呼ばれないはず（バリデーションで弾かれる）
	mockRepo.AssertNotCalled(t, "Update")
}

func TestUserService_RegisterCrush_NoMatch(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockNotificationService)
	ctx := context.Background()

	currentUser := &model.User{
		LineID:   "U_A",
		Name:     "ヤマダタロウ",
		Birthday: "1990-01-01",
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U_A").Return(currentUser, nil)

	// User.Update が呼ばれることを期待（CrushName, CrushBirthday, RegistrationStep=2）
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_A" &&
			u.CrushName.String == "サトウハナコ" &&
			u.CrushBirthday.String == "1992-02-02"
	})).Return(nil)

	// MatchingService.CheckAndUpdateMatch がマッチなしを返す
	mockMatchingService.On("CheckAndUpdateMatch", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_A" &&
			u.CrushName.String == "サトウハナコ" &&
			u.CrushBirthday.String == "1992-02-02"
	})).Return(false, nil, nil)

	// SendCrushRegistrationComplete が呼ばれることを期待（初回登録）
	mockNotificationService.On("SendCrushRegistrationComplete", ctx, "U_A", true).Return(nil)

	matched, _, err := service.RegisterCrush(ctx, "U_A", "サトウハナコ", "1992-02-02", false)

	assert.NoError(t, err)
	assert.False(t, matched)
	mockRepo.AssertExpectations(t)
	mockMatchingService.AssertExpectations(t)
	mockNotificationService.AssertExpectations(t)
}

func TestUserService_RegisterCrush_SelfRegistrationError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockNotificationService)
	ctx := context.Background()

	user := &model.User{
		LineID:   "U_SELF",
		Name:     "ヤマダタロウ",
		Birthday: "1990-01-01",
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
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockNotificationService)
	ctx := context.Background()

	user := &model.User{
		LineID:   "U_INVALID",
		Name:     "ヤマダタロウ",
		Birthday: "2000-01-15",
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
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockNotificationService)
	ctx := context.Background()

	currentUser := &model.User{
		LineID:   "U_B",
		Name:     "サトウハナコ",
		Birthday: "1992-02-02",
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U_B").Return(currentUser, nil)

	// User.Update が呼ばれることを期待（CrushName, CrushBirthday）
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_B" &&
			u.CrushName.String == "ヤマダタロウ" &&
			u.CrushBirthday.String == "1990-01-01"
	})).Return(nil)

	// MatchingService.CheckAndUpdateMatch がマッチを返す
	matchedUser := &model.User{
		LineID:   "U_A",
		Name:     "ヤマダタロウ",
		Birthday: "1990-01-01",
	}
	mockMatchingService.On("CheckAndUpdateMatch", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_B" &&
			u.CrushName.String == "ヤマダタロウ" &&
			u.CrushBirthday.String == "1990-01-01"
	})).Return(true, matchedUser, nil)

	// SendMatchNotification が2回呼ばれることを期待（現在のユーザーと相手ユーザー）
	mockNotificationService.On("SendMatchNotification", ctx, "U_B", "ヤマダタロウ").Return(nil)
	mockNotificationService.On("SendMatchNotification", ctx, "U_A", "サトウハナコ").Return(nil)

	matched, _, err := service.RegisterCrush(ctx, "U_B", "ヤマダタロウ", "1990-01-01", false)

	assert.NoError(t, err)
	assert.True(t, matched)
	mockRepo.AssertExpectations(t)
	mockMatchingService.AssertExpectations(t)
	mockNotificationService.AssertExpectations(t)
}

func TestUserService_RegisterCrush_Matched_NotificationFails(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockNotificationService)
	ctx := context.Background()

	currentUser := &model.User{
		LineID:   "U_B",
		Name:     "サトウハナコ",
		Birthday: "1992-02-02",
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U_B").Return(currentUser, nil)

	// User.Update が呼ばれることを期待（CrushName, CrushBirthday）
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_B" &&
			u.CrushName.String == "ヤマダタロウ" &&
			u.CrushBirthday.String == "1990-01-01"
	})).Return(nil)

	// MatchingService.CheckAndUpdateMatch がマッチを返す
	matchedUser := &model.User{
		LineID:   "U_A",
		Name:     "ヤマダタロウ",
		Birthday: "1990-01-01",
	}
	mockMatchingService.On("CheckAndUpdateMatch", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U_B" &&
			u.CrushName.String == "ヤマダタロウ" &&
			u.CrushBirthday.String == "1990-01-01"
	})).Return(true, matchedUser, nil)

	// SendMatchNotification が2回呼ばれるがエラーを返す
	mockNotificationService.On("SendMatchNotification", ctx, "U_B", "ヤマダタロウ").Return(errors.New("notification failed"))
	mockNotificationService.On("SendMatchNotification", ctx, "U_A", "サトウハナコ").Return(errors.New("notification failed"))

	matched, _, err := service.RegisterCrush(ctx, "U_B", "ヤマダタロウ", "1990-01-01", false)

	// 通知失敗してもマッチ成立は正常に返される
	assert.NoError(t, err)
	assert.True(t, matched)
	mockRepo.AssertExpectations(t)
	mockMatchingService.AssertExpectations(t)
	mockNotificationService.AssertExpectations(t)
}

func TestUserService_RegisterFromLIFF_MatchedUserExists(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockNotificationService)
	ctx := context.Background()

	// マッチング中のユーザー
	matchedUser := &model.User{
		LineID:             "U-existing",
		Name:               "ヤマダタロウ",
		Birthday:           "2000-01-01",
		MatchedWithUserID:  null.StringFrom("U-partner"),
		CrushName:          null.StringFrom("サトウハナコ"),
		CrushBirthday:      null.StringFrom("2000-02-02"),
		RegisteredAt:       "2024-01-01 00:00:00",
	}

	// 相手のユーザー
	partnerUser := &model.User{
		LineID:             "U-partner",
		Name:               "サトウハナコ",
		Birthday:           "2000-02-02",
		MatchedWithUserID:  null.StringFrom("U-existing"),
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U-existing").Return(matchedUser, nil).Once()
	// 相手のユーザー情報を取得
	mockRepo.On("FindByLineID", ctx, "U-partner").Return(partnerUser, nil).Once()

	// confirmUnmatch=falseで更新を試みる
	_, err := service.RegisterUser(ctx, "U-existing", "アタラシイナマエ", "2000-12-25", false)

	// MatchedUserExistsError が返されることを確認
	assert.Error(t, err)
	var matchedErr *MatchedUserExistsError
	assert.ErrorAs(t, err, &matchedErr)
	assert.Equal(t, "サトウハナコ", matchedErr.MatchedUserName)

	// errors.Is でも判定できることを確認
	assert.ErrorIs(t, err, ErrMatchedUserExists)

	// Update は呼ばれないはず
	mockRepo.AssertNotCalled(t, "Update")
	mockRepo.AssertExpectations(t)
}

func TestUserService_RegisterCrush_MatchedUserExists(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockMatchingService := mocks.NewMockMatchingService(t)
	mockNotificationService := mocks.NewMockNotificationService(t)
	service := NewUserService(mockRepo, "", "", mockMatchingService, mockNotificationService)
	ctx := context.Background()

	// マッチング中のユーザー
	currentUser := &model.User{
		LineID:             "U_A",
		Name:               "ヤマダタロウ",
		Birthday:           "1990-01-01",
		MatchedWithUserID:  null.StringFrom("U_B"),
		CrushName:          null.StringFrom("サトウハナコ"),
		CrushBirthday:      null.StringFrom("1992-02-02"),
	}

	// 相手のユーザー
	partnerUser := &model.User{
		LineID:             "U_B",
		Name:               "サトウハナコ",
		Birthday:           "1992-02-02",
		MatchedWithUserID:  null.StringFrom("U_A"),
	}

	// FindByLineID が既存ユーザーを返す
	mockRepo.On("FindByLineID", ctx, "U_A").Return(currentUser, nil).Once()
	// 相手のユーザー情報を取得
	mockRepo.On("FindByLineID", ctx, "U_B").Return(partnerUser, nil).Once()

	// confirmUnmatch=falseで好きな人を変更しようとする
	_, _, err := service.RegisterCrush(ctx, "U_A", "タナカハナコ", "1995-05-20", false)

	// MatchedUserExistsError が返されることを確認
	assert.Error(t, err)
	var matchedErr *MatchedUserExistsError
	assert.ErrorAs(t, err, &matchedErr)
	assert.Equal(t, "サトウハナコ", matchedErr.MatchedUserName)

	// errors.Is でも判定できることを確認
	assert.ErrorIs(t, err, ErrMatchedUserExists)

	// Update や CheckAndUpdateMatch は呼ばれないはず
	mockRepo.AssertNotCalled(t, "Update")
	mockMatchingService.AssertNotCalled(t, "CheckAndUpdateMatch")
	mockRepo.AssertExpectations(t)
}
