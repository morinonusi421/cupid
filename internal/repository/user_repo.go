package repository

import (
	"context"
	"database/sql"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/morinonusi421/cupid/entities"
	"github.com/morinonusi421/cupid/internal/model"
)

// UserRepository はユーザーのデータアクセス層のインターフェース
type UserRepository interface {
	FindByLineID(ctx context.Context, lineID string) (*model.User, error)
	FindByNameAndBirthday(ctx context.Context, name, birthday string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	FindMatchingUser(ctx context.Context, currentUser *model.User) (*model.User, error)
}

type userRepository struct {
	db *sql.DB
}

// NewUserRepository は UserRepository の新しいインスタンスを作成する
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

// FindByLineID は LINE ユーザーID でユーザーを検索する
func (r *userRepository) FindByLineID(ctx context.Context, lineID string) (*model.User, error) {
	entityUser, err := entities.Users(
		qm.Where("line_user_id = ?", lineID),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // ユーザーが見つからない場合は nil を返す
		}
		return nil, err
	}

	return entityToModel(entityUser), nil
}

// FindByNameAndBirthday は名前と誕生日でユーザーを検索する
func (r *userRepository) FindByNameAndBirthday(ctx context.Context, name, birthday string) (*model.User, error) {
	entityUser, err := entities.Users(
		qm.Where("name = ? AND birthday = ?", name, birthday),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return entityToModel(entityUser), nil
}

// Create は新しいユーザーを作成する
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	entityUser := modelToEntity(user)
	return entityUser.Insert(ctx, r.db, boil.Infer())
}

// Update は既存のユーザーを更新する
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	entityUser := modelToEntity(user)
	_, err := entityUser.Update(ctx, r.db, boil.Infer())
	return err
}

// FindMatchingUser は相互にcrushしているユーザーを検索する
func (r *userRepository) FindMatchingUser(ctx context.Context, currentUser *model.User) (*model.User, error) {
	entityUser, err := entities.Users(
		qm.Where("name = ? AND birthday = ? AND crush_name = ? AND crush_birthday = ? AND matched_with_user_id IS NULL",
			currentUser.CrushName,
			currentUser.CrushBirthday,
			currentUser.Name,
			currentUser.Birthday,
		),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return entityToModel(entityUser), nil
}

// entityToModel は entities.User を model.User に変換する
func entityToModel(e *entities.User) *model.User {
	return &model.User{
		LineID:            e.LineUserID.String,
		Name:              e.Name,
		Birthday:          e.Birthday,
		RegistrationStep:  int(e.RegistrationStep),
		CrushName:         e.CrushName,
		CrushBirthday:     e.CrushBirthday,
		MatchedWithUserID: e.MatchedWithUserID,
		RegisteredAt:      e.RegisteredAt,
		UpdatedAt:         e.UpdatedAt,
	}
}

// modelToEntity は model.User を entities.User に変換する
func modelToEntity(m *model.User) *entities.User {
	return &entities.User{
		LineUserID:        null.StringFrom(m.LineID),
		Name:              m.Name,
		Birthday:          m.Birthday,
		RegistrationStep:  int64(m.RegistrationStep),
		CrushName:         m.CrushName,
		CrushBirthday:     m.CrushBirthday,
		MatchedWithUserID: m.MatchedWithUserID,
		RegisteredAt:      m.RegisteredAt,
		UpdatedAt:         m.UpdatedAt,
	}
}
