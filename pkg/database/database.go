package database

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

// InitDB はデータベース接続を初期化する
func InitDB(dbPath string) (*sql.DB, error) {
	// データベース接続
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// 接続確認
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	// 外部キー制約を有効化
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}

	// テーブル作成（存在しない場合のみ）
	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	log.Println("Database connected")
	return db, nil
}

// createTables はテーブルを作成する
func createTables(db *sql.DB) error {
	schema := `
-- ユーザーテーブル
CREATE TABLE IF NOT EXISTS users (
  line_user_id TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  birthday TEXT NOT NULL DEFAULT '',
  registration_step INTEGER NOT NULL DEFAULT 0,
  registered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 名前と誕生日の組み合わせで検索するためのインデックス
CREATE INDEX IF NOT EXISTS idx_users_name_birthday ON users(name, birthday);

-- 好きな人の登録テーブル
CREATE TABLE IF NOT EXISTS likes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  from_user_id TEXT NOT NULL,
  to_name TEXT NOT NULL,
  to_birthday TEXT NOT NULL,
  matched INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (from_user_id) REFERENCES users(line_user_id),
  UNIQUE(from_user_id)
);

-- マッチング検索用のインデックス
CREATE INDEX IF NOT EXISTS idx_likes_to_name_birthday ON likes(to_name, to_birthday);
`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// updated_at自動更新トリガー
	trigger := `
CREATE TRIGGER IF NOT EXISTS update_users_updated_at
AFTER UPDATE ON users
FOR EACH ROW
BEGIN
  UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE line_user_id = NEW.line_user_id;
END;
`

	if _, err := db.Exec(trigger); err != nil {
		return err
	}

	return nil
}
