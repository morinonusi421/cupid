package database

import (
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

// InitDB はデータベース接続を初期化し、必要に応じてスキーマを作成する
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

	// スキーマが存在しない場合は作成
	if err := ensureSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	log.Println("Database connected")
	return db, nil
}

// ensureSchema はスキーマが存在しない場合に作成する
func ensureSchema(db *sql.DB) error {
	// usersテーブルの存在確認
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	if err == sql.ErrNoRows {
		// テーブルが存在しない場合、スキーマを作成
		log.Println("Creating database schema...")

		// スキーマファイルのパスを複数試す（実行ディレクトリによってパスが変わるため）
		schemaPaths := []string{
			"db/schema.sql",         // アプリケーションルートから実行
			"../../db/schema.sql",   // pkg/database/ から実行（テスト）
		}

		var schema []byte
		var readErr error
		for _, path := range schemaPaths {
			schema, readErr = os.ReadFile(path)
			if readErr == nil {
				break
			}
		}
		if readErr != nil {
			return readErr
		}

		if _, err := db.Exec(string(schema)); err != nil {
			return err
		}
		log.Println("Database schema created successfully")
	} else if err != nil {
		return err
	}
	return nil
}
