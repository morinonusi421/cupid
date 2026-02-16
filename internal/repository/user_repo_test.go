package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/pkg/testutil"
)

// setupTestDB はテスト用のデータベースをセットアップする
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	return testutil.SetupTestDB(t, "test_repo_cupid.db", "../../db/schema.sql")
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		LineID:       "U123456789",
		Name:         "Test User",
		Birthday:     "1990-01-01",
		RegisteredAt: "2026-01-23 00:00:00",
		UpdatedAt:    "2026-01-23 00:00:00",
	}

	// ユーザーを作成
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 作成したユーザーを取得
	found, err := repo.FindByLineID(ctx, "U123456789")
	if err != nil {
		t.Fatalf("FindByLineID failed: %v", err)
	}

	if found == nil {
		t.Fatal("Expected user to be found, got nil")
	}

	if found.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got '%s'", found.Name)
	}

	if found.Birthday != "1990-01-01" {
		t.Errorf("Expected birthday '1990-01-01', got '%s'", found.Birthday)
	}
}

func TestUserRepository_FindByLineID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// 存在しないユーザーを検索
	found, err := repo.FindByLineID(ctx, "U_NOT_EXISTS")
	if err != nil {
		t.Fatalf("FindByLineID failed: %v", err)
	}

	if found != nil {
		t.Error("Expected user to be nil, but got non-nil")
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// ユーザーを作成
	user := &model.User{
		LineID:       "U987654321",
		Name:         "Original Name",
		Birthday:     "1985-05-15",
		RegisteredAt: "2026-01-23 00:00:00",
		UpdatedAt:    "2026-01-23 00:00:00",
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// ユーザー情報を更新
	user.Name = "Updated Name"

	err = repo.Update(ctx, user)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 更新後のユーザーを取得
	found, err := repo.FindByLineID(ctx, "U987654321")
	if err != nil {
		t.Fatalf("FindByLineID failed: %v", err)
	}

	if found.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", found.Name)
	}
}

func TestUserRepository_FindByNameAndBirthday(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	// テストユーザーを作成
	user := &model.User{
		LineID:   "U_FIND_TEST",
		Name:     "山田太郎",
		Birthday: "1990-01-01",
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	// 名前と誕生日で検索
	found, err := repo.FindByNameAndBirthday(context.Background(), "山田太郎", "1990-01-01")
	if err != nil {
		t.Errorf("FindByNameAndBirthday failed: %v", err)
	}
	if found == nil {
		t.Error("User not found")
	}
	if found.LineID != "U_FIND_TEST" {
		t.Errorf("LineID mismatch: got %s, want U_FIND_TEST", found.LineID)
	}

	// 存在しないユーザー
	notFound, err := repo.FindByNameAndBirthday(context.Background(), "存在しない", "2000-01-01")
	if err != nil {
		t.Errorf("FindByNameAndBirthday failed: %v", err)
	}
	if notFound != nil {
		t.Error("Expected nil for non-existent user")
	}
}
