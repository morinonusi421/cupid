package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/morinonusi421/cupid/internal/model"
)

// MessageService はメッセージ処理のビジネスロジック層のインターフェース
type MessageService interface {
	ProcessTextMessage(ctx context.Context, userID, text string) (string, error)
}

type messageService struct {
	userService UserService
}

// NewMessageService は MessageService の新しいインスタンスを作成する
func NewMessageService(userService UserService) MessageService {
	return &messageService{userService: userService}
}

// ProcessTextMessage はテキストメッセージを処理して返信テキストを決定する
func (s *messageService) ProcessTextMessage(ctx context.Context, userID, text string) (string, error) {
	// ユーザーを取得または作成
	user, err := s.userService.GetOrCreateUser(ctx, userID, "")
	if err != nil {
		return "", fmt.Errorf("failed to get or create user: %w", err)
	}

	// registration_step に応じて処理分岐
	switch user.RegistrationStep {
	case 0:
		// 初期状態 - 名前入力の案内
		return s.handleInitialMessage(ctx, user)
	case 1:
		// 名前入力待ち
		return s.handleNameInput(ctx, user, text)
	case 2:
		// 誕生日入力待ち
		return s.handleBirthdayInput(ctx, user, text)
	case 3:
		// 登録完了済み - オウム返し（後で通常機能に変更予定）
		return text, nil
	default:
		return "", fmt.Errorf("invalid registration step: %d", user.RegistrationStep)
	}
}

// handleInitialMessage は初回メッセージを処理する（名前入力の案内）
func (s *messageService) handleInitialMessage(ctx context.Context, user *model.User) (string, error) {
	// step を 1 に進める（名前入力待ち）
	user.RegistrationStep = 1

	if err := s.userService.UpdateUser(ctx, user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	return "初めまして！まずは名前を教えてね。", nil
}

// handleNameInput は名前入力を処理する
func (s *messageService) handleNameInput(ctx context.Context, user *model.User, text string) (string, error) {
	// 名前のバリデーション
	name := strings.TrimSpace(text)
	if name == "" {
		return "名前を入力してください。", nil
	}
	if len(name) > 50 {
		return "名前が長すぎます。50文字以内で入力してください。", nil
	}

	// ユーザー情報を更新
	user.Name = name
	user.RegistrationStep = 1

	if err := s.userService.UpdateUser(ctx, user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	return fmt.Sprintf("%sさん、よろしくね。\n次に、誕生日を教えて（YYYY-MM-DD形式で入力してね）", name), nil
}

// handleBirthdayInput は誕生日入力を処理する
func (s *messageService) handleBirthdayInput(ctx context.Context, user *model.User, text string) (string, error) {
	// 誕生日のバリデーション（YYYY-MM-DD形式）
	birthday := strings.TrimSpace(text)
	birthdayPattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !birthdayPattern.MatchString(birthday) {
		return "誕生日はYYYY-MM-DD形式で入力してください。\n例: 2000-01-15", nil
	}

	// ユーザー情報を更新
	user.Birthday = birthday
	user.RegistrationStep = 2

	if err := s.userService.UpdateUser(ctx, user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	return "登録完了！ありがとう。", nil
}
