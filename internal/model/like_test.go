package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLike(t *testing.T) {
	// 正常系: NewLikeで新しいLikeを作成
	like := NewLike("user123", "田中太郎", "1990-01-01")

	assert.Equal(t, "user123", like.FromUserID)
	assert.Equal(t, "田中太郎", like.ToName)
	assert.Equal(t, "1990-01-01", like.ToBirthday)
	assert.False(t, like.Matched, "新規作成時はMatchedがfalseであること")
	assert.Equal(t, int64(0), like.ID, "新規作成時はIDが0であること")
}

func TestLike_MarkAsMatched(t *testing.T) {
	// 初期状態でMatchedがfalse
	like := &Like{
		ID:         1,
		FromUserID: "user123",
		ToName:     "田中太郎",
		ToBirthday: "1990-01-01",
		Matched:    false,
	}

	// MarkAsMatchedを呼び出す
	like.MarkAsMatched()

	// Matchedがtrueになること
	assert.True(t, like.Matched)
}
