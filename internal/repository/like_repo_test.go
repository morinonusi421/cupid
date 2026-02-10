package repository

import (
	"context"
	"testing"

	"github.com/morinonusi421/cupid/internal/model"
)

func TestLikeRepository_FindMatchingLike(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewLikeRepository(db)
	userRepo := NewUserRepository(db)

	// ユーザーA: 山田太郎
	userA := &model.User{
		LineID:           "U_A",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
		RegisteredAt:     "2026-02-02 00:00:00",
		UpdatedAt:        "2026-02-02 00:00:00",
	}
	if err := userRepo.Create(context.Background(), userA); err != nil {
		t.Fatal(err)
	}

	// ユーザーB: 佐藤花子
	userB := &model.User{
		LineID:           "U_B",
		Name:             "佐藤花子",
		Birthday:         "1992-02-02",
		RegistrationStep: 1,
		RegisteredAt:     "2026-02-02 00:00:00",
		UpdatedAt:        "2026-02-02 00:00:00",
	}
	if err := userRepo.Create(context.Background(), userB); err != nil {
		t.Fatal(err)
	}

	// A → B を登録
	likeAtoB := &model.Like{
		FromUserID: "U_A",
		ToName:     "佐藤花子",
		ToBirthday: "1992-02-02",
		Matched:    false,
	}
	if err := repo.Create(context.Background(), likeAtoB); err != nil {
		t.Fatal(err)
	}

	// B → A を登録
	likeBtoA := &model.Like{
		FromUserID: "U_B",
		ToName:     "山田太郎",
		ToBirthday: "1990-01-01",
		Matched:    false,
	}
	if err := repo.Create(context.Background(), likeBtoA); err != nil {
		t.Fatal(err)
	}

	// B が A を登録しているか検索
	found, err := repo.FindMatchingLike(context.Background(), "U_B", "山田太郎", "1990-01-01")
	if err != nil {
		t.Errorf("FindMatchingLike failed: %v", err)
	}
	if found == nil {
		t.Error("Matching like not found")
	}
	if found.FromUserID != "U_B" {
		t.Errorf("FromUserID mismatch: got %s, want U_B", found.FromUserID)
	}
}

func TestLikeRepository_UpdateMatched(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewLikeRepository(db)
	userRepo := NewUserRepository(db)

	user := &model.User{
		LineID:           "U_TEST",
		Name:             "テスト",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
		RegisteredAt:     "2026-02-02 00:00:00",
		UpdatedAt:        "2026-02-02 00:00:00",
	}
	userRepo.Create(context.Background(), user)

	like := &model.Like{
		FromUserID: "U_TEST",
		ToName:     "相手",
		ToBirthday: "1995-05-05",
		Matched:    false,
	}
	repo.Create(context.Background(), like)

	// matchedをtrueに更新
	found, _ := repo.FindMatchingLike(context.Background(), "U_TEST", "相手", "1995-05-05")
	err := repo.UpdateMatched(context.Background(), found.ID, true)
	if err != nil {
		t.Errorf("UpdateMatched failed: %v", err)
	}

	// 更新されたか確認
	updated, _ := repo.FindMatchingLike(context.Background(), "U_TEST", "相手", "1995-05-05")
	if !updated.Matched {
		t.Error("Matched flag not updated")
	}
}
