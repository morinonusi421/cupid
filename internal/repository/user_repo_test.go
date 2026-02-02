package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/morinonusi421/cupid/internal/model"
	_ "modernc.org/sqlite"
)

// setupTestDB はテスト用のデータベースをセットアップする
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// テスト用の DB ファイル名
	testDBPath := "test_repo_cupid.db"
	t.Cleanup(func() {
		os.Remove(testDBPath)
	})

	// DB を作成
	db, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// 外部キー制約を有効化
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// スキーマを作成（migration SQLから抽出）
	schema := `
		CREATE TABLE users (
		  line_user_id TEXT PRIMARY KEY,
		  name TEXT NOT NULL DEFAULT '',
		  birthday TEXT NOT NULL DEFAULT '',
		  registration_step INTEGER NOT NULL DEFAULT 0,
		  temp_crush_name TEXT,
		  registered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX idx_users_name_birthday ON users(name, birthday);

		CREATE TABLE likes (
		  id INTEGER PRIMARY KEY AUTOINCREMENT,
		  from_user_id TEXT NOT NULL,
		  to_name TEXT NOT NULL,
		  to_birthday TEXT NOT NULL,
		  matched INTEGER NOT NULL DEFAULT 0,
		  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  FOREIGN KEY (from_user_id) REFERENCES users(line_user_id),
		  UNIQUE(from_user_id)
		);

		CREATE INDEX idx_likes_to_name_birthday ON likes(to_name, to_birthday);

		CREATE TRIGGER update_users_updated_at
		AFTER UPDATE ON users
		FOR EACH ROW
		BEGIN
		  UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE line_user_id = NEW.line_user_id;
		END;
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		LineID:           "U123456789",
		Name:             "Test User",
		Birthday:         "1990-01-01",
		RegistrationStep: 2,
		RegisteredAt:     "2026-01-23 00:00:00",
		UpdatedAt:        "2026-01-23 00:00:00",
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

	if found.RegistrationStep != 2 {
		t.Errorf("Expected registration_step 2, got %d", found.RegistrationStep)
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
		LineID:           "U987654321",
		Name:             "Original Name",
		Birthday:         "1985-05-15",
		RegistrationStep: 1,
		RegisteredAt:     "2026-01-23 00:00:00",
		UpdatedAt:        "2026-01-23 00:00:00",
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// ユーザー情報を更新
	user.Name = "Updated Name"
	user.RegistrationStep = 2

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

	if found.RegistrationStep != 2 {
		t.Errorf("Expected registration_step 2, got %d", found.RegistrationStep)
	}
}

func TestUserRepository_FindByNameAndBirthday(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	// テストユーザーを作成
	user := &model.User{
		LineID:           "U_FIND_TEST",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
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
