package service

import "errors"

// Service層で使用するカスタムエラー定義
//
// 各 sentinel error の文字列はそのまま API レスポンスの error コードとして使われる。
// アンダースコアつなぎのマシン可読コードで統一すること。
var (
	// ErrUserNotFound はユーザーが見つからない場合のエラー
	ErrUserNotFound = errors.New("user_not_found")

	// ErrMatchedUserExists はマッチング中のユーザーが存在する場合のエラー
	// 注: 詳細情報が必要な場合は MatchedUserExistsError を使用すること
	ErrMatchedUserExists = errors.New("matched_user_exists")

	// ErrCannotRegisterYourself は自分自身を登録しようとした場合のエラー
	ErrCannotRegisterYourself = errors.New("cannot_register_yourself")

	// ErrDuplicateUser は重複するユーザーが存在する場合のエラー
	ErrDuplicateUser = errors.New("duplicate_user")
)

// MatchedUserExistsError はマッチング中のユーザーが存在する場合の詳細エラー
// 相手のユーザー名を含む
type MatchedUserExistsError struct {
	MatchedUserName string
}

func (e *MatchedUserExistsError) Error() string {
	return ErrMatchedUserExists.Error()
}

// Is implements error comparison for errors.Is()
func (e *MatchedUserExistsError) Is(target error) bool {
	return target == ErrMatchedUserExists
}
