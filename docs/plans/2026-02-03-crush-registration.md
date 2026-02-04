# 好きな人登録機能 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** ユーザーが好きな人を登録し、相思相愛の場合に自動的にLINE通知を送る機能を実装する

**Architecture:** 既存の3層アーキテクチャ（handler → service → repository）を維持。マッチング判定はname+birthdayの完全一致で行い、マッチング時は両方のユーザーにLINE Push Messageを送信。

**Tech Stack:** Go 1.25, SQLite, SQLBoiler, LINE Messaging API, Vanilla JS

---

## Task 1: Like Model作成

**Files:**
- Create: `internal/model/like.go`
- Reference: `internal/model/user.go` (既存のモデル参照)

**Step 1: Likeモデル構造体を作成**

```go
package model

// Like は好きな人の登録情報を表す
type Like struct {
	ID           int64
	FromUserID   string // 登録したユーザーのLINE ID
	ToName       string // 好きな人の名前
	ToBirthday   string // 好きな人の誕生日 (YYYY-MM-DD)
	Matched      bool   // マッチングフラグ
	CreatedAt    string // 作成日時
}
```

**Step 2: entities.Likeからmodel.Likeへの変換関数を追加**

```go
// EntityToLike は entities.Like を model.Like に変換する
func EntityToLike(entity *entities.Like) *Like {
	if entity == nil {
		return nil
	}

	return &Like{
		ID:           entity.ID,
		FromUserID:   entity.FromUserID,
		ToName:       entity.ToName,
		ToBirthday:   entity.ToBirthday,
		Matched:      entity.Matched == 1,
		CreatedAt:    entity.CreatedAt,
	}
}
```

**Step 3: model.LikeからSQLBoiler用のカラム構造体への変換関数を追加**

```go
// LikeToColumns は model.Like を SQLBoiler の Columns 構造体に変換する
func LikeToColumns(like *Like) entities.M {
	matched := 0
	if like.Matched {
		matched = 1
	}

	return entities.M{
		entities.LikeColumns.FromUserID:  like.FromUserID,
		entities.LikeColumns.ToName:      like.ToName,
		entities.LikeColumns.ToBirthday:  like.ToBirthday,
		entities.LikeColumns.Matched:     matched,
	}
}
```

**Step 4: Commit**

```bash
git add internal/model/like.go
git commit -m "feat: add Like model with conversion functions

- Add Like struct representing crush registration
- Add EntityToLike conversion from SQLBoiler entity
- Add LikeToColumns conversion to SQLBoiler columns

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: LikeRepository実装

**Files:**
- Create: `internal/repository/like_repo.go`
- Create: `internal/repository/like_repo_test.go`

**Step 1: LikeRepositoryインターフェースを定義**

```go
package repository

import (
	"context"
	"github.com/morinonusi421/cupid/internal/model"
)

// LikeRepository は好きな人登録のデータアクセス層
type LikeRepository interface {
	// Create は新しい好きな人登録を作成（UPSERT）
	Create(ctx context.Context, like *model.Like) error

	// FindByFromUserID は登録者IDで検索
	FindByFromUserID(ctx context.Context, fromUserID string) (*model.Like, error)

	// FindMatchingLike は相互マッチングを検索
	// fromUserIDのユーザーが toName+toBirthday を登録しているか
	FindMatchingLike(ctx context.Context, fromUserID, toName, toBirthday string) (*model.Like, error)

	// UpdateMatched はマッチングフラグを更新
	UpdateMatched(ctx context.Context, id int64, matched bool) error
}
```

**Step 2: likeRepository構造体とコンストラクタを実装**

```go
type likeRepository struct {
	db *sql.DB
}

func NewLikeRepository(db *sql.DB) LikeRepository {
	return &likeRepository{db: db}
}
```

**Step 3: テストファイルを作成（Create メソッドのテスト）**

`internal/repository/like_repo_test.go`:

```go
package repository

import (
	"context"
	"testing"

	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/pkg/database"
)

