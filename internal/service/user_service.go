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
}

type userService struct {
	userRepo     repository.UserRepository
	liffVerifier *liff.Verifier
}

// NewUserService は UserService の新しいインスタンスを作成する
func NewUserService(userRepo repository.UserRepository, liffVerifier *liff.Verifier) UserService {
	return &userService{
		userRepo:     userRepo,
		liffVerifier: liffVerifier,
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
