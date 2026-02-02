package model

import "github.com/morinonusi421/cupid/entities"

// Like は好きな人の登録情報を表す
type Like struct {
	ID         int64
	FromUserID string // 登録したユーザーのLINE ID
	ToName     string // 好きな人の名前
	ToBirthday string // 好きな人の誕生日 (YYYY-MM-DD)
	Matched    bool   // マッチングフラグ
	CreatedAt  string // 作成日時
}

// EntityToLike は entities.Like を model.Like に変換する
func EntityToLike(entity *entities.Like) *Like {
	if entity == nil {
		return nil
	}

	return &Like{
		ID:         entity.ID.Int64,
		FromUserID: entity.FromUserID,
		ToName:     entity.ToName,
		ToBirthday: entity.ToBirthday,
		Matched:    entity.Matched == 1,
		CreatedAt:  entity.CreatedAt,
	}
}

// LikeToColumns は model.Like を SQLBoiler の Columns 構造体に変換する
func LikeToColumns(like *Like) entities.M {
	matched := 0
	if like.Matched {
		matched = 1
	}

	return entities.M{
		entities.LikeColumns.FromUserID: like.FromUserID,
		entities.LikeColumns.ToName:     like.ToName,
		entities.LikeColumns.ToBirthday: like.ToBirthday,
		entities.LikeColumns.Matched:    matched,
	}
}
