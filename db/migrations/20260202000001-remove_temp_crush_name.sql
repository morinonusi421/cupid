-- +migrate Up
-- temp_crush_nameカラムを削除（Web登録フォーム方式では不要）
ALTER TABLE users DROP COLUMN temp_crush_name;

-- +migrate Down
-- ロールバック時にカラムを再追加
ALTER TABLE users ADD COLUMN temp_crush_name TEXT;
