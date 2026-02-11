package service

import (
	"context"
	"fmt"
	"log"

	"github.com/aarondl/null/v8"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/morinonusi421/cupid/internal/linebot"
	"github.com/morinonusi421/cupid/internal/message"
	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/internal/repository"
)

// UserService はユーザーのビジネスロジック層のインターフェース
type UserService interface {
	ProcessTextMessage(ctx context.Context, userID, text string) (string, error)
	RegisterFromLIFF(ctx context.Context, userID, name, birthday string, confirmUnmatch bool) (isFirstRegistration bool, err error)
	RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string, confirmUnmatch bool) (matched bool, matchedUserName string, isFirstCrushRegistration bool, err error)
	HandleFollowEvent(ctx context.Context, replyToken string) error
}

type userService struct {
	userRepo        repository.UserRepository
	userLiffURL     string
	crushLiffURL    string
	matchingService MatchingService
	lineBotClient   linebot.Client
}

// NewUserService は UserService の新しいインスタンスを作成する
func NewUserService(userRepo repository.UserRepository, userLiffURL string, crushLiffURL string, matchingService MatchingService, lineBotClient linebot.Client) UserService {
	return &userService{
		userRepo:        userRepo,
		userLiffURL:     userLiffURL,
		crushLiffURL:    crushLiffURL,
		matchingService: matchingService,
		lineBotClient:   lineBotClient,
	}
}

// ProcessTextMessage はテキストメッセージを処理して返信テキストを決定する
func (s *userService) ProcessTextMessage(ctx context.Context, userID, text string) (string, error) {
	// DBからユーザーを検索（createはしない）
	user, err := s.userRepo.FindByLineID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	// ユーザーが未登録の場合
	if user == nil {
		// LIFFフォームへの案内（DB登録はしない）
		return message.UnregisteredUserPrompt(s.userLiffURL), nil
	}

	// 登録済みの場合、registration_step に応じて処理分岐
	switch user.RegistrationStep {
	case 0:
		// DB登録済みなのに registration_step が 0 は異常な状態
		return "", fmt.Errorf("invalid state: user exists but registration_step is 0 (user_id: %s)", userID)
	case 1:
		// ユーザー登録完了済み - 好きな人の登録を案内（LIFF URL）
		return message.RegistrationStep1Prompt(s.crushLiffURL), nil
	case 2:
		// 好きな人登録完了済み - 再登録を案内（LIFF URL）
		return message.AlreadyRegisteredMessage, nil
	default:
		return "", fmt.Errorf("invalid registration step: %d", user.RegistrationStep)
	}
}

// RegisterFromLIFF はLIFFフォームから送信された登録情報を保存する
func (s *userService) RegisterFromLIFF(ctx context.Context, userID, name, birthday string, confirmUnmatch bool) (isFirstRegistration bool, err error) {
	// 1. バリデーション
	if ok, errMsg := model.IsValidName(name); !ok {
		return false, fmt.Errorf("%s", errMsg)
	}

	// 2. 重複チェック（既存ユーザーと名前・誕生日が被っていないか）
	existingUser, err := s.userRepo.FindByNameAndBirthday(ctx, name, birthday)
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate user: %w", err)
	}
	// 見つかったユーザーが他人（LineIDが違う）の場合はエラー
	if existingUser != nil && existingUser.LineID != userID {
		return false, fmt.Errorf("duplicate_user")
	}

	// 3. ユーザー検索
	user, err := s.userRepo.FindByLineID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to find user: %w", err)
	}

	// 4. 初回登録 vs 再登録で分岐
	if user == nil {
		// 初回登録
		if err := s.registerNewUser(ctx, userID, name, birthday); err != nil {
			return false, err
		}
		return true, nil
	} else {
		// 再登録（情報更新）
		if err := s.updateUserInfo(ctx, user, name, birthday, confirmUnmatch); err != nil {
			return false, err
		}
		return false, nil
	}
}

