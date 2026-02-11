-- +migrate Up

-- registration_stepカラムを削除（CrushName.Validで判定できるため不要）
ALTER TABLE users DROP COLUMN registration_step;

-- +migrate Down

-- ロールバック時はregistration_stepカラムを復元
ALTER TABLE users ADD COLUMN registration_step INTEGER NOT NULL DEFAULT 1;