func TestLikeRepository_Create(t *testing.T) {
	db := database.NewTestDB(t)
	defer db.Close()

	repo := NewLikeRepository(db)

	// テストユーザーを作成
	userRepo := NewUserRepository(db)
	user := &model.User{
		LineUserID:       "U_TEST_USER",
		Name:             "テストユーザー",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	// 好きな人を登録
	like := &model.Like{
		FromUserID:  "U_TEST_USER",
		ToName:      "好きな人",
		ToBirthday:  "1995-05-05",
		Matched:     false,
	}

	err := repo.Create(context.Background(), like)
	if err != nil {
		t.Errorf("Create failed: %v", err)
	}

	// 登録されたか確認
	found, err := repo.FindByFromUserID(context.Background(), "U_TEST_USER")
	if err != nil {
		t.Errorf("FindByFromUserID failed: %v", err)
	}
	if found == nil {
		t.Error("Like not found after Create")
	}
	if found.ToName != "好きな人" {
		t.Errorf("ToName mismatch: got %s, want 好きな人", found.ToName)
	}
}
```

**Step 4: テストを実行して失敗を確認**

```bash
go test ./internal/repository -run TestLikeRepository_Create -v
```

Expected: FAIL (Createメソッドが未実装)

**Step 5: Create メソッドを実装**

`internal/repository/like_repo.go`:

```go
func (r *likeRepository) Create(ctx context.Context, like *model.Like) error {
	// UPSERT: from_user_id が存在すれば UPDATE、なければ INSERT
	cols := model.LikeToColumns(like)

	// SQLBoiler の Upsert を使用
	err := entities.NewLike().Upsert(
		ctx,
		r.db,
		true, // updateOnConflict
		[]string{entities.LikeColumns.FromUserID}, // conflict columns
		boil.Whitelist(
			entities.LikeColumns.ToName,
			entities.LikeColumns.ToBirthday,
			entities.LikeColumns.Matched,
		),
		boil.Infer(),
	)

	return err
}
```

**Step 6: FindByFromUserID メソッドを実装**

```go
func (r *likeRepository) FindByFromUserID(ctx context.Context, fromUserID string) (*model.Like, error) {
	entity, err := entities.Likes(
		qm.Where("from_user_id = ?", fromUserID),
	).One(ctx, r.db)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return model.EntityToLike(entity), nil
}
```

**Step 7: テストを実行して成功を確認**

```bash
go test ./internal/repository -run TestLikeRepository_Create -v
```

Expected: PASS

**Step 8: FindMatchingLike のテストを追加**

```go
func TestLikeRepository_FindMatchingLike(t *testing.T) {
	db := database.NewTestDB(t)
	defer db.Close()

	repo := NewLikeRepository(db)
	userRepo := NewUserRepository(db)

	// ユーザーA: 山田太郎
	userA := &model.User{
		LineUserID:       "U_A",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), userA)

	// ユーザーB: 佐藤花子
	userB := &model.User{
		LineUserID:       "U_B",
		Name:             "佐藤花子",
		Birthday:         "1992-02-02",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), userB)

	// A → B を登録
	likeAtoB := &model.Like{
		FromUserID:  "U_A",
		ToName:      "佐藤花子",
		ToBirthday:  "1992-02-02",
		Matched:     false,
	}
	repo.Create(context.Background(), likeAtoB)

	// B → A を登録
	likeBtoA := &model.Like{
		FromUserID:  "U_B",
		ToName:      "山田太郎",
		ToBirthday:  "1990-01-01",
		Matched:     false,
	}
	repo.Create(context.Background(), likeBtoA)

	// B が A を登録しているか検索
	found, err := repo.FindMatchingLike(context.Background(), "U_B", "山田太郎", "1990-01-01")
	if err != nil {
		t.Errorf("FindMatchingLike failed: %v", err)
	}
	if found == nil {
		t.Error("Matching like not found")
	}
	if found.FromUserID != "U_B" {
		t.Errorf("FromUserID mismatch: got %s, want U_B", found.FromUserID)
	}
}
```

**Step 9: テストを実行して失敗を確認**

```bash
go test ./internal/repository -run TestLikeRepository_FindMatchingLike -v
```

Expected: FAIL (FindMatchingLikeメソッドが未実装)

**Step 10: FindMatchingLike メソッドを実装**

```go
func (r *likeRepository) FindMatchingLike(ctx context.Context, fromUserID, toName, toBirthday string) (*model.Like, error) {
	entity, err := entities.Likes(
		qm.Where("from_user_id = ? AND to_name = ? AND to_birthday = ?", fromUserID, toName, toBirthday),
	).One(ctx, r.db)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return model.EntityToLike(entity), nil
}
```

**Step 11: テストを実行して成功を確認**

```bash
go test ./internal/repository -run TestLikeRepository_FindMatchingLike -v
```

Expected: PASS

**Step 12: UpdateMatched のテストを追加**

```go
func TestLikeRepository_UpdateMatched(t *testing.T) {
	db := database.NewTestDB(t)
	defer db.Close()

	repo := NewLikeRepository(db)
	userRepo := NewUserRepository(db)

	user := &model.User{
		LineUserID:       "U_TEST",
		Name:             "テスト",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), user)

	like := &model.Like{
		FromUserID:  "U_TEST",
		ToName:      "相手",
		ToBirthday:  "1995-05-05",
		Matched:     false,
	}
	repo.Create(context.Background(), like)

	// matchedをtrueに更新
	found, _ := repo.FindByFromUserID(context.Background(), "U_TEST")
	err := repo.UpdateMatched(context.Background(), found.ID, true)
	if err != nil {
		t.Errorf("UpdateMatched failed: %v", err)
	}

	// 更新されたか確認
	updated, _ := repo.FindByFromUserID(context.Background(), "U_TEST")
	if !updated.Matched {
		t.Error("Matched flag not updated")
	}
}
```

**Step 13: テストを実行して失敗を確認**

```bash
go test ./internal/repository -run TestLikeRepository_UpdateMatched -v
```

Expected: FAIL (UpdateMatchedメソッドが未実装)

**Step 14: UpdateMatched メソッドを実装**

```go
func (r *likeRepository) UpdateMatched(ctx context.Context, id int64, matched bool) error {
	matchedInt := 0
	if matched {
		matchedInt = 1
	}

	_, err := entities.Likes(
		qm.Where("id = ?", id),
	).UpdateAll(ctx, r.db, entities.M{
		entities.LikeColumns.Matched: matchedInt,
	})

	return err
}
```

**Step 15: テストを実行して成功を確認**

```bash
go test ./internal/repository -run TestLikeRepository_UpdateMatched -v
```

Expected: PASS

**Step 16: すべてのテストを実行**

```bash
make test
```

Expected: All PASS

**Step 17: Commit**

```bash
git add internal/repository/like_repo.go internal/repository/like_repo_test.go
git commit -m "feat: implement LikeRepository with full test coverage

