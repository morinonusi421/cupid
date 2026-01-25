package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/morinonusi421/cupid/internal/liff"
	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/internal/repository"
)

// UserService はユーザーのビジネスロジック層のインターフェース
type UserService interface {
	RegisterUser(ctx context.Context, lineID, displayName string) error
	GetOrCreateUser(ctx context.Context, lineID, displayName string) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	VerifyLIFFToken(accessToken string) (string, error)
	ProcessTextMessage(ctx context.Context, userID, text string) (string, error)
	RegisterFromLIFF(ctx context.Context, userID, name, birthday string) error
}

type userService struct {
	userRepo          repository.UserRepository
	liffVerifier      *liff.Verifier
	liffRegisterURL   string
}

// NewUserService は UserService の新しいインスタンスを作成する
func NewUserService(userRepo repository.UserRepository, liffVerifier *liff.Verifier, liffRegisterURL string) UserService {
	return &userService{
		userRepo:        userRepo,
		liffVerifier:    liffVerifier,
		liffRegisterURL: liffRegisterURL,
	}
}

// RegisterUser は新しいユーザーを登録する
func (s *userService) RegisterUser(ctx context.Context, lineID, displayName string) error {
	user := &model.User{
		LineID:           lineID,
		Name:             displayName,
		Birthday:         "",
		RegistrationStep: 0, // 0: awaiting_name
		TempCrushName:    "",
		RegisteredAt:     "", // DBのDEFAULTを使用
		UpdatedAt:        "", // DBのDEFAULTを使用
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetOrCreateUser はユーザーを取得するか、存在しない場合は作成する
func (s *userService) GetOrCreateUser(ctx context.Context, lineID, displayName string) (*model.User, error) {
	// 既存ユーザーを検索
	user, err := s.userRepo.FindByLineID(ctx, lineID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// ユーザーが存在する場合は返す
	if user != nil {
		return user, nil
	}

	// ユーザーが存在しない場合は作成
	if err := s.RegisterUser(ctx, lineID, displayName); err != nil {
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	// 作成したユーザーを取得
	user, err = s.userRepo.FindByLineID(ctx, lineID)
	if err != nil {
		return nil, fmt.Errorf("failed to find created user: %w", err)
	}

	return user, nil
}

// UpdateUser は既存のユーザー情報を更新する
func (s *userService) UpdateUser(ctx context.Context, user *model.User) error {
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// VerifyLIFFToken はLIFFアクセストークンを検証してLINE user IDを返す
func (s *userService) VerifyLIFFToken(accessToken string) (string, error) {
	userID, err := s.liffVerifier.VerifyAccessToken(accessToken)
	if err != nil {
		return "", fmt.Errorf("failed to verify LIFF token: %w", err)
	}
	return userID, nil
}

// ProcessTextMessage はテキストメッセージを処理して返信テキストを決定する
func (s *userService) ProcessTextMessage(ctx context.Context, userID, text string) (string, error) {
	// ユーザーを取得または作成
	user, err := s.GetOrCreateUser(ctx, userID, "")
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

// handleInitialMessage は初回メッセージを処理する（LIFF登録フォームの案内）
func (s *userService) handleInitialMessage(ctx context.Context, user *model.User) (string, error) {
	// LIFF URLが設定されている場合はLIFF登録を案内、設定されていない場合は従来の手動登録
	if s.liffRegisterURL != "" {
		return fmt.Sprintf("初めまして！💘\n\n下のリンクから登録してね。\n\n%s", s.liffRegisterURL), nil
	}

	// LIFF URLが設定されていない場合は、手動登録フロー
	user.RegistrationStep = 1
	if err := s.UpdateUser(ctx, user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}
	return "初めまして！まずは名前を教えてね。", nil
}

// handleNameInput は名前入力を処理する
func (s *userService) handleNameInput(ctx context.Context, user *model.User, text string) (string, error) {
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

	if err := s.UpdateUser(ctx, user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	return fmt.Sprintf("%sさん、よろしくね。\n次に、誕生日を教えて（YYYY-MM-DD形式で入力してね）", name), nil
}

// handleBirthdayInput は誕生日入力を処理する
func (s *userService) handleBirthdayInput(ctx context.Context, user *model.User, text string) (string, error) {
	// 誕生日のバリデーション（YYYY-MM-DD形式）
	birthday := strings.TrimSpace(text)
	birthdayPattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !birthdayPattern.MatchString(birthday) {
		return "誕生日はYYYY-MM-DD形式で入力してください。\n例: 2000-01-15", nil
	}

	// ユーザー情報を更新
	user.Birthday = birthday
	user.RegistrationStep = 2

	if err := s.UpdateUser(ctx, user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	return "登録完了！ありがとう。", nil
}

// RegisterFromLIFF はLIFFフォームから送信された登録情報を保存する
func (s *userService) RegisterFromLIFF(ctx context.Context, userID, name, birthday string) error {
	// Get or create user
	user, err := s.GetOrCreateUser(ctx, userID, "")
	if err != nil {
		return fmt.Errorf("failed to get or create user: %w", err)
	}

	// Update user info
	user.Name = name
	user.Birthday = birthday
	user.RegistrationStep = 3 // Registration complete

	if err := s.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}
