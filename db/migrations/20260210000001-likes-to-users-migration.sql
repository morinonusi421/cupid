-- +migrate Up

-- 1. likesテーブルを削除
DROP TABLE IF EXISTS likes;
DROP INDEX IF EXISTS idx_likes_to_name_birthday;

-- 2. 既存のusersテーブルを削除
DROP TRIGGER IF EXISTS update_users_updated_at;
DROP INDEX IF EXISTS idx_users_name_birthday;
DROP TABLE IF EXISTS users;

-- 3. 新しいusersテーブルを作成
CREATE TABLE users (
  line_user_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  birthday TEXT NOT NULL,
  registration_step INTEGER NOT NULL DEFAULT 1,
  crush_name TEXT,
  crush_birthday TEXT,
  matched_with_user_id TEXT,
  registered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (matched_with_user_id) REFERENCES users(line_user_id)
);

-- 4. インデックスを作成
CREATE INDEX idx_users_name_birthday ON users(name, birthday);
CREATE INDEX idx_users_crush ON users(crush_name, crush_birthday);

-- +migrate Down
DROP INDEX IF EXISTS idx_users_crush;
DROP INDEX IF EXISTS idx_users_name_birthday;
DROP TABLE IF EXISTS users;