- Add Create (UPSERT), FindByFromUserID, FindMatchingLike, UpdateMatched
- Add comprehensive unit tests for all methods
- Test matching logic with dual registration scenario

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 3: UserRepository拡張（FindByNameAndBirthday）

**Files:**
- Modify: `internal/repository/user_repo.go`
- Modify: `internal/repository/user_repo_test.go`

**Step 1: UserRepositoryインターフェースにFindByNameAndBirthdayを追加**

```go
type UserRepository interface {
	FindByLineID(ctx context.Context, lineID string) (*model.User, error)
	FindByNameAndBirthday(ctx context.Context, name, birthday string) (*model.User, error) // 追加
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
}
```

**Step 2: テストを追加**

`internal/repository/user_repo_test.go`:

```go
func TestUserRepository_FindByNameAndBirthday(t *testing.T) {
	db := database.NewTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	// テストユーザーを作成
	user := &model.User{
		LineUserID:       "U_FIND_TEST",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	// 名前と誕生日で検索
	found, err := repo.FindByNameAndBirthday(context.Background(), "山田太郎", "1990-01-01")
	if err != nil {
		t.Errorf("FindByNameAndBirthday failed: %v", err)
	}
	if found == nil {
		t.Error("User not found")
	}
	if found.LineUserID != "U_FIND_TEST" {
		t.Errorf("LineUserID mismatch: got %s, want U_FIND_TEST", found.LineUserID)
	}

	// 存在しないユーザー
	notFound, err := repo.FindByNameAndBirthday(context.Background(), "存在しない", "2000-01-01")
	if err != nil {
		t.Errorf("FindByNameAndBirthday failed: %v", err)
	}
	if notFound != nil {
		t.Error("Expected nil for non-existent user")
	}
}
```

**Step 3: テストを実行して失敗を確認**

```bash
go test ./internal/repository -run TestUserRepository_FindByNameAndBirthday -v
```

Expected: FAIL (FindByNameAndBirthdayメソッドが未実装)

**Step 4: FindByNameAndBirthday メソッドを実装**

`internal/repository/user_repo.go`:

```go
// FindByNameAndBirthday は名前と誕生日でユーザーを検索する
func (r *userRepository) FindByNameAndBirthday(ctx context.Context, name, birthday string) (*model.User, error) {
	entityUser, err := entities.Users(
		qm.Where("name = ? AND birthday = ?", name, birthday),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return model.EntityToUser(entityUser), nil
}
```

**Step 5: テストを実行して成功を確認**

```bash
go test ./internal/repository -run TestUserRepository_FindByNameAndBirthday -v
```

Expected: PASS

**Step 6: Commit**

```bash
git add internal/repository/user_repo.go internal/repository/user_repo_test.go
git commit -m "feat: add FindByNameAndBirthday to UserRepository

- Add method to find users by name and birthday combination
- Add unit test covering found and not found cases
- Required for matching logic in crush registration

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: UserService.RegisterCrush実装

**Files:**
- Modify: `internal/service/user_service.go`
- Modify: `internal/service/user_service_test.go`

**Step 1: UserServiceにLikeRepositoryを追加**

```go
type userService struct {
	userRepo repository.UserRepository
	likeRepo repository.LikeRepository // 追加
	bot      *linebot.Client
}

