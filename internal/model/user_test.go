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
