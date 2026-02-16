package service

import (
	"context"
	"log"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/morinonusi421/cupid/internal/linebot"
	"github.com/morinonusi421/cupid/internal/message"
)

// NotificationService はLINE通知送信を担当するサービス
type NotificationService interface {
	// SendMatchNotification はマッチ成立時にLINE Push通知を送信する
	SendMatchNotification(ctx context.Context, toUserLineID, matchedUserName string) error

	// SendCrushRegistrationPrompt はユーザー登録完了後に好きな人登録を促すメッセージを送信する
	SendCrushRegistrationPrompt(ctx context.Context, toUserLineID, crushLiffURL string) error

	// SendUserInfoUpdateConfirmation は情報更新完了のメッセージを送信する
	SendUserInfoUpdateConfirmation(ctx context.Context, toUserLineID string) error

	// SendCrushRegistrationComplete は好きな人登録完了時（マッチなし）のメッセージを送信する
	SendCrushRegistrationComplete(ctx context.Context, toUserLineID string, isFirstRegistration bool) error

	// SendUnmatchNotification はマッチング解除時にLINE Push通知を送信する
	SendUnmatchNotification(ctx context.Context, toUserLineID, partnerUserName string, isInitiator bool) error

	// SendFollowGreeting はFollowイベント時の挨拶メッセージ（QuickReply付き）を送信する
	SendFollowGreeting(ctx context.Context, replyToken, userLiffURL string) error

	// SendJoinGroupGreeting はグループに招待された時の挨拶メッセージを送信する
	SendJoinGroupGreeting(ctx context.Context, replyToken string) error
}

type notificationService struct {
	lineBotClient linebot.Client
}

// NewNotificationService は NotificationService の新しいインスタンスを作成する
func NewNotificationService(lineBotClient linebot.Client) NotificationService {
	return &notificationService{
		lineBotClient: lineBotClient,
	}
}

// SendMatchNotification はマッチ成立時にLINE Push通知を送信する
//
// 【重要】有償メッセージ（無料プランでは月200通まで）
// Push APIを使用するため、LINE Messaging APIの有償カウント対象
func (s *notificationService) SendMatchNotification(ctx context.Context, toUserLineID, matchedUserName string) error {
	request := &messaging_api.PushMessageRequest{
		To: toUserLineID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: message.MatchNotification(matchedUserName),
			},
		},
		NotificationDisabled: false,
	}

	_, err := s.lineBotClient.PushMessage(request)
	if err != nil {
		log.Printf("[ERROR] Failed to send match notification (paid message): %v", err)
	}
	return err
}

// SendCrushRegistrationPrompt はユーザー登録完了後に好きな人登録を促すメッセージを送信する
//
// 【重要】有償メッセージ（無料プランでは月200通まで）
// Push APIを使用するため、LINE Messaging APIの有償カウント対象
func (s *notificationService) SendCrushRegistrationPrompt(ctx context.Context, toUserLineID, crushLiffURL string) error {
	request := &messaging_api.PushMessageRequest{
		To: toUserLineID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: message.UserRegistrationComplete,
				QuickReply: &messaging_api.QuickReply{
					Items: []messaging_api.QuickReplyItem{
						{
							Type: "action",
							Action: &messaging_api.UriAction{
								Label: "好きな人を登録",
								Uri:   crushLiffURL,
							},
						},
					},
				},
			},
		},
		NotificationDisabled: false,
	}

	_, err := s.lineBotClient.PushMessage(request)
	if err != nil {
		log.Printf("[ERROR] Failed to send crush registration prompt (paid message): %v", err)
	}
	return err
}

// SendUserInfoUpdateConfirmation は情報更新完了のメッセージを送信する
//
// 【重要】有償メッセージ（無料プランでは月200通まで）
// Push APIを使用するため、LINE Messaging APIの有償カウント対象
func (s *notificationService) SendUserInfoUpdateConfirmation(ctx context.Context, toUserLineID string) error {
	request := &messaging_api.PushMessageRequest{
		To: toUserLineID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: message.UserInfoUpdateConfirmation,
			},
		},
		NotificationDisabled: false,
	}

	_, err := s.lineBotClient.PushMessage(request)
	if err != nil {
		log.Printf("[ERROR] Failed to send user info update confirmation (paid message): %v", err)
	}
	return err
}

// SendCrushRegistrationComplete は好きな人登録完了時（マッチなし）のメッセージを送信する
//
// 【重要】有償メッセージ（無料プランでは月200通まで）
// Push APIを使用するため、LINE Messaging APIの有償カウント対象
func (s *notificationService) SendCrushRegistrationComplete(ctx context.Context, toUserLineID string, isFirstRegistration bool) error {
	var messageText string
	if isFirstRegistration {
		messageText = message.CrushRegistrationCompleteFirst
	} else {
		messageText = message.CrushRegistrationCompleteUpdate
	}

	request := &messaging_api.PushMessageRequest{
		To: toUserLineID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: messageText,
			},
		},
		NotificationDisabled: false,
	}

	_, err := s.lineBotClient.PushMessage(request)
	if err != nil {
		log.Printf("[ERROR] Failed to send crush registration complete (paid message): %v", err)
	}
	return err
}

// SendUnmatchNotification はマッチング解除時にLINE Push通知を送信する
//
// 【重要】有償メッセージ（無料プランでは月200通まで）
// Push APIを使用するため、LINE Messaging APIの有償カウント対象
func (s *notificationService) SendUnmatchNotification(ctx context.Context, toUserLineID, partnerUserName string, isInitiator bool) error {
	var messageText string
	if isInitiator {
		messageText = message.UnmatchNotificationInitiator(partnerUserName)
	} else {
		messageText = message.UnmatchNotificationPartner(partnerUserName)
	}

	request := &messaging_api.PushMessageRequest{
		To: toUserLineID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: messageText,
			},
		},
		NotificationDisabled: false,
	}

	_, err := s.lineBotClient.PushMessage(request)
	if err != nil {
		log.Printf("[ERROR] Failed to send unmatch notification (paid message): %v", err)
	}
	return err
}

// SendFollowGreeting はFollowイベント時の挨拶メッセージ（QuickReply付き）を送信する
func (s *notificationService) SendFollowGreeting(ctx context.Context, replyToken, userLiffURL string) error {
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
								Uri:   userLiffURL,
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

// SendJoinGroupGreeting はグループに招待された時の挨拶メッセージを送信する
func (s *notificationService) SendJoinGroupGreeting(ctx context.Context, replyToken string) error {
	request := &messaging_api.ReplyMessageRequest{
		ReplyToken: replyToken,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: message.JoinGroupGreeting,
			},
		},
	}

	_, err := s.lineBotClient.ReplyMessage(request)
	return err
}
