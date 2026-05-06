package model

import (
	"errors"
	"unicode"

	"github.com/aarondl/null/v8"
)

// User はユーザーのドメインモデル
type User struct {
	LineID            string
	Name              string
	Birthday          string
	CrushName         null.String // 好きな人の名前（NULL=未設定）
	CrushBirthday     null.String // 好きな人の誕生日（NULL=未設定）
	MatchedWithUserID null.String // マッチング相手のLINE ID（NULL=未マッチ）
	RegisteredAt      string
	UpdatedAt         string
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

// バリデーションエラーは「コードのみ」を表現する sentinel error。
// 文言（ユーザー向けメッセージ）は持たない。表示はクライアント側の責務。
// JSON レスポンスには err.Error() の文字列がそのまま error コードとして載る想定。
var (
	// ErrNameInvalidLength は名前が2〜20文字の範囲外
	ErrNameInvalidLength = errors.New("name_invalid_length")

	// ErrNameInvalidFormat は名前にカタカナ・長音符以外の文字が含まれる
	ErrNameInvalidFormat = errors.New("name_invalid_format")
)

// ValidateName は名前のバリデーションを行う。
// 2〜20文字の全角カタカナ（長音符可、スペース不可）であることを要求する。
//
// 失敗時は ErrNameInvalidLength または ErrNameInvalidFormat を返す。
// これらの error は「コード」であり、ユーザー向けの文言は含まない。
func ValidateName(name string) error {
	runes := []rune(name)
	if len(runes) < 2 || len(runes) > 20 {
		return ErrNameInvalidLength
	}

	for _, r := range runes {
		if !unicode.In(r, unicode.Katakana) && r != 'ー' {
			return ErrNameInvalidFormat
		}
	}

	return nil
}
