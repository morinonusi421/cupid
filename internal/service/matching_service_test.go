package service

import (
	"context"
	"testing"

	"github.com/aarondl/null/v8"
	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotRegistered は
// 相手がユーザーテーブルに未登録の場合のテスト
func TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotRegistered(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := mocks.NewMockUserRepository(t)

	// Setup
	currentUser := &model.User{
		LineID:       "user1",
		Name:         "Alice",
		Birthday:     "1990-01-01",
		CrushName:    null.StringFrom("Bob"),
		CrushBirthday: null.StringFrom("1995-05-05"),
	}

	// FindMatchingUser が nil を返す（相手が未登録）
	mockUserRepo.EXPECT().
		FindMatchingUser(ctx, currentUser).
		Return(nil, nil)

	// Execute
	service := NewMatchingService(mockUserRepo)
	matched, matchedUser, err := service.CheckAndUpdateMatch(ctx, currentUser)

	// Assert
	assert.NoError(t, err)
	assert.False(t, matched)
	assert.Nil(t, matchedUser)
}

// TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotLikeBack は
// 相手がユーザー登録済みだが、相手が自分を登録していない場合のテスト
func TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotLikeBack(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := mocks.NewMockUserRepository(t)

	// Setup
	currentUser := &model.User{
		LineID:       "user1",
		Name:         "Alice",
		Birthday:     "1990-01-01",
		CrushName:    null.StringFrom("Bob"),
		CrushBirthday: null.StringFrom("1995-05-05"),
	}

	// FindMatchingUser が nil を返す（相手は自分を登録していない）
	mockUserRepo.EXPECT().
		FindMatchingUser(ctx, currentUser).
		Return(nil, nil)

	// Execute
	service := NewMatchingService(mockUserRepo)
	matched, matchedUser, err := service.CheckAndUpdateMatch(ctx, currentUser)

	// Assert
	assert.NoError(t, err)
	assert.False(t, matched)
	assert.Nil(t, matchedUser)
}

// TestMatchingService_CheckAndUpdateMatch_Match は
// 相互に登録しあっている場合（マッチング成立）のテスト
func TestMatchingService_CheckAndUpdateMatch_Match(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := mocks.NewMockUserRepository(t)

	// Setup
	currentUser := &model.User{
		LineID:       "user1",
		Name:         "Alice",
		Birthday:     "1990-01-01",
		CrushName:    null.StringFrom("Bob"),
		CrushBirthday: null.StringFrom("1995-05-05"),
	}

	matchedUser := &model.User{
		LineID:       "user2",
		Name:         "Bob",
		Birthday:     "1995-05-05",
		CrushName:    null.StringFrom("Alice"),
		CrushBirthday: null.StringFrom("1990-01-01"),
	}

	// FindMatchingUser がマッチング相手を返す
	mockUserRepo.EXPECT().
		FindMatchingUser(ctx, currentUser).
		Return(matchedUser, nil)

	// 両方の matched_with_user_id を更新
	mockUserRepo.EXPECT().
		Update(ctx, mock.MatchedBy(func(u *model.User) bool {
			return (u.LineID == "user1" && u.MatchedWithUserID.String == "user2") ||
				(u.LineID == "user2" && u.MatchedWithUserID.String == "user1")
		})).
		Return(nil).
		Times(2)

	// Execute
	service := NewMatchingService(mockUserRepo)
	matched, resultUser, err := service.CheckAndUpdateMatch(ctx, currentUser)

	// Assert
	assert.NoError(t, err)
	assert.True(t, matched)
	assert.NotNil(t, resultUser)
	assert.Equal(t, "Bob", resultUser.Name)
	assert.Equal(t, "user2", resultUser.LineID)
}