// RegisterCrush は好きな人を登録し、マッチング判定を行う
func (s *userService) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string, confirmUnmatch bool) (matched bool, matchedUserName string, isFirstCrushRegistration bool, err error) {
	// 1. 現在のユーザー情報を取得
	currentUser, err := s.userRepo.FindByLineID(ctx, userID)
	if err != nil {
		return false, "", false, err
	}
	if currentUser == nil {
		return false, "", false, fmt.Errorf("user not found: %s", userID)
	}

	// 2. マッチング中かチェック
	if currentUser.IsMatched() && !confirmUnmatch {
		return false, "", false, fmt.Errorf("matched_user_exists")
	}

	// 3. マッチング解除処理
	if currentUser.IsMatched() && confirmUnmatch {
		if err := s.unmatchUsers(ctx, currentUser, currentUser.MatchedWithUserID.String); err != nil {
			log.Printf("Failed to unmatch users: %v", err)
			// エラーをログに記録するが、処理は継続（Crush更新は実施）
		}
	}

	// 4. 自己登録チェック（domain method使用）
	if currentUser.IsSamePerson(crushName, crushBirthday) {
		return false, "", false, fmt.Errorf("cannot register yourself")
	}

	// 5. 名前のバリデーション
	if valid, errMsg := model.IsValidName(crushName); !valid {
		return false, "", false, fmt.Errorf("%s", errMsg)
	}

	// 6. 初回登録か再登録かを判定（RegistrationStepを変更する前に）
	isFirstCrushRegistration = currentUser.RegistrationStep == 1

	// 7. 好きな人を登録（usersテーブルに直接保存）
	currentUser.CrushName = null.StringFrom(crushName)
	currentUser.CrushBirthday = null.StringFrom(crushBirthday)

	// 8. RegistrationStepを2に更新（domain method使用）
	currentUser.CompleteCrushRegistration()

	if err := s.userRepo.Update(ctx, currentUser); err != nil {
		return false, "", false, err
	}

	// 9. マッチング判定（MatchingService に委譲）
	var matchedUser *model.User
	matched, matchedUser, err = s.matchingService.CheckAndUpdateMatch(ctx, currentUser)
	if err != nil {
		return false, "", false, fmt.Errorf("matching check failed: %w", err)
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
	} else {
		// マッチしなかった場合も登録完了を通知
		if err := s.sendCrushRegistrationComplete(ctx, currentUser, isFirstCrushRegistration); err != nil {
			log.Printf("Failed to send crush registration complete notification to %s: %v", currentUser.LineID, err)
			// エラーをログに記録するが、処理は継続
		}
	}

	matchedUserName = ""
	if matchedUser != nil {
		matchedUserName = matchedUser.Name
	}

	return matched, matchedUserName, isFirstCrushRegistration, nil
}

// registerNewUser は初回登録時に新規ユーザーを作成する
func (s *userService) registerNewUser(ctx context.Context, userID, name, birthday string) error {
	// 1. 完全なユーザーオブジェクトを作成
	user := &model.User{
		LineID:           userID,
		Name:             name,
		Birthday:         birthday,
		RegistrationStep: 1,  // 最初から登録完了状態
		RegisteredAt:     "", // DBのDEFAULT（現在時刻）を使用
		UpdatedAt:        "", // DBのDEFAULT（現在時刻）を使用
	}

	// 2. DBに保存
	if err := s.userRepo.Create(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// 3. 好きな人登録を促すメッセージを送信
	if err := s.sendCrushRegistrationPrompt(ctx, user); err != nil {
		log.Printf("Failed to send crush registration prompt to %s: %v", user.LineID, err)
		// エラーをログに記録するが、登録処理は成功として扱う
	}

	return nil
}

// updateUserInfo は再登録時に既存ユーザーの情報を更新する
func (s *userService) updateUserInfo(ctx context.Context, user *model.User, name, birthday string, confirmUnmatch bool) error {
	// 1. 自己登録チェック（好きな人と同じ名前・誕生日にならないか）
	if user.CrushName.Valid && user.CrushBirthday.Valid {
		if user.CrushName.String == name && user.CrushBirthday.String == birthday {
			return fmt.Errorf("自分自身は登録できません")
		}
	}

	// 2. マッチング中かチェック
	if user.IsMatched() && !confirmUnmatch {
		return fmt.Errorf("matched_user_exists")
	}

	// 3. マッチング解除処理
	if user.IsMatched() && confirmUnmatch {
		if err := s.unmatchUsers(ctx, user, user.MatchedWithUserID.String); err != nil {
			log.Printf("Failed to unmatch users: %v", err)
			// エラーをログに記録するが、処理は継続（情報更新は実施）
		}
	}

	// 4. ユーザー情報を更新
	user.Name = name
	user.Birthday = birthday

	// 4. registration_step が 0 の場合のみ 1 に更新（通常はありえないが念のため）
	if user.RegistrationStep == 0 {
		user.CompleteUserRegistration()
	}

	// 5. DBに保存
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// 6. 更新完了メッセージを送信
	if err := s.sendUserInfoUpdateConfirmation(ctx, user); err != nil {
		log.Printf("Failed to send update confirmation to %s: %v", user.LineID, err)
		// エラーをログに記録するが、更新処理は成功として扱う
	}

	return nil
}

// sendMatchNotification はマッチ成立時にLINE Push通知を送信する
func (s *userService) sendMatchNotification(ctx context.Context, toUser *model.User, matchedWithUser *model.User) error {
	request := &messaging_api.PushMessageRequest{
		To: toUser.LineID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: message.MatchNotification(matchedWithUser.Name),
			},
		},
		NotificationDisabled: false,
	}

	_, err := s.lineBotClient.PushMessage(request)
	return err
}

