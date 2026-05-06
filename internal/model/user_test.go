package model

import (
	"testing"

	"github.com/aarondl/null/v8"
	"github.com/stretchr/testify/assert"
)

func TestUser_IsSamePerson(t *testing.T) {
	user := User{
		Name:     "山田太郎",
		Birthday: "1990-01-01",
	}

	t.Run("同じ名前と誕生日の場合trueを返す", func(t *testing.T) {
		result := user.IsSamePerson("山田太郎", "1990-01-01")
		assert.True(t, result)
	})

	t.Run("名前が異なる場合falseを返す", func(t *testing.T) {
		result := user.IsSamePerson("田中花子", "1990-01-01")
		assert.False(t, result)
	})

	t.Run("誕生日が異なる場合falseを返す", func(t *testing.T) {
		result := user.IsSamePerson("山田太郎", "1995-05-05")
		assert.False(t, result)
	})

	t.Run("両方異なる場合falseを返す", func(t *testing.T) {
		result := user.IsSamePerson("田中花子", "1995-05-05")
		assert.False(t, result)
	})
}

func TestUser_HasCrush(t *testing.T) {
	t.Run("好きな人が登録されている場合trueを返す", func(t *testing.T) {
		user := User{
			CrushName:     null.StringFrom("タナカハナコ"),
			CrushBirthday: null.StringFrom("1990-05-05"),
		}
		assert.True(t, user.HasCrush())
	})

	t.Run("好きな人が登録されていない場合falseを返す", func(t *testing.T) {
		user := User{
			CrushName:     null.String{Valid: false},
			CrushBirthday: null.String{Valid: false},
		}
		assert.False(t, user.HasCrush())
	})

	t.Run("名前のみ登録されている場合falseを返す", func(t *testing.T) {
		user := User{
			CrushName:     null.StringFrom("タナカハナコ"),
			CrushBirthday: null.String{Valid: false},
		}
		assert.False(t, user.HasCrush())
	})

	t.Run("誕生日のみ登録されている場合falseを返す", func(t *testing.T) {
		user := User{
			CrushName:     null.String{Valid: false},
			CrushBirthday: null.StringFrom("1990-05-05"),
		}
		assert.False(t, user.HasCrush())
	})
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{name: "有効な標準的なカタカナ名前", input: "ヤマダタロウ", expectedErr: nil},
		{name: "有効な最小文字数（2文字）", input: "アイ", expectedErr: nil},
		{name: "有効な最大文字数（20文字）", input: "アイウエオカキクケコサシスセソタチツテト", expectedErr: nil},
		{name: "有効な長音符を含む名前", input: "マーサー", expectedErr: nil},

		{name: "無効な最小文字数未満（1文字）", input: "ア", expectedErr: ErrNameInvalidLength},
		{name: "無効な最大文字数超過（21文字）", input: "アイウエオカキクケコサシスセソタチツテトナ", expectedErr: ErrNameInvalidLength},
		{name: "無効な空文字", input: "", expectedErr: ErrNameInvalidLength},

		{name: "無効な漢字を含む名前", input: "山田太郎", expectedErr: ErrNameInvalidFormat},
		{name: "無効なひらがなを含む名前", input: "やまだたろう", expectedErr: ErrNameInvalidFormat},
		{name: "無効な半角カタカナを含む名前", input: "ﾔﾏﾀﾞﾀﾛｳ", expectedErr: ErrNameInvalidFormat},
		{name: "無効なスペースを含む名前", input: "ヤマダ タロウ", expectedErr: ErrNameInvalidFormat},
		{name: "無効な英字を含む名前", input: "Yamada", expectedErr: ErrNameInvalidFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.expectedErr)
			}
		})
	}
}
