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

func (m *MockMatchingUserRepository) FindMatchingUser(ctx context.Context, user *model.User) (*model.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

// TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotRegistered は
// 相手がユーザーテーブルに未登録の場合のテスト
func TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotRegistered(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(MockMatchingUserRepository)

	// Setup
	currentUser := &model.User{
		LineID:       "user1",
		Name:         "Alice",
		Birthday:     "1990-01-01",
		CrushName:    "Bob",
		CrushBirthday: "1995-05-05",
	}

	// FindMatchingUser が nil を返す（相手が未登録）
	mockUserRepo.On("FindMatchingUser", ctx, currentUser).
		Return(nil, nil)

	// Execute
	service := NewMatchingService(mockUserRepo)
	matched, matchedUser, err := service.CheckAndUpdateMatch(ctx, currentUser)

	// Assert
	assert.NoError(t, err)
	assert.False(t, matched)
	assert.Nil(t, matchedUser)
	mockUserRepo.AssertExpectations(t)
}

// TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotLikeBack は
// 相手がユーザー登録済みだが、相手が自分を登録していない場合のテスト
func TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotLikeBack(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(MockMatchingUserRepository)

	// Setup
	currentUser := &model.User{
		LineID:       "user1",
		Name:         "Alice",
		Birthday:     "1990-01-01",
		CrushName:    "Bob",
		CrushBirthday: "1995-05-05",
	}

	// FindMatchingUser が nil を返す（相手は自分を登録していない）
	mockUserRepo.On("FindMatchingUser", ctx, currentUser).
		Return(nil, nil)

	// Execute
	service := NewMatchingService(mockUserRepo)
	matched, matchedUser, err := service.CheckAndUpdateMatch(ctx, currentUser)

	// Assert
	assert.NoError(t, err)
	assert.False(t, matched)
	assert.Nil(t, matchedUser)
	mockUserRepo.AssertExpectations(t)
}

// TestMatchingService_CheckAndUpdateMatch_Match は
// 相互に登録しあっている場合（マッチング成立）のテスト
func TestMatchingService_CheckAndUpdateMatch_Match(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := new(MockMatchingUserRepository)

	// Setup
	currentUser := &model.User{
		LineID:       "user1",
		Name:         "Alice",
		Birthday:     "1990-01-01",
		CrushName:    "Bob",
		CrushBirthday: "1995-05-05",
	}

	matchedUser := &model.User{
		LineID:       "user2",
		Name:         "Bob",
		Birthday:     "1995-05-05",
		CrushName:    "Alice",
		CrushBirthday: "1990-01-01",
	}

	// FindMatchingUser がマッチング相手を返す
	mockUserRepo.On("FindMatchingUser", ctx, currentUser).
		Return(matchedUser, nil)

	// 両方の matched_with_user_id を更新
	mockUserRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
		return (u.LineID == "user1" && u.MatchedWithUserID == "user2") ||
			(u.LineID == "user2" && u.MatchedWithUserID == "user1")
	})).Return(nil).Times(2)

	// Execute
	service := NewMatchingService(mockUserRepo)
	matched, resultUser, err := service.CheckAndUpdateMatch(ctx, currentUser)

	// Assert
	assert.NoError(t, err)
	assert.True(t, matched)
	assert.NotNil(t, resultUser)
	assert.Equal(t, "Bob", resultUser.Name)
	assert.Equal(t, "user2", resultUser.LineID)
	mockUserRepo.AssertExpectations(t)
}
