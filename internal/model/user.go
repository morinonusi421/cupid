package model

// User はユーザーのドメインモデル
type User struct {
	LineID           string
	Name             string
	Birthday         string
	RegistrationStep int
	TempCrushName    string
	RegisteredAt     string
	UpdatedAt        string
}
