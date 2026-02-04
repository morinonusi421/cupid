package model

import (
	"testing"

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

func TestUser_CanRegisterCrush(t *testing.T) {
	t.Run("登録ステップ0の場合falseを返す", func(t *testing.T) {
		user := User{RegistrationStep: 0}
		result := user.CanRegisterCrush()
		assert.False(t, result)
	})

	t.Run("登録ステップ1の場合trueを返す", func(t *testing.T) {
		user := User{RegistrationStep: 1}
		result := user.CanRegisterCrush()
		assert.True(t, result)
	})

	t.Run("登録ステップ2の場合trueを返す", func(t *testing.T) {
		user := User{RegistrationStep: 2}
		result := user.CanRegisterCrush()
		assert.True(t, result)
	})
}

func TestUser_CompleteCrushRegistration(t *testing.T) {
	t.Run("登録ステップが2に設定される", func(t *testing.T) {
		user := User{RegistrationStep: 1}
		user.CompleteCrushRegistration()
		assert.Equal(t, 2, user.RegistrationStep)
	})
}

func TestUser_IsRegistrationComplete(t *testing.T) {
	t.Run("登録ステップ0の場合falseを返す", func(t *testing.T) {
		user := User{RegistrationStep: 0}
		result := user.IsRegistrationComplete()
		assert.False(t, result)
	})

	t.Run("登録ステップ1の場合trueを返す", func(t *testing.T) {
		user := User{RegistrationStep: 1}
		result := user.IsRegistrationComplete()
		assert.True(t, result)
	})

	t.Run("登録ステップ2の場合trueを返す", func(t *testing.T) {
		user := User{RegistrationStep: 2}
		result := user.IsRegistrationComplete()
		assert.True(t, result)
	})
}

func TestUser_CompleteUserRegistration(t *testing.T) {
	t.Run("登録ステップが1に設定される", func(t *testing.T) {
		user := User{RegistrationStep: 0}
		user.CompleteUserRegistration()
		assert.Equal(t, 1, user.RegistrationStep)
	})
}

func TestIsValidName(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValid bool
		expectedError string
	}{
		{
			name:          "有効な標準的なカタカナ名前",
			input:         "ヤマダタロウ",
			expectedValid: true,
			expectedError: "",
		},
		{
			name:          "有効な最小文字数（2文字）",
			input:         "アイ",
			expectedValid: true,
			expectedError: "",
		},
		{
			name:          "有効な最大文字数（20文字）",
			input:         "アイウエオカキクケコサシスセソタチツテト",
			expectedValid: true,
			expectedError: "",
		},
		{
			name:          "有効な長音符を含む名前",
			input:         "マーサー",
			expectedValid: true,
			expectedError: "",
		},
		{
			name:          "無効な最小文字数未満（1文字）",
			input:         "ア",
			expectedValid: false,
			expectedError: "名前は2〜20文字で入力してください",
		},
		{
			name:          "無効な最大文字数超過（21文字）",
			input:         "アイウエオカキクケコサシスセソタチツテトナ",
			expectedValid: false,
			expectedError: "名前は2〜20文字で入力してください",
		},
		{
			name:          "無効な漢字を含む名前",
			input:         "山田太郎",
			expectedValid: false,
			expectedError: "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name:          "無効なひらがなを含む名前",
			input:         "やまだたろう",
			expectedValid: false,
			expectedError: "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name:          "無効な半角カタカナを含む名前",
			input:         "ﾔﾏﾀﾞﾀﾛｳ",
			expectedValid: false,
			expectedError: "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name:          "無効なスペースを含む名前",
			input:         "ヤマダ タロウ",
			expectedValid: false,
			expectedError: "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name:          "無効な英字を含む名前",
			input:         "Yamada",
			expectedValid: false,
			expectedError: "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name:          "無効な空文字",
			input:         "",
			expectedValid: false,
			expectedError: "名前は2〜20文字で入力してください",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, errMsg := IsValidName(tt.input)
			assert.Equal(t, tt.expectedValid, valid)
			assert.Equal(t, tt.expectedError, errMsg)
		})
	}
}