func NewUserService(userRepo repository.UserRepository, likeRepo repository.LikeRepository, bot *linebot.Client) UserService {
	return &userService{
		userRepo: userRepo,
		likeRepo: likeRepo,
		bot:      bot,
	}
}
```

**Step 2: RegisterCrushメソッドのシグネチャを追加**

```go
type UserService interface {
	RegisterFromLIFF(ctx context.Context, lineID, name, birthday string) error
	RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string) (matched bool, matchedUserName string, err error) // 追加
	// ... 他のメソッド
}
```

**Step 3: テストを追加（マッチングなしケース）**

`internal/service/user_service_test.go`:

```go
func TestUserService_RegisterCrush_NoMatch(t *testing.T) {
	db := database.NewTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	likeRepo := repository.NewLikeRepository(db)
	mockBot := &linebot.Client{} // モック（実際には送信しない）

	service := NewUserService(userRepo, likeRepo, mockBot)

	// ユーザーA作成
	userA := &model.User{
		LineUserID:       "U_A",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), userA)

	// 好きな人を登録（相手は未登録）
	matched, matchedName, err := service.RegisterCrush(context.Background(), "U_A", "佐藤花子", "1992-02-02")
	if err != nil {
		t.Errorf("RegisterCrush failed: %v", err)
	}
	if matched {
		t.Error("Expected no match")
	}
	if matchedName != "" {
		t.Errorf("Expected empty matchedName, got %s", matchedName)
	}

	// DBに登録されたか確認
	like, _ := likeRepo.FindByFromUserID(context.Background(), "U_A")
	if like == nil {
		t.Error("Like not created")
	}
	if like.ToName != "佐藤花子" {
		t.Errorf("ToName mismatch: got %s", like.ToName)
	}
}
```

**Step 4: テストを実行して失敗を確認**

```bash
go test ./internal/service -run TestUserService_RegisterCrush_NoMatch -v
```

Expected: FAIL (RegisterCrushメソッドが未実装)

**Step 5: RegisterCrush メソッドを実装（基本ロジック）**

```go
func (s *userService) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string) (matched bool, matchedUserName string, err error) {
	// 1. 現在のユーザー情報を取得
	currentUser, err := s.userRepo.FindByLineID(ctx, userID)
	if err != nil {
		return false, "", err
	}
	if currentUser == nil {
		return false, "", fmt.Errorf("user not found: %s", userID)
	}

	// 2. 自己登録チェック
	if currentUser.Name == crushName && currentUser.Birthday == crushBirthday {
		return false, "", fmt.Errorf("cannot register yourself")
	}

	// 3. 好きな人を登録（UPSERT）
	like := &model.Like{
		FromUserID:  userID,
		ToName:      crushName,
		ToBirthday:  crushBirthday,
		Matched:     false,
	}
	if err := s.likeRepo.Create(ctx, like); err != nil {
		return false, "", err
	}

	// 4. マッチング判定は後で実装
	return false, "", nil
}
```

**Step 6: テストを実行して成功を確認**

```bash
go test ./internal/service -run TestUserService_RegisterCrush_NoMatch -v
```

Expected: PASS

**Step 7: 自己登録エラーのテストを追加**

```go
func TestUserService_RegisterCrush_SelfRegistrationError(t *testing.T) {
	db := database.NewTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	likeRepo := repository.NewLikeRepository(db)
	mockBot := &linebot.Client{}

	service := NewUserService(userRepo, likeRepo, mockBot)

	user := &model.User{
		LineUserID:       "U_SELF",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), user)

	// 自分自身を登録しようとする
	_, _, err := service.RegisterCrush(context.Background(), "U_SELF", "山田太郎", "1990-01-01")
	if err == nil {
		t.Error("Expected error for self-registration")
	}
	if err.Error() != "cannot register yourself" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}
