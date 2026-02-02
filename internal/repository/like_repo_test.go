package repository

import (
	"context"
	"testing"

	"github.com/morinonusi421/cupid/internal/model"
)

func TestLikeRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewLikeRepository(db)

	// テストユーザーを作成
	userRepo := NewUserRepository(db)
	user := &model.User{
		LineID:           "U_TEST_USER",
		Name:             "テストユーザー",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
		RegisteredAt:     "2026-02-02 00:00:00",
		UpdatedAt:        "2026-02-02 00:00:00",
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	// 好きな人を登録
	like := &model.Like{
		FromUserID: "U_TEST_USER",
		ToName:     "好きな人",
		ToBirthday: "1995-05-05",
		Matched:    false,
	}

	err := repo.Create(context.Background(), like)
	if err != nil {
		t.Errorf("Create failed: %v", err)
	}

	// 登録されたか確認
	found, err := repo.FindByFromUserID(context.Background(), "U_TEST_USER")
	if err != nil {
		t.Errorf("FindByFromUserID failed: %v", err)
	}
	if found == nil {
		t.Error("Like not found after Create")
	}
	if found.ToName != "好きな人" {
		t.Errorf("ToName mismatch: got %s, want 好きな人", found.ToName)
	}
}

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
	userRepo.Create(context.Background(), userA)

	// ユーザーB: 佐藤花子
	userB := &model.User{
		LineID:           "U_B",
		Name:             "佐藤花子",
		Birthday:         "1992-02-02",
		RegistrationStep: 1,
		RegisteredAt:     "2026-02-02 00:00:00",
		UpdatedAt:        "2026-02-02 00:00:00",
	}
	userRepo.Create(context.Background(), userB)

	// A → B を登録
	likeAtoB := &model.Like{
		FromUserID: "U_A",
		ToName:     "佐藤花子",
		ToBirthday: "1992-02-02",
		Matched:    false,
	}
	repo.Create(context.Background(), likeAtoB)

	// B → A を登録
	likeBtoA := &model.Like{
		FromUserID: "U_B",
		ToName:     "山田太郎",
		ToBirthday: "1990-01-01",
		Matched:    false,
	}
	repo.Create(context.Background(), likeBtoA)

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
	found, _ := repo.FindByFromUserID(context.Background(), "U_TEST")
	err := repo.UpdateMatched(context.Background(), found.ID, true)
	if err != nil {
		t.Errorf("UpdateMatched failed: %v", err)
	}

	// 更新されたか確認
	updated, _ := repo.FindByFromUserID(context.Background(), "U_TEST")
	if !updated.Matched {
		t.Error("Matched flag not updated")
	}
}
