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

func TestLike_IsMatched(t *testing.T) {
	tests := []struct {
		name    string
		matched bool
		want    bool
	}{
		{
			name:    "Matched=trueの場合、trueを返す",
			matched: true,
			want:    true,
		},
		{
			name:    "Matched=falseの場合、falseを返す",
			matched: false,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			like := &Like{
				ID:         1,
				FromUserID: "user123",
				ToName:     "田中太郎",
				ToBirthday: "1990-01-01",
				Matched:    tt.matched,
			}

			got := like.IsMatched()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLike_MatchesCrush(t *testing.T) {
	like := &Like{
		ID:         1,
		FromUserID: "user123",
		ToName:     "田中太郎",
		ToBirthday: "1990-01-01",
		Matched:    false,
	}

	tests := []struct {
		name     string
		testName string
		birthday string
		want     bool
	}{
		{
			name:     "名前と誕生日が一致する場合、trueを返す",
			testName: "田中太郎",
			birthday: "1990-01-01",
			want:     true,
		},
		{
			name:     "名前が一致しない場合、falseを返す",
			testName: "佐藤花子",
			birthday: "1990-01-01",
			want:     false,
		},
		{
			name:     "誕生日が一致しない場合、falseを返す",
			testName: "田中太郎",
			birthday: "1991-01-01",
			want:     false,
		},
		{
			name:     "名前も誕生日も一致しない場合、falseを返す",
			testName: "佐藤花子",
			birthday: "1991-01-01",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := like.MatchesCrush(tt.testName, tt.birthday)
			assert.Equal(t, tt.want, got)
		})
	}
}
