-- Cupid LINE Bot Database Schema
-- SQLite3

-- ユーザーテーブル
CREATE TABLE users (
  line_user_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  birthday TEXT NOT NULL,
  crush_name TEXT,
  crush_birthday TEXT,
  matched_with_user_id TEXT,
  registered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (matched_with_user_id) REFERENCES users(line_user_id)
);

-- 名前と誕生日の組み合わせで検索するためのインデックス
CREATE INDEX idx_users_name_birthday ON users(name, birthday);

-- 好きな人の検索用インデックス
CREATE INDEX idx_users_crush ON users(crush_name, crush_birthday);
