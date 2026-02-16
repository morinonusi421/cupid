package service

import (
	"context"
	"fmt"

	"github.com/aarondl/null/v8"
	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/internal/repository"
)

// MatchingService はマッチング処理を行うドメインサービスのインターフェース
type MatchingService interface {
	CheckAndUpdateMatch(ctx context.Context, currentUser *model.User) (matched bool, matchedUser *model.User, err error)
	UnmatchUsers(ctx context.Context, initiatorUserID, partnerUserID string) (initiatorUser *model.User, partnerUser *model.User, err error)
}

// matchingService は MatchingService の実装
type matchingService struct {
	userRepo repository.UserRepository
}

// NewMatchingService は MatchingService の新しいインスタンスを作成する
func NewMatchingService(userRepo repository.UserRepository) MatchingService {
	return &matchingService{
		userRepo: userRepo,
	}
}

// CheckAndUpdateMatch は相互マッチングをチェックし、マッチした場合は両方の matched_with_user_id を更新する
//
// 処理の流れ:
// 1. 相互にcrushしているユーザーを検索（FindMatchingUser）
// 2. 両方が真の場合、両方の matched_with_user_id を更新
//
// 戻り値:
//   - matched: マッチングが成立したかどうか
//   - matchedUser: マッチング相手のUserオブジェクト（マッチング成立時のみ）
//   - err: エラー（あれば）
func (s *matchingService) CheckAndUpdateMatch(
	ctx context.Context,
	currentUser *model.User,
) (matched bool, matchedUser *model.User, err error) {
	// 1. 相互にcrushしているユーザーを検索
	matchedUser, err = s.userRepo.FindMatchingUser(ctx, currentUser)
	if err != nil {
		return false, nil, err
	}

	// マッチング相手が見つからない場合
	if matchedUser == nil {
		return false, nil, nil
	}

	// 2. 両方の matched_with_user_id を更新
	currentUser.MatchedWithUserID = null.StringFrom(matchedUser.LineID)
	matchedUser.MatchedWithUserID = null.StringFrom(currentUser.LineID)

	if err := s.userRepo.Update(ctx, currentUser); err != nil {
		return false, nil, err
	}

	if err := s.userRepo.Update(ctx, matchedUser); err != nil {
		return false, nil, err
	}

	return true, matchedUser, nil
}

// UnmatchUsers はマッチングを解除する（通知送信は行わない）
//
// 戻り値:
//   - initiatorUser: 解除を開始したユーザー（matched_with_user_id が NULL に更新済み）
//   - partnerUser: 相手ユーザー（matched_with_user_id が NULL に更新済み）
//   - err: エラー（あれば）
func (s *matchingService) UnmatchUsers(ctx context.Context, initiatorUserID, partnerUserID string) (*model.User, *model.User, error) {
	// 開始ユーザーの情報を取得
	initiatorUser, err := s.userRepo.FindByLineID(ctx, initiatorUserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find initiator user: %w", err)
	}
	if initiatorUser == nil {
		return nil, nil, fmt.Errorf("initiator user not found: %s", initiatorUserID)
	}

	// 相手のユーザー情報を取得
	partnerUser, err := s.userRepo.FindByLineID(ctx, partnerUserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find partner user: %w", err)
	}
	if partnerUser == nil {
		return nil, nil, fmt.Errorf("partner user not found: %s", partnerUserID)
	}

	// 両方の matched_with_user_id を NULL に
	initiatorUser.MatchedWithUserID = null.String{Valid: false}
	partnerUser.MatchedWithUserID = null.String{Valid: false}

	if err := s.userRepo.Update(ctx, initiatorUser); err != nil {
		return nil, nil, fmt.Errorf("failed to update initiator user: %w", err)
	}

	if err := s.userRepo.Update(ctx, partnerUser); err != nil {
		return nil, nil, fmt.Errorf("failed to update partner user: %w", err)
	}

	return initiatorUser, partnerUser, nil
}