```

**Step 8: テストを実行して成功を確認**

```bash
go test ./internal/service -run TestUserService_RegisterCrush_SelfRegistrationError -v
```

Expected: PASS

**Step 9: マッチング成立ケースのテストを追加**

```go
func TestUserService_RegisterCrush_Matched(t *testing.T) {
	db := database.NewTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	likeRepo := repository.NewLikeRepository(db)

	// LINE Bot のモック（実際には送信しない）
	// TODO: 本来はモックライブラリを使うべき
	mockBot := &linebot.Client{}

	service := NewUserService(userRepo, likeRepo, mockBot)

	// ユーザーA作成
	userA := &model.User{
		LineUserID:       "U_A",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), userA)

	// ユーザーB作成
	userB := &model.User{
		LineUserID:       "U_B",
		Name:             "佐藤花子",
		Birthday:         "1992-02-02",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), userB)

	// A → B を登録
	service.RegisterCrush(context.Background(), "U_A", "佐藤花子", "1992-02-02")

	// B → A を登録（マッチング成立）
	matched, matchedName, err := service.RegisterCrush(context.Background(), "U_B", "山田太郎", "1990-01-01")
	if err != nil {
		t.Errorf("RegisterCrush failed: %v", err)
	}
	if !matched {
		t.Error("Expected match")
	}
	if matchedName != "山田太郎" {
		t.Errorf("matchedName mismatch: got %s, want 山田太郎", matchedName)
	}

	// 両方のmatchedフラグが1になっているか確認
	likeA, _ := likeRepo.FindByFromUserID(context.Background(), "U_A")
	if !likeA.Matched {
		t.Error("UserA's like.matched not updated")
	}

	likeB, _ := likeRepo.FindByFromUserID(context.Background(), "U_B")
	if !likeB.Matched {
		t.Error("UserB's like.matched not updated")
	}
}
```

**Step 10: テストを実行して失敗を確認**

```bash
go test ./internal/service -run TestUserService_RegisterCrush_Matched -v
```

Expected: FAIL (マッチング判定ロジックが未実装)

**Step 11: RegisterCrush メソッドにマッチング判定ロジックを追加**

```go
func (s *userService) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string) (matched bool, matchedUserName string, err error) {
	// 1. 現在のユーザー情報を取得
	currentUser, err := s.userRepo.FindByLineID(ctx, userID)
	if err != nil {
		return false, "", err
	}
	if currentUser == nil {
		return false, "", fmt.Errorf("user not found: %s", userID)
	}

	// 2. 自己登録チェック
	if currentUser.Name == crushName && currentUser.Birthday == crushBirthday {
		return false, "", fmt.Errorf("cannot register yourself")
	}

	// 3. 好きな人を登録（UPSERT）
	like := &model.Like{
		FromUserID:  userID,
		ToName:      crushName,
		ToBirthday:  crushBirthday,
		Matched:     false,
	}
	if err := s.likeRepo.Create(ctx, like); err != nil {
		return false, "", err
	}

	// 4. マッチング判定
	// 4-1. 好きな人がusersテーブルに存在するか確認
	crushUser, err := s.userRepo.FindByNameAndBirthday(ctx, crushName, crushBirthday)
	if err != nil {
		return false, "", err
	}
	if crushUser == nil {
		// 相手が未登録 → マッチング不可
		return false, "", nil
	}

	// 4-2. 相手も自分を登録しているか確認
	reverseLike, err := s.likeRepo.FindMatchingLike(ctx, crushUser.LineUserID, currentUser.Name, currentUser.Birthday)
	if err != nil {
		return false, "", err
	}
	if reverseLike == nil {
		// 相手は自分を登録していない → マッチング不可
		return false, "", nil
	}

	// 5. マッチング成立！
	// 5-1. 自分のlikeレコードを取得してIDを確認
	currentLike, err := s.likeRepo.FindByFromUserID(ctx, userID)
	if err != nil {
		return false, "", err
	}

	// 5-2. 両方のmatchedフラグを1に更新
	if err := s.likeRepo.UpdateMatched(ctx, currentLike.ID, true); err != nil {
		return false, "", err
	}
	if err := s.likeRepo.UpdateMatched(ctx, reverseLike.ID, true); err != nil {
		return false, "", err
	}

	// 5-3. LINE通知を送信
	// TODO: PushMessage実装後に追加

	return true, crushName, nil
}
```

**Step 12: テストを実行して成功を確認**

```bash
go test ./internal/service -run TestUserService_RegisterCrush_Matched -v
```

Expected: PASS

**Step 13: すべてのテストを実行**

```bash
make test
```

Expected: All PASS

**Step 14: Commit**

```bash
git add internal/service/user_service.go internal/service/user_service_test.go
git commit -m "feat: implement RegisterCrush with matching logic

- Add RegisterCrush method to UserService
- Implement self-registration validation
- Implement matching logic with dual-check
- Update matched flags for both users on match
- Add comprehensive unit tests (no match, self-error, matched)
- TODO: Add LINE Push Message notification

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: CrushRegistrationAPIHandler実装

**Files:**
- Create: `internal/handler/crush_registration_api.go`
- Create: `internal/handler/crush_registration_api_test.go`

**Step 1: ハンドラー構造体とリクエスト/レスポンス型を定義**

