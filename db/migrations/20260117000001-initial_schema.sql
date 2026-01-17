-- +migrate Up
-- ユーザーテーブル
-- registration_step: 0=awaiting_name, 1=awaiting_birthday, 2=completed, 3=awaiting_crush_birthday
CREATE TABLE users (
  line_user_id TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  birthday TEXT NOT NULL DEFAULT '',
  registration_step INTEGER NOT NULL DEFAULT 0,
  temp_crush_name TEXT,
  registered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 名前と誕生日の組み合わせで検索するためのインデックス
CREATE INDEX idx_users_name_birthday ON users(name, birthday);

-- 好きな人の登録テーブル
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

-- マッチング検索用のインデックス
CREATE INDEX idx_likes_to_name_birthday ON likes(to_name, to_birthday);

-- updated_at自動更新トリガー（SQLite + SQLBoilerの組み合わせで必要）
-- +migrate StatementBegin
CREATE TRIGGER update_users_updated_at
AFTER UPDATE ON users
FOR EACH ROW
BEGIN
  UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE line_user_id = NEW.line_user_id;
END;
-- +migrate StatementEnd

-- +migrate Down
DROP TRIGGER IF EXISTS update_users_updated_at;
DROP INDEX IF EXISTS idx_likes_to_name_birthday;
DROP TABLE IF EXISTS likes;
DROP INDEX IF EXISTS idx_users_name_birthday;
DROP TABLE IF EXISTS users;
