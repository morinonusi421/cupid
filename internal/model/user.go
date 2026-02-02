package model

// User はユーザーのドメインモデル
// TODO: Model層の充実化 - Validationメソッドとbusiness logicを追加
//   - IsValidName(), IsValidBirthday() などのValidation
//   - CanCompleteRegistration(), CompleteRegistration() などの状態遷移
//   - ドメインロジックをServiceから移動してModelに集約
type User struct {
	LineID           string
	Name             string
	Birthday         string
	RegistrationStep int // 0: 未登録, 1: 登録完了
	RegisteredAt     string
	UpdatedAt        string
}
