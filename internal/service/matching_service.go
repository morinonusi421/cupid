package service

import (
	"context"

	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/internal/repository"
)

// MatchingService はマッチング処理を行うドメインサービスのインターフェース
type MatchingService interface {
	CheckAndUpdateMatch(ctx context.Context, currentUser *model.User, currentLike *model.Like) (matched bool, matchedUserName string, err error)
}

// matchingService は MatchingService の実装
type matchingService struct {
	userRepo repository.UserRepository
	likeRepo repository.LikeRepository
}

// NewMatchingService は MatchingService の新しいインスタンスを作成する
func NewMatchingService(userRepo repository.UserRepository, likeRepo repository.LikeRepository) MatchingService {
	return &matchingService{
		userRepo: userRepo,
		likeRepo: likeRepo,
	}
}

// CheckAndUpdateMatch は相互マッチングをチェックし、マッチした場合は両方の matched フラグを更新する
//
// 処理の流れ:
// 1. 相手（crush）がユーザーテーブルに存在するかチェック
// 2. 相手も自分を登録しているかチェック（FindMatchingLike）
// 3. 両方が真の場合、両方の Like レコードの matched を true に更新
//
// 戻り値:
//   - matched: マッチングが成立したかどうか
//   - matchedUserName: マッチング相手の名前（マッチング成立時のみ）
//   - err: エラー（あれば）
func (s *matchingService) CheckAndUpdateMatch(
	ctx context.Context,
	currentUser *model.User,
	currentLike *model.Like,
) (matched bool, matchedUserName string, err error) {
	// 1. 相手がユーザーテーブルに存在するかチェック
	crushUser, err := s.userRepo.FindByNameAndBirthday(ctx, currentLike.ToName, currentLike.ToBirthday)
	if err != nil {
		return false, "", err
	}

	// 相手が未登録の場合はマッチング成立しない
	if crushUser == nil {
		return false, "", nil
	}

	// 2. 相手も自分を登録しているかチェック
	crushLike, err := s.likeRepo.FindMatchingLike(ctx, crushUser.LineID, currentUser.Name, currentUser.Birthday)
	if err != nil {
		return false, "", err
	}

	// 相手が自分を登録していない場合はマッチング成立しない
	if crushLike == nil {
		return false, "", nil
	}

	// 3. 両方の Like レコードの matched を true に更新
	currentLike.MarkAsMatched()
	crushLike.MarkAsMatched()

	if err := s.likeRepo.UpdateMatched(ctx, currentLike.ID, true); err != nil {
		return false, "", err
	}

	if err := s.likeRepo.UpdateMatched(ctx, crushLike.ID, true); err != nil {
		return false, "", err
	}

	return true, crushUser.Name, nil
}
