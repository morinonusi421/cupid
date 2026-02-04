package service

import (
	"context"
	"fmt"
	"log"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/morinonusi421/cupid/internal/liff"
	"github.com/morinonusi421/cupid/internal/linebot"
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
	matchingService   MatchingService
	lineBotClient     linebot.Client
}

// NewUserService は UserService の新しいインスタンスを作成する
func NewUserService(userRepo repository.UserRepository, likeRepo repository.LikeRepository, liffVerifier *liff.Verifier, liffRegisterURL string, matchingService MatchingService, lineBotClient linebot.Client) UserService {
	return &userService{
		userRepo:        userRepo,
		likeRepo:        likeRepo,
		liffVerifier:    liffVerifier,
		liffRegisterURL: liffRegisterURL,
		matchingService: matchingService,
		lineBotClient:   lineBotClient,
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
		// ユーザー登録完了済み - 好きな人の登録を案内
		crushRegisterURL := fmt.Sprintf("https://cupid-linebot.click/crush/register.html?user_id=%s", userID)
		return fmt.Sprintf("次に、好きな人を登録してください💘\n\n%s", crushRegisterURL), nil
	case 2:
		// 好きな人登録完了済み - 再登録を案内
		crushRegisterURL := fmt.Sprintf("https://cupid-linebot.click/crush/register.html?user_id=%s", userID)
		return fmt.Sprintf("登録済みです。好きな人を変更する場合は下のリンクから再登録できます。\n\n%s", crushRegisterURL), nil
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
	// Validate name format
	if ok, errMsg := model.IsValidName(name); !ok {
		return fmt.Errorf("%s", errMsg)
	}

	// Get or create user
	user, err := s.GetOrCreateUser(ctx, userID, "")
	if err != nil {
		return fmt.Errorf("failed to get or create user: %w", err)
	}

	// Update user info using domain method
	user.Name = name
	user.Birthday = birthday
	user.CompleteUserRegistration()

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

	// 2. 自己登録チェック（domain method使用）
	if currentUser.IsSamePerson(crushName, crushBirthday) {
		return false, "", fmt.Errorf("cannot register yourself")
	}

	// 3. 名前のバリデーション
	if valid, errMsg := model.IsValidName(crushName); !valid {
		return false, "", fmt.Errorf("%s", errMsg)
	}

	// 4. 好きな人を登録（factory method使用）
	like := model.NewLike(userID, crushName, crushBirthday)
	if err := s.likeRepo.Create(ctx, like); err != nil {
		return false, "", err
	}

	// 5. RegistrationStepを2に更新（domain method使用）
	currentUser.CompleteCrushRegistration()
	if err := s.userRepo.Update(ctx, currentUser); err != nil {
		return false, "", err
	}

	// 6. マッチング判定（MatchingService に委譲）
	var matchedUser *model.User
	matched, matchedUser, err = s.matchingService.CheckAndUpdateMatch(ctx, currentUser, like)
	if err != nil {
		return false, "", fmt.Errorf("matching check failed: %w", err)
	}

	// マッチした場合、両方のユーザーにLINE通知を送信
	if matched {
		// 現在のユーザーに通知
		if err := s.sendMatchNotification(ctx, currentUser, matchedUser); err != nil {
			log.Printf("Failed to send match notification to %s: %v", currentUser.LineID, err)
			// エラーをログに記録するが、処理は継続
		}

		// 相手ユーザーに通知
		if err := s.sendMatchNotification(ctx, matchedUser, currentUser); err != nil {
			log.Printf("Failed to send match notification to %s: %v", matchedUser.LineID, err)
			// エラーをログに記録するが、処理は継続
		}
	}

	matchedUserName = ""
	if matchedUser != nil {
		matchedUserName = matchedUser.Name
	}

	return matched, matchedUserName, nil
}

// sendMatchNotification はマッチ成立時にLINE Push通知を送信する
func (s *userService) sendMatchNotification(ctx context.Context, toUser *model.User, matchedWithUser *model.User) error {
	message := fmt.Sprintf("相思相愛が成立しました！\n相手：%s", matchedWithUser.Name)

	request := &messaging_api.PushMessageRequest{
		To: toUser.LineID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: message,
			},
		},
		NotificationDisabled: false,
	}

	_, err := s.lineBotClient.PushMessage(request)
	return err
}