// HandleFollowEvent はFollowイベント時の挨拶メッセージ（QuickReply付き）を送信する
func (s *userService) HandleFollowEvent(ctx context.Context, replyToken string) error {
	request := &messaging_api.ReplyMessageRequest{
		ReplyToken: replyToken,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: message.FollowGreeting,
				QuickReply: &messaging_api.QuickReply{
					Items: []messaging_api.QuickReplyItem{
						{
							Type: "action",
							Action: &messaging_api.UriAction{
								Label: "登録する",
								Uri:   s.userLiffURL,
							},
						},
					},
				},
			},
		},
	}

	_, err := s.lineBotClient.ReplyMessage(request)
	return err
}

// sendCrushRegistrationPrompt はユーザー登録完了後に好きな人登録を促すメッセージを送信する
func (s *userService) sendCrushRegistrationPrompt(ctx context.Context, user *model.User) error {
	request := &messaging_api.PushMessageRequest{
		To: user.LineID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: message.UserRegistrationComplete,
				QuickReply: &messaging_api.QuickReply{
					Items: []messaging_api.QuickReplyItem{
						{
							Type: "action",
							Action: &messaging_api.UriAction{
								Label: "好きな人を登録",
								Uri:   s.crushLiffURL,
							},
						},
					},
				},
			},
		},
		NotificationDisabled: false,
	}

	_, err := s.lineBotClient.PushMessage(request)
	return err
}

// sendUserInfoUpdateConfirmation は情報更新完了のメッセージを送信する
func (s *userService) sendUserInfoUpdateConfirmation(ctx context.Context, user *model.User) error {
	request := &messaging_api.PushMessageRequest{
		To: user.LineID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: message.UserInfoUpdateConfirmation,
			},
		},
		NotificationDisabled: false,
	}

	_, err := s.lineBotClient.PushMessage(request)
	return err
}

// sendCrushRegistrationComplete は好きな人登録完了時（マッチなし）のメッセージを送信する
func (s *userService) sendCrushRegistrationComplete(ctx context.Context, user *model.User, isFirstRegistration bool) error {
	var messageText string
	if isFirstRegistration {
		messageText = message.CrushRegistrationCompleteFirst
	} else {
		messageText = message.CrushRegistrationCompleteUpdate
	}

	request := &messaging_api.PushMessageRequest{
		To: user.LineID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: messageText,
			},
		},
		NotificationDisabled: false,
	}

	_, err := s.lineBotClient.PushMessage(request)
	return err
}

// unmatchUsers はマッチングを解除し、両方のユーザーに通知を送信する
func (s *userService) unmatchUsers(ctx context.Context, initiatorUser *model.User, partnerUserID string) error {
	// 相手のユーザー情報を取得
	partnerUser, err := s.userRepo.FindByLineID(ctx, partnerUserID)
	if err != nil {
		return fmt.Errorf("failed to find partner user: %w", err)
	}
	if partnerUser == nil {
		return fmt.Errorf("partner user not found: %s", partnerUserID)
	}

	// 両方の matched_with_user_id を NULL に
	initiatorUser.MatchedWithUserID = null.String{Valid: false}
	partnerUser.MatchedWithUserID = null.String{Valid: false}

	if err := s.userRepo.Update(ctx, initiatorUser); err != nil {
		return fmt.Errorf("failed to update initiator user: %w", err)
	}

	if err := s.userRepo.Update(ctx, partnerUser); err != nil {
		return fmt.Errorf("failed to update partner user: %w", err)
	}

	// 両方のユーザーに解除通知を送信
	if err := s.sendUnmatchNotification(ctx, initiatorUser, partnerUser, true); err != nil {
		log.Printf("Failed to send unmatch notification to initiator %s: %v", initiatorUser.LineID, err)
	}

	if err := s.sendUnmatchNotification(ctx, partnerUser, initiatorUser, false); err != nil {
		log.Printf("Failed to send unmatch notification to partner %s: %v", partnerUser.LineID, err)
	}

	return nil
}

// sendUnmatchNotification はマッチング解除時にLINE Push通知を送信する
func (s *userService) sendUnmatchNotification(ctx context.Context, toUser *model.User, partnerUser *model.User, isInitiator bool) error {
	var messageText string
	if isInitiator {
		messageText = message.UnmatchNotificationInitiator(partnerUser.Name)
	} else {
		messageText = message.UnmatchNotificationPartner(partnerUser.Name)
	}

	request := &messaging_api.PushMessageRequest{
		To: toUser.LineID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: messageText,
			},
		},
		NotificationDisabled: false,
	}

	_, err := s.lineBotClient.PushMessage(request)
	return err
}
