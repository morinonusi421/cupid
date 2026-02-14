package service

import "errors"

// Service層で使用するカスタムエラー定義
var (
	// ErrUserNotFound はユーザーが見つからない場合のエラー
	ErrUserNotFound = errors.New("user not found")

	// ErrMatchedUserExists はマッチング中のユーザーが存在する場合のエラー
	// 注: 詳細情報が必要な場合は MatchedUserExistsError を使用すること
	ErrMatchedUserExists = errors.New("matched user exists")

	// ErrCannotRegisterYourself は自分自身を登録しようとした場合のエラー
	ErrCannotRegisterYourself = errors.New("cannot register yourself")

	// ErrDuplicateUser は重複するユーザーが存在する場合のエラー
	ErrDuplicateUser = errors.New("duplicate user")

	// ErrInvalidName は名前のバリデーションに失敗した場合のエラー
	// 注: 詳細情報が必要な場合は ValidationError を使用すること
	ErrInvalidName = errors.New("invalid name")
)

// ValidationError はバリデーションエラーの詳細情報を含む
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Is implements error comparison for errors.Is()
func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidName
}

// MatchedUserExistsError はマッチング中のユーザーが存在する場合の詳細エラー
// 相手のユーザー名を含む
type MatchedUserExistsError struct {
	MatchedUserName string
}

func (e *MatchedUserExistsError) Error() string {
	return "matched user exists"
}

// Is implements error comparison for errors.Is()
func (e *MatchedUserExistsError) Is(target error) bool {
	return target == ErrMatchedUserExists
}