```go
package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/morinonusi421/cupid/internal/service"
)

type CrushRegistrationAPIHandler struct {
	userService service.UserService
}

func NewCrushRegistrationAPIHandler(userService service.UserService) *CrushRegistrationAPIHandler {
	return &CrushRegistrationAPIHandler{
		userService: userService,
	}
}

type RegisterCrushRequest struct {
	UserID        string `json:"user_id"`
	CrushName     string `json:"crush_name"`
	CrushBirthday string `json:"crush_birthday"`
}

type RegisterCrushResponse struct {
	Status  string `json:"status"`
	Matched bool   `json:"matched"`
	Message string `json:"message"`
}
```

**Step 2: テストを追加（正常系: マッチングなし）**

`internal/handler/crush_registration_api_test.go`:

```go
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/internal/repository"
	"github.com/morinonusi421/cupid/internal/service"
	"github.com/morinonusi421/cupid/pkg/database"
)

func TestCrushRegistrationAPIHandler_RegisterCrush_NoMatch(t *testing.T) {
	db := database.NewTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	likeRepo := repository.NewLikeRepository(db)
	bot, _ := messaging_api.NewMessagingApiAPI("dummy_token")
	userService := service.NewUserService(userRepo, likeRepo, bot)
	handler := NewCrushRegistrationAPIHandler(userService)

	// テストユーザー作成
	user := &model.User{
		LineUserID:       "U_TEST",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), user)

	// リクエスト作成
	reqBody := RegisterCrushRequest{
		UserID:        "U_TEST",
		CrushName:     "佐藤花子",
		CrushBirthday: "1992-02-02",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/register-crush", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// レスポンス記録
	w := httptest.NewRecorder()

	// ハンドラー実行
	handler.RegisterCrush(w, req)

	// ステータスコード確認
	if w.Code != http.StatusOK {
		t.Errorf("Status code mismatch: got %d, want %d", w.Code, http.StatusOK)
	}

	// レスポンスボディ確認
	var resp RegisterCrushResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Status != "ok" {
		t.Errorf("Status mismatch: got %s, want ok", resp.Status)
	}
	if resp.Matched {
		t.Error("Expected matched=false")
	}
	if resp.Message != "登録しました。相手があなたを登録したらマッチングします。" {
		t.Errorf("Message mismatch: got %s", resp.Message)
	}
}
```

**Step 3: テストを実行して失敗を確認**

```bash
go test ./internal/handler -run TestCrushRegistrationAPIHandler_RegisterCrush_NoMatch -v
```

Expected: FAIL (RegisterCrushメソッドが未実装)

**Step 4: RegisterCrush ハンドラーメソッドを実装**

```go
func (h *CrushRegistrationAPIHandler) RegisterCrush(w http.ResponseWriter, r *http.Request) {
	// TODO: セキュリティ改善 - ワンタイムトークン方式に変更する

	// リクエストボディをデコード
	var req RegisterCrushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// バリデーション
	if req.UserID == "" {
		log.Println("Missing user_id in request")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_id is required"})
		return
	}
	if req.CrushName == "" || req.CrushBirthday == "" {
		log.Println("Missing crush_name or crush_birthday in request")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "crush_name and crush_birthday are required"})
		return
	}

	// サービス呼び出し
	matched, matchedName, err := h.userService.RegisterCrush(r.Context(), req.UserID, req.CrushName, req.CrushBirthday)
	if err != nil {
		log.Printf("Failed to register crush: %v", err)

		// 自己登録エラーの場合は400を返す
		if err.Error() == "cannot register yourself" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "自分自身は登録できません"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "registration failed"})
		return
	}

	// レスポンス作成
	var message string
	if matched {
		message = matchedName + "さんとマッチしました！💘"
	} else {
		message = "登録しました。相手があなたを登録したらマッチングします。"
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(RegisterCrushResponse{
		Status:  "ok",
		Matched: matched,
		Message: message,
	})

	log.Printf("Crush registration successful for user %s: crush=%s, matched=%t", req.UserID, req.CrushName, matched)
}
```

**Step 5: テストを実行して成功を確認**

```bash
go test ./internal/handler -run TestCrushRegistrationAPIHandler_RegisterCrush_NoMatch -v
```

Expected: PASS

**Step 6: 自己登録エラーのテストを追加**

```go
func TestCrushRegistrationAPIHandler_RegisterCrush_SelfRegistrationError(t *testing.T) {
	db := database.NewTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	likeRepo := repository.NewLikeRepository(db)
	bot, _ := messaging_api.NewMessagingApiAPI("dummy_token")
	userService := service.NewUserService(userRepo, likeRepo, bot)
	handler := NewCrushRegistrationAPIHandler(userService)

	user := &model.User{
		LineUserID:       "U_SELF",
		Name:             "山田太郎",
		Birthday:         "1990-01-01",
		RegistrationStep: 1,
	}
	userRepo.Create(context.Background(), user)

	reqBody := RegisterCrushRequest{
		UserID:        "U_SELF",
		CrushName:     "山田太郎",
		CrushBirthday: "1990-01-01",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/register-crush", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.RegisterCrush(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code mismatch: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "自分自身は登録できません" {
		t.Errorf("Error message mismatch: got %s", resp["error"])
	}
}
```

