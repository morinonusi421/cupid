package service

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	_ "modernc.org/sqlite"
)

// setupTestDB はテスト用のデータベースをセットアップする
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// テスト用の DB ファイル名
	testDBPath := "test_service_cupid.db"
	t.Cleanup(func() {
		os.Remove(testDBPath)
	})

	// DB を作成
	db, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// 外部キー制約を有効化
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		t.Fatalf("Failed to enable foreign_keys: %v", err)
	}

	// usersテーブルを作成
	usersTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		line_user_id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		birthday TEXT NOT NULL DEFAULT '',
		registration_step INTEGER NOT NULL DEFAULT 0,
		registered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(usersTableSQL); err != nil {
		db.Close()
		t.Fatalf("Failed to create users table: %v", err)
	}

	// likesテーブルを作成
	likesTableSQL := `
	CREATE TABLE IF NOT EXISTS likes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		from_user_id TEXT NOT NULL,
		to_name TEXT NOT NULL,
		to_birthday TEXT NOT NULL,
		matched INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (from_user_id) REFERENCES users(line_user_id),
		UNIQUE(from_user_id)
	);`

	if _, err := db.Exec(likesTableSQL); err != nil {
		db.Close()
		t.Fatalf("Failed to create likes table: %v", err)
	}

	return db
}

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

func TestUserService_RegisterUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockLikeRepo := new(MockLikeRepository)
	service := NewUserService(mockRepo, mockLikeRepo, nil, "")
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
	service := NewUserService(mockRepo, mockLikeRepo, nil, "")
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
	service := NewUserService(mockRepo, mockLikeRepo, nil, "")
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
	service := NewUserService(mockRepo, mockLikeRepo, nil, "")
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
	service := NewUserService(mockRepo, mockLikeRepo, nil, "")
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
	service := NewUserService(mockRepo, mockLikeRepo, nil, "")
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
	registerURL := "https://cupid-linebot.click/liff/register.html"
	service := NewUserService(mockRepo, mockLikeRepo, nil, registerURL)
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
	service := NewUserService(mockRepo, mockLikeRepo, nil, "")
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
	service := NewUserService(mockRepo, mockLikeRepo, nil, "")
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
	service := NewUserService(mockRepo, mockLikeRepo, nil, "")
	ctx := context.Background()

	mockRepo.On("FindByLineID", ctx, "U123").Return(nil, errors.New("db error"))

	replyText, err := service.ProcessTextMessage(ctx, "U123", "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get or create user")
	assert.Empty(t, replyText)
}

func TestUserService_RegisterCrush_NoMatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	likeRepo := repository.NewLikeRepository(db)

	service := NewUserService(userRepo, likeRepo, nil, "")

	// ユーザーA作成
	userA := &model.User{
		LineID:       "U_A",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), userA)

	// 好きな人を登録（相手は未登録）
	matched, matchedName, err := service.RegisterCrush(context.Background(), "U_A", "佐藤花子", "1992-02-02")
	if err != nil {
		t.Errorf("RegisterCrush failed: %v", err)
	}
	if matched {
		t.Error("Expected no match")
	}
	if matchedName != "" {
		t.Errorf("Expected empty matchedName, got %s", matchedName)
	}

	// DBに登録されたか確認
	like, _ := likeRepo.FindByFromUserID(context.Background(), "U_A")
	if like == nil {
		t.Error("Like not created")
	}
	if like.ToName != "佐藤花子" {
		t.Errorf("ToName mismatch: got %s", like.ToName)
	}
}

func TestUserService_RegisterCrush_SelfRegistrationError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	likeRepo := repository.NewLikeRepository(db)

	service := NewUserService(userRepo, likeRepo, nil, "")

	user := &model.User{
		LineID:           "U_SELF",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), user)

	// 自分自身を登録しようとする
	_, _, err := service.RegisterCrush(context.Background(), "U_SELF", "山田太郎", "1990-01-01")
	if err == nil {
		t.Error("Expected error for self-registration")
	}
	if err.Error() != "cannot register yourself" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}

func TestUserService_RegisterCrush_Matched(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	likeRepo := repository.NewLikeRepository(db)

	service := NewUserService(userRepo, likeRepo, nil, "")

	// ユーザーA作成
	userA := &model.User{
		LineID:           "U_A",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), userA)

	// ユーザーB作成
	userB := &model.User{
		LineID:           "U_B",
		Name:             "佐藤花子",
		Birthday:         "1992-02-02",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), userB)

	// A → B を登録
	service.RegisterCrush(context.Background(), "U_A", "佐藤花子", "1992-02-02")

	// B → A を登録（マッチング成立）
	matched, matchedName, err := service.RegisterCrush(context.Background(), "U_B", "山田太郎", "1990-01-01")
	if err != nil {
		t.Errorf("RegisterCrush failed: %v", err)
	}
	if !matched {
		t.Error("Expected match")
	}
	if matchedName != "山田太郎" {
		t.Errorf("matchedName mismatch: got %s, want 山田太郎", matchedName)
	}

	// 両方のmatchedフラグが1になっているか確認
	likeA, _ := likeRepo.FindByFromUserID(context.Background(), "U_A")
	if !likeA.Matched {
		t.Error("UserA's like.matched not updated")
	}

	likeB, _ := likeRepo.FindByFromUserID(context.Background(), "U_B")
	if !likeB.Matched {
		t.Error("UserB's like.matched not updated")
	}
}
