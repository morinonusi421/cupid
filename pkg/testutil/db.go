package testutil

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

// SetupTestDB はテスト用のデータベースをセットアップする
// schemaPath は db/schema.sql へのパス（相対パスまたは絶対パス）
func SetupTestDB(t *testing.T, dbPath string, schemaPath string) *sql.DB {
	t.Helper()

	// テスト終了時にDBファイルを削除
	t.Cleanup(func() {
		os.Remove(dbPath)
	})

	// DB を作成
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// 外部キー制約を有効化
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// スキーマファイルを読み込み
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to read schema file: %v", err)
	}

	// スキーマを実行
	if _, err := db.Exec(string(schema)); err != nil {
		db.Close()
		t.Fatalf("Failed to execute schema: %v", err)
	}

	return db
}
