package service

import (
	"context"
	"fmt"

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
	RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string) (matched bool, matchedUserName string, err error)
}

type userService struct {
	userRepo          repository.UserRepository
	likeRepo          repository.LikeRepository
	liffVerifier      *liff.Verifier
	liffRegisterURL   string
}

// NewUserService は UserService の新しいインスタンスを作成する
func NewUserService(userRepo repository.UserRepository, likeRepo repository.LikeRepository, liffVerifier *liff.Verifier, liffRegisterURL string) UserService {
	return &userService{
		userRepo:        userRepo,
		likeRepo:        likeRepo,
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
		RegistrationStep: 0, // 0: 未登録
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
		// 初期状態 - Web登録フォームの案内
		return s.handleInitialMessage(ctx, user)
	case 1:
		// 登録完了済み - オウム返し（後で通常機能に変更予定）
		return text, nil
	default:
		return "", fmt.Errorf("invalid registration step: %d", user.RegistrationStep)
	}
}

// handleInitialMessage は初回メッセージを処理する（Web登録フォームの案内）
func (s *userService) handleInitialMessage(ctx context.Context, user *model.User) (string, error) {
	// TODO: セキュリティ改善 - ワンタイムトークン方式に変更する
	// 現在はURLパラメータに直接user_idを含めているが、なりすまし可能

	// URLにuser_idをクエリパラメータとして追加
	registerURL := fmt.Sprintf("%s?user_id=%s", s.liffRegisterURL, user.LineID)
	return fmt.Sprintf("初めまして！💘\n\n下のリンクから登録してね。\n\n%s", registerURL), nil
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
	user.RegistrationStep = 1 // Registration complete

	if err := s.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// RegisterCrush は好きな人を登録し、マッチング判定を行う
func (s *userService) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string) (matched bool, matchedUserName string, err error) {
	// 1. 現在のユーザー情報を取得
	currentUser, err := s.userRepo.FindByLineID(ctx, userID)
	if err != nil {
		return false, "", err
	}
	if currentUser == nil {
		return false, "", fmt.Errorf("user not found: %s", userID)
	}

	// 2. 自己登録チェック
	if currentUser.Name == crushName && currentUser.Birthday == crushBirthday {
		return false, "", fmt.Errorf("cannot register yourself")
	}

	// 3. 好きな人を登録（UPSERT）
	like := &model.Like{
		FromUserID:  userID,
		ToName:      crushName,
		ToBirthday:  crushBirthday,
		Matched:     false,
	}
	if err := s.likeRepo.Create(ctx, like); err != nil {
		return false, "", err
	}

	// 4. マッチング判定
	// 4-1. 好きな人がusersテーブルに存在するか確認
	crushUser, err := s.userRepo.FindByNameAndBirthday(ctx, crushName, crushBirthday)
	if err != nil {
		return false, "", err
	}
	if crushUser == nil {
		// 相手が未登録 → マッチング不可
		return false, "", nil
	}

	// 4-2. 相手も自分を登録しているか確認
	reverseLike, err := s.likeRepo.FindMatchingLike(ctx, crushUser.LineID, currentUser.Name, currentUser.Birthday)
	if err != nil {
		return false, "", err
	}
	if reverseLike == nil {
		// 相手は自分を登録していない → マッチング不可
		return false, "", nil
	}

	// 5. マッチング成立！
	// 5-1. 自分のlikeレコードを取得してIDを確認
	currentLike, err := s.likeRepo.FindByFromUserID(ctx, userID)
	if err != nil {
		return false, "", err
	}

	// 5-2. 両方のmatchedフラグを1に更新
	if err := s.likeRepo.UpdateMatched(ctx, currentLike.ID, true); err != nil {
		return false, "", err
	}
	if err := s.likeRepo.UpdateMatched(ctx, reverseLike.ID, true); err != nil {
		return false, "", err
	}

	// 5-3. LINE通知を送信
	// TODO: PushMessage実装後に追加

	return true, crushName, nil
}
