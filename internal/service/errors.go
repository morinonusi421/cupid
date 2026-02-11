package service

import "errors"

// Service層で使用するカスタムエラー定義
var (
	// ErrUserNotFound はユーザーが見つからない場合のエラー
	ErrUserNotFound = errors.New("user not found")

	// ErrMatchedUserExists はマッチング中のユーザーが存在する場合のエラー
	ErrMatchedUserExists = errors.New("matched user exists")

	// ErrCannotRegisterYourself は自分自身を登録しようとした場合のエラー
	ErrCannotRegisterYourself = errors.New("cannot register yourself")

	// ErrDuplicateUser は重複するユーザーが存在する場合のエラー
	ErrDuplicateUser = errors.New("duplicate user")
)
