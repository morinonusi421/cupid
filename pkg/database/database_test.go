package database

import (
	"os"
	"testing"
)

func TestInitDB(t *testing.T) {
	// テスト用のDBファイル名
	testDBPath := "test_cupid.db"
	defer os.Remove(testDBPath) // テスト終了後に削除

	// データベース初期化
	db, err := InitDB(testDBPath)
	if err != nil {
		t.Fatalf("initDB failed: %v", err)
	}
	defer db.Close()

	// Ping テスト
	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping() failed: %v", err)
	}

	// 外部キー制約が有効か確認
	var foreignKeys int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
	if err != nil {
		t.Fatalf("Failed to check foreign_keys: %v", err)
	}

	if foreignKeys != 1 {
		t.Errorf("Expected foreign_keys=1, got %d", foreignKeys)
	}
}
