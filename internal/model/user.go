package model

import "regexp"

// User はユーザーのドメインモデル
// TODO: Model層の充実化 - Validationメソッドとbusiness logicを追加
//   - IsValidName(), IsValidBirthday() などのValidation
//   - CanCompleteRegistration(), CompleteRegistration() などの状態遷移
//   - ドメインロジックをServiceから移動してModelに集約
type User struct {
	LineID           string
	Name             string
	Birthday         string
	RegistrationStep int // 0: 未登録, 1: 登録完了
	RegisteredAt     string
	UpdatedAt        string
}

// IsSamePerson は、指定された名前と誕生日が自分と一致するかをチェックする
func (u *User) IsSamePerson(name, birthday string) bool {
	return u.Name == name && u.Birthday == birthday
}

// CanRegisterCrush は、Crush登録が可能かをチェックする
// ユーザー登録が完了している（RegistrationStep >= 1）必要がある
func (u *User) CanRegisterCrush() bool {
	return u.RegistrationStep >= 1
}

// CompleteCrushRegistration は、Crush登録を完了する
func (u *User) CompleteCrushRegistration() {
	u.RegistrationStep = 2
}

// IsRegistrationComplete は、ユーザー登録が完了しているかをチェックする
func (u *User) IsRegistrationComplete() bool {
	return u.RegistrationStep >= 1
}

// CompleteUserRegistration は、ユーザー登録を完了する
func (u *User) CompleteUserRegistration() {
	u.RegistrationStep = 1
}

// IsValidName は名前が有効なカタカナ文字列かをチェックする
// 2〜20文字の全角カタカナ（スペース不可）であること
// 返り値: (有効かどうか, エラーメッセージ)
func IsValidName(name string) (bool, string) {
	runes := []rune(name)
	length := len(runes)

	// 長さチェック: 2〜20文字
	if length < 2 || length > 20 {
		return false, "名前は2〜20文字で入力してください"
	}

	// カタカナチェック: 全角カタカナのみ（長音符を含む）
	katakanaPattern := regexp.MustCompile(`^[ァ-ヴー]+$`)
	if !katakanaPattern.MatchString(name) {
		return false, "名前は全角カタカナ2〜20文字で入力してください（スペース不可）"
	}

	return true, ""
}