**Step 7: テストを実行して成功を確認**

```bash
go test ./internal/handler -run TestCrushRegistrationAPIHandler_RegisterCrush_SelfRegistrationError -v
```

Expected: PASS

**Step 8: すべてのテストを実行**

```bash
make test
```

Expected: All PASS

**Step 9: Commit**

```bash
git add internal/handler/crush_registration_api.go internal/handler/crush_registration_api_test.go
git commit -m "feat: implement CrushRegistrationAPIHandler

- Add POST /api/register-crush endpoint
- Implement request validation (user_id, name, birthday)
- Handle self-registration error with 400 status
- Return matched status and message in response
- Add unit tests for no-match and self-error cases

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 6: フロントエンド実装

**Files:**
- Create: `static/crush/register.html`
- Create: `static/crush/register.css`
- Create: `static/crush/register.js`

**Step 1: register.html を作成**

`static/crush/register.html`:

```html
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Cupid - 好きな人登録</title>
    <link rel="stylesheet" href="register.css">
</head>
<body>
    <div class="container">
        <h1>💘 Cupid</h1>
        <p class="subtitle">好きな人を登録</p>

        <form id="register-form">
            <div class="form-group">
                <label for="name">好きな人の名前</label>
                <input
                    type="text"
                    id="name"
                    placeholder="例: 山田太郎"
                    maxlength="50"
                    required
                >
            </div>

            <div class="form-group">
                <label for="birthday">好きな人の誕生日</label>
                <input
                    type="date"
                    id="birthday"
                    required
                >
            </div>

            <button type="submit" id="submit-button">登録する</button>
        </form>

        <div id="loading" style="display: none;">
            <p>登録中...</p>
        </div>

        <div id="message" style="display: none;"></div>
    </div>

    <script src="register.js"></script>
</body>
</html>
```

**Step 2: register.css を作成（liff/register.cssをコピーして微調整）**

```bash
cp static/liff/register.css static/crush/register.css
```

**Step 3: register.js を作成**

`static/crush/register.js`:

```javascript
// URLパラメータからuser_idを取得
function getUserIdFromURL() {
    const params = new URLSearchParams(window.location.search);
    return params.get('user_id');
}

// フォーム送信処理
document.getElementById('register-form').addEventListener('submit', async (e) => {
    e.preventDefault();

    const name = document.getElementById('name').value.trim();
    const birthday = document.getElementById('birthday').value;
    const userId = getUserIdFromURL();

    if (!userId) {
        showMessage('エラー: ユーザーIDが取得できませんでした', 'error');
        return;
    }

    if (!name || !birthday) {
        showMessage('名前と誕生日を入力してください', 'error');
        return;
    }

    // UI更新
    document.getElementById('submit-button').disabled = true;
    document.getElementById('loading').style.display = 'block';
    document.getElementById('message').style.display = 'none';

    try {
        const response = await fetch('/api/register-crush', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                user_id: userId,
                crush_name: name,
                crush_birthday: birthday
            })
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error || '登録に失敗しました');
        }

        // 成功
        showMessage(data.message, data.matched ? 'matched' : 'success');

        // マッチングした場合は3秒後にLINEに戻る
        if (data.matched) {
            setTimeout(() => {
                if (window.liff && window.liff.isInClient()) {
                    window.liff.closeWindow();
                }
            }, 3000);
        }

    } catch (error) {
        console.error('Registration error:', error);
        showMessage(error.message, 'error');
    } finally {
        document.getElementById('submit-button').disabled = false;
        document.getElementById('loading').style.display = 'none';
    }
});

// メッセージ表示
function showMessage(text, type) {
    const messageEl = document.getElementById('message');
    messageEl.textContent = text;
    messageEl.className = type;
    messageEl.style.display = 'block';
}
```

**Step 4: Commit**

```bash
git add static/crush/
git commit -m "feat: add crush registration frontend

- Add register.html with name and birthday form
- Copy and reuse register.css from liff directory
- Add register.js with API integration
- Handle matched/no-match responses with UI feedback
- Auto-close LIFF window on match after 3 seconds

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 7: main.goにルート追加

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: main.go を確認**

```bash
cat cmd/server/main.go | head -50
```

**Step 2: LikeRepositoryとCrushRegistrationAPIHandlerを初期化**

