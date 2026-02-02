package model

// User はユーザーのドメインモデル
type User struct {
	LineID           string
	Name             string
	Birthday         string
	RegistrationStep int    // 0: 未登録, 1: 登録完了
	RegisteredAt     string
	UpdatedAt        string
}
