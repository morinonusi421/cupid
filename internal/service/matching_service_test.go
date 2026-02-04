package service

import (
	"context"
	"testing"

	"github.com/morinonusi421/cupid/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMatchingUserRepository は MatchingService のテスト用 UserRepository モック
type MockMatchingUserRepository struct {
	mock.Mock
}

func (m *MockMatchingUserRepository) FindByLineID(ctx context.Context, lineID string) (*model.User, error) {
	args := m.Called(ctx, lineID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockMatchingUserRepository) FindByNameAndBirthday(ctx context.Context, name, birthday string) (*model.User, error) {
	args := m.Called(ctx, name, birthday)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockMatchingUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockMatchingUserRepository) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// MockMatchingLikeRepository は MatchingService のテスト用 LikeRepository モック
type MockMatchingLikeRepository struct {
	mock.Mock
}

func (m *MockMatchingLikeRepository) Create(ctx context.Context, like *model.Like) error {
	args := m.Called(ctx, like)
	return args.Error(0)
}

func (m *MockMatchingLikeRepository) FindByFromUserID(ctx context.Context, fromUserID string) (*model.Like, error) {
	args := m.Called(ctx, fromUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Like), args.Error(1)
}

func (m *MockMatchingLikeRepository) FindMatchingLike(ctx context.Context, fromUserID, toName, toBirthday string) (*model.Like, error) {
	args := m.Called(ctx, fromUserID, toName, toBirthday)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Like), args.Error(1)
}

func (m *MockMatchingLikeRepository) UpdateMatched(ctx context.Context, id int64, matched bool) error {
	args := m.Called(ctx, id, matched)
	return args.Error(0)
}

// TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotRegistered は
// 相手がユーザーテーブルに未登録の場合のテスト
func TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotRegistered(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(MockMatchingUserRepository)
	mockLikeRepo := new(MockMatchingLikeRepository)

	// Setup
	currentUser := &model.User{
		LineID:   "user1",
		Name:     "Alice",
		Birthday: "1990-01-01",
	}
	currentLike := model.NewLike("user1", "Bob", "1995-05-05")

	// 相手（Bob）がユーザーテーブルに未登録
	mockUserRepo.On("FindByNameAndBirthday", ctx, "Bob", "1995-05-05").
		Return(nil, nil)

	// Execute
	service := NewMatchingService(mockUserRepo, mockLikeRepo)
	matched, matchedUserName, err := service.CheckAndUpdateMatch(ctx, currentUser, currentLike)

	// Assert
	assert.NoError(t, err)
	assert.False(t, matched)
	assert.Empty(t, matchedUserName)
	mockUserRepo.AssertExpectations(t)
	mockLikeRepo.AssertExpectations(t)
}

// TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotLikeBack は
// 相手がユーザー登録済みだが、相手が自分を登録していない場合のテスト
func TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotLikeBack(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(MockMatchingUserRepository)
	mockLikeRepo := new(MockMatchingLikeRepository)

	// Setup
	currentUser := &model.User{
		LineID:   "user1",
		Name:     "Alice",
		Birthday: "1990-01-01",
	}
	currentLike := model.NewLike("user1", "Bob", "1995-05-05")

	crushUser := &model.User{
		LineID:   "user2",
		Name:     "Bob",
		Birthday: "1995-05-05",
	}

	// 相手（Bob）はユーザー登録済み
	mockUserRepo.On("FindByNameAndBirthday", ctx, "Bob", "1995-05-05").
		Return(crushUser, nil)

	// しかし相手は自分（Alice）を登録していない
	mockLikeRepo.On("FindMatchingLike", ctx, "user2", "Alice", "1990-01-01").
		Return(nil, nil)

	// Execute
	service := NewMatchingService(mockUserRepo, mockLikeRepo)
	matched, matchedUserName, err := service.CheckAndUpdateMatch(ctx, currentUser, currentLike)

	// Assert
	assert.NoError(t, err)
	assert.False(t, matched)
	assert.Empty(t, matchedUserName)
	mockUserRepo.AssertExpectations(t)
	mockLikeRepo.AssertExpectations(t)
}

// TestMatchingService_CheckAndUpdateMatch_Match は
// 相互に登録しあっている場合（マッチング成立）のテスト
func TestMatchingService_CheckAndUpdateMatch_Match(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(MockMatchingUserRepository)
	mockLikeRepo := new(MockMatchingLikeRepository)

	// Setup
	currentUser := &model.User{
		LineID:   "user1",
		Name:     "Alice",
		Birthday: "1990-01-01",
	}
	currentLike := model.NewLike("user1", "Bob", "1995-05-05")
	currentLike.ID = 1

	crushUser := &model.User{
		LineID:   "user2",
		Name:     "Bob",
		Birthday: "1995-05-05",
	}

	crushLike := model.NewLike("user2", "Alice", "1990-01-01")
	crushLike.ID = 2

	// 相手（Bob）はユーザー登録済み
	mockUserRepo.On("FindByNameAndBirthday", ctx, "Bob", "1995-05-05").
		Return(crushUser, nil)

	// 相手も自分（Alice）を登録している
	mockLikeRepo.On("FindMatchingLike", ctx, "user2", "Alice", "1990-01-01").
		Return(crushLike, nil)

	// 両方の matched フラグを更新
	mockLikeRepo.On("UpdateMatched", ctx, int64(1), true).Return(nil)
	mockLikeRepo.On("UpdateMatched", ctx, int64(2), true).Return(nil)

	// Execute
	service := NewMatchingService(mockUserRepo, mockLikeRepo)
	matched, matchedUser, err := service.CheckAndUpdateMatch(ctx, currentUser, currentLike)

	// Assert
	assert.NoError(t, err)
	assert.True(t, matched)
	assert.NotNil(t, matchedUser)
	assert.Equal(t, "Bob", matchedUser.Name)
	assert.Equal(t, "user2", matchedUser.LineID)
	mockUserRepo.AssertExpectations(t)
	mockLikeRepo.AssertExpectations(t)
}