`cmd/server/main.go` の適切な場所に追加：

```go
// Repository層
userRepo := repository.NewUserRepository(db)
likeRepo := repository.NewLikeRepository(db) // 追加

// Service層
userService := service.NewUserService(userRepo, likeRepo, bot) // likeRepoを追加

// Handler層
registrationAPIHandler := handler.NewRegistrationAPIHandler(userService)
crushRegistrationAPIHandler := handler.NewCrushRegistrationAPIHandler(userService) // 追加
```

**Step 3: /api/register-crush ルートを追加**

```go
// API routes
http.HandleFunc("/api/register", registrationAPIHandler.Register)
http.HandleFunc("/api/register-crush", crushRegistrationAPIHandler.RegisterCrush) // 追加
```

**Step 4: ローカルで起動してテスト**

```bash
go run cmd/server/main.go
```

別ターミナルで：

```bash
# ユーザー登録
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"user_id":"U_TEST","name":"山田太郎","birthday":"1990-01-01"}'

# 好きな人登録
curl -X POST http://localhost:8080/api/register-crush \
  -H "Content-Type: application/json" \
  -d '{"user_id":"U_TEST","crush_name":"佐藤花子","crush_birthday":"1992-02-02"}'
```

Expected: `{"status":"ok","matched":false,"message":"登録しました。相手があなたを登録したらマッチングします。"}`

**Step 5: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire up crush registration API in main

- Initialize LikeRepository
- Pass likeRepo to UserService
- Initialize CrushRegistrationAPIHandler
- Add /api/register-crush route

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Nginx設定追加

**Files:**
- Modify: `nginx/cupid.conf`

**Step 1: /crush/ パスの設定を追加**

`nginx/cupid.conf`:

```nginx
# 既存の /liff/ の下に追加
location /crush/ {
    alias /home/ec2-user/cupid/static/crush/;
    try_files $uri $uri/ =404;
}
```

**Step 2: Commit**

```bash
git add nginx/cupid.conf
git commit -m "feat: add nginx config for /crush/ path

- Serve static/crush/ files at /crush/ path
- Matches existing /liff/ pattern

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 9: LINE Bot メッセージハンドラー更新

**Files:**
- Modify: `internal/handler/message_handler.go`

**Step 1: RegistrationStep = 1 の場合に好きな人登録URLを送る処理を追加**

`internal/handler/message_handler.go` の適切な場所に追加：

```go
// RegistrationStep = 1（ユーザー登録完了済み）の場合
if user.RegistrationStep == 1 {
	crushRegisterURL := fmt.Sprintf("https://cupid-linebot.click/crush/register.html?user_id=%s", event.Source.UserId)

	replyMessage := fmt.Sprintf(
		"次に、好きな人を登録してください💘\n\n%s",
		crushRegisterURL,
	)

	if _, err := bot.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: event.ReplyToken,
			Messages: []messaging_api.MessageInterface{
				&messaging_api.TextMessage{
					Text: replyMessage,
				},
			},
		},
	); err != nil {
		log.Printf("Failed to reply message: %v", err)
	}
	return
}
```

**Step 2: Commit**

```bash
git add internal/handler/message_handler.go
git commit -m "feat: send crush registration URL when user is registered

- Check RegistrationStep = 1 (user registration complete)
- Reply with crush registration URL
- Guide user to next step

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 10: デプロイとテスト

**Files:**
- None (deployment)

**Step 1: すべてのテストを実行**

```bash
make test
```

Expected: All PASS

**Step 2: デプロイ**

```bash
make deploy
```

**Step 3: EC2でNginx設定をリロード**

```bash
ssh cupid-bot
cd ~/cupid
git pull
sudo nginx -t
sudo systemctl reload nginx
```

**Step 4: サービス再起動**

```bash
sudo systemctl restart cupid
sudo systemctl status cupid
```

**Step 5: 動作確認（LINE Botで実際にテスト）**

1. LINE Botにメッセージ送信
2. ユーザー登録URLが届く → 登録
3. 好きな人登録URLが届く → 登録
4. 相手も登録すればマッチング通知

**Step 6: Commit**

```bash
git add .
git commit -m "chore: deploy crush registration feature

- All tests passing
- Deployed to EC2
- Nginx config updated
- Service restarted

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## TODO（将来の改善）

- [ ] LINE Push Message実装（現在はTODO）
- [ ] 再登録機能（好きな人の変更）
- [ ] ワンタイムトークン方式でセキュリティ改善
- [ ] マッチング履歴の表示
- [ ] マッチング解除機能

---

## 完了

全てのタスクが完了したら、@superpowers:finishing-a-development-branch を使用してブランチを統合してください。
