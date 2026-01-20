package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

// initDB はデータベース接続を初期化する
func initDB(dbPath string) (*sql.DB, error) {
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

	log.Println("Database connected")
	return db, nil
}
