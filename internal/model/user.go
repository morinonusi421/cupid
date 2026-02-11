package model

import (
	"regexp"

	"github.com/aarondl/null/v8"
)

// User はユーザーのドメインモデル
// TODO: Model層の充実化 - Validationメソッドとbusiness logicを追加
//   - IsValidName(), IsValidBirthday() などのValidation
//   - CanCompleteRegistration(), CompleteRegistration() などの状態遷移
//   - ドメインロジックをServiceから移動してModelに集約
type User struct {
	LineID             string
	Name               string
	Birthday           string
	CrushName          null.String // 好きな人の名前（NULL=未設定）
	CrushBirthday      null.String // 好きな人の誕生日（NULL=未設定）
	MatchedWithUserID  null.String // マッチング相手のLINE ID（NULL=未マッチ）
	RegisteredAt       string
	UpdatedAt          string
}

// IsSamePerson は、指定された名前と誕生日が自分と一致するかをチェックする
func (u *User) IsSamePerson(name, birthday string) bool {
	return u.Name == name && u.Birthday == birthday
}

// IsMatched は、マッチング中かどうかを返す
func (u *User) IsMatched() bool {
	return u.MatchedWithUserID.Valid
}

// HasCrush は、好きな人が登録されているかを返す
func (u *User) HasCrush() bool {
	return u.CrushName.Valid && u.CrushBirthday.Valid
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
