package repository

import (
	"context"
	"database/sql"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/morinonusi421/cupid/entities"
	"github.com/morinonusi421/cupid/internal/model"
)

// LikeRepository は好きな人登録のデータアクセス層
type LikeRepository interface {
	// Create は新しい好きな人登録を作成（UPSERT）
	Create(ctx context.Context, like *model.Like) error

	// FindByFromUserID は登録者IDで検索
	FindByFromUserID(ctx context.Context, fromUserID string) (*model.Like, error)

	// FindMatchingLike は相互マッチングを検索
	// fromUserIDのユーザーが toName+toBirthday を登録しているか
	FindMatchingLike(ctx context.Context, fromUserID, toName, toBirthday string) (*model.Like, error)

	// UpdateMatched はマッチングフラグを更新
	UpdateMatched(ctx context.Context, id int64, matched bool) error
}

type likeRepository struct {
	db *sql.DB
}

func NewLikeRepository(db *sql.DB) LikeRepository {
	return &likeRepository{db: db}
}

// Create は新しい好きな人登録を作成（UPSERT）
func (r *likeRepository) Create(ctx context.Context, like *model.Like) error {
	cols := model.LikeToColumns(like)

	entityLike := &entities.Like{}
	// colsの値をentityLikeにセット
	entityLike.FromUserID = cols[entities.LikeColumns.FromUserID].(string)
	entityLike.ToName = cols[entities.LikeColumns.ToName].(string)
	entityLike.ToBirthday = cols[entities.LikeColumns.ToBirthday].(string)
	entityLike.Matched = int64(cols[entities.LikeColumns.Matched].(int))

	return entityLike.Upsert(
		ctx,
		r.db,
		true, // updateOnConflict
		[]string{entities.LikeColumns.FromUserID}, // conflict columns
		boil.Whitelist(
			entities.LikeColumns.ToName,
			entities.LikeColumns.ToBirthday,
			entities.LikeColumns.Matched,
		),
		boil.Infer(),
	)
}

// FindByFromUserID は登録者IDで検索
func (r *likeRepository) FindByFromUserID(ctx context.Context, fromUserID string) (*model.Like, error) {
	entity, err := entities.Likes(
		qm.Where("from_user_id = ?", fromUserID),
	).One(ctx, r.db)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return model.EntityToLike(entity), nil
}

// FindMatchingLike は相互マッチングを検索
func (r *likeRepository) FindMatchingLike(ctx context.Context, fromUserID, toName, toBirthday string) (*model.Like, error) {
	entity, err := entities.Likes(
		qm.Where("from_user_id = ? AND to_name = ? AND to_birthday = ?", fromUserID, toName, toBirthday),
	).One(ctx, r.db)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return model.EntityToLike(entity), nil
}

// UpdateMatched はマッチングフラグを更新
func (r *likeRepository) UpdateMatched(ctx context.Context, id int64, matched bool) error {
	matchedInt := int64(0)
	if matched {
		matchedInt = 1
	}

	_, err := entities.Likes(
		qm.Where("id = ?", id),
	).UpdateAll(ctx, r.db, entities.M{
		entities.LikeColumns.Matched: matchedInt,
	})

	return err
}
