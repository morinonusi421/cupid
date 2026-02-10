# Likes Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Consolidate likes table into users table, eliminating redundancy and enabling proper re-registration flow with unmatch confirmation.

**Architecture:** Remove likes table entirely and add crush_name, crush_birthday, matched_with_user_id columns to users table. Replace LikeRepository methods with direct user table queries. Add unmatch confirmation flow with confirm_unmatch parameter.

**Tech Stack:** Go, SQLite, sql-migrate, sqlboiler, LINE Messaging API

---

## Task 1: Create Migration File

**Files:**
- Create: `db/migrations/20260210000001-likes-to-users-migration.sql`

**Step 1: Create migration file**

```bash
touch db/migrations/20260210000001-likes-to-users-migration.sql
```

**Step 2: Write migration SQL**

Write this exact SQL:

```sql
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

-- 5. トリガーを作成
CREATE TRIGGER update_users_updated_at
AFTER UPDATE ON users
FOR EACH ROW
BEGIN
  UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE line_user_id = NEW.line_user_id;
END;

-- +migrate Down
DROP TRIGGER IF EXISTS update_users_updated_at;
DROP INDEX IF EXISTS idx_users_crush;
DROP INDEX IF EXISTS idx_users_name_birthday;
DROP TABLE IF EXISTS users;
```

**Step 3: Commit migration file**

```bash
git add db/migrations/20260210000001-likes-to-users-migration.sql
git commit -m "feat: add migration to consolidate likes into users table"
```

---

## Task 2: Update User Model

**Files:**
- Modify: `internal/model/user.go:10-17`

**Step 1: Add new fields to User struct**

Replace the User struct (lines 10-17) with:

```go
type User struct {
	LineID             string
	Name               string
	Birthday           string
	RegistrationStep   int // 1: 登録完了（好きな人未登録）, 2: 好きな人登録完了
	CrushName          string
	CrushBirthday      string
	MatchedWithUserID  string
	RegisteredAt       string
	UpdatedAt          string
}
```

**Step 2: Add IsMatched domain method**

Add after CompleteCrushRegistration method:

```go
// IsMatched は、マッチング中かどうかを返す
func (u *User) IsMatched() bool {
	return u.MatchedWithUserID != ""
}
```

**Step 3: Update comment on line 14**

Change line 14 comment from:
```go
RegistrationStep int // 0: 未登録, 1: 登録完了
```

to:
```go
RegistrationStep int // 1: 登録完了（好きな人未登録）, 2: 好きな人登録完了
```

**Step 4: Commit model changes**

```bash
git add internal/model/user.go
git commit -m "feat: add crush and match fields to User model"
```

---

## Task 3: Update User Repository

**Files:**
- Modify: `internal/repository/user_repo.go:15-20` (interface)
- Modify: `internal/repository/user_repo.go:74-84` (entityToModel)
- Modify: `internal/repository/user_repo.go:86-96` (modelToEntity)

**Step 1: Add FindMatchingUser to interface**

Add to UserRepository interface after Update method (line 19):

```go
FindMatchingUser(ctx context.Context, currentUser *model.User) (*model.User, error)
```

**Step 2: Implement FindMatchingUser**

Add after Update method implementation (after line 72):

```go
// FindMatchingUser は相互にcrushしているユーザーを検索する
func (r *userRepository) FindMatchingUser(ctx context.Context, currentUser *model.User) (*model.User, error) {
	entityUser, err := entities.Users(
		qm.Where("name = ? AND birthday = ? AND crush_name = ? AND crush_birthday = ? AND matched_with_user_id IS NULL",
			currentUser.CrushName,
			currentUser.CrushBirthday,
			currentUser.Name,
			currentUser.Birthday,
		),
	).One(ctx, r.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return entityToModel(entityUser), nil
}
```

**Step 3: Update entityToModel**

Replace entityToModel function (lines 74-84) with:

```go
// entityToModel は entities.User を model.User に変換する
func entityToModel(e *entities.User) *model.User {
	return &model.User{
		LineID:            e.LineUserID.String,
		Name:              e.Name,
		Birthday:          e.Birthday,
		RegistrationStep:  int(e.RegistrationStep),
		CrushName:         e.CrushName.String,
		CrushBirthday:     e.CrushBirthday.String,
		MatchedWithUserID: e.MatchedWithUserID.String,
		RegisteredAt:      e.RegisteredAt,
		UpdatedAt:         e.UpdatedAt,
	}
}
```

**Step 4: Update modelToEntity**

Replace modelToEntity function (lines 86-96) with:

```go
// modelToEntity は model.User を entities.User に変換する
func modelToEntity(m *model.User) *entities.User {
	return &entities.User{
		LineUserID:        null.StringFrom(m.LineID),
		Name:              m.Name,
		Birthday:          m.Birthday,
		RegistrationStep:  int64(m.RegistrationStep),
		CrushName:         null.StringFrom(m.CrushName),
		CrushBirthday:     null.StringFrom(m.CrushBirthday),
		MatchedWithUserID: null.StringFrom(m.MatchedWithUserID),
		RegisteredAt:      m.RegisteredAt,
		UpdatedAt:         m.UpdatedAt,
	}
}
```

**Step 5: Commit repository changes**

```bash
git add internal/repository/user_repo.go
git commit -m "feat: add FindMatchingUser and update entity converters"
```

---

## Task 4: Run Migration and Regenerate Entities

**Files:**
- Modify: `entities/*.go` (auto-generated)

**Step 1: Run migration**

```bash
make migrate-up
```

Expected output:
```
Applied 1 migration(s).
```

**Step 2: Regenerate SQLBoiler entities**

```bash
make generate
```

Expected output:
```
Generating entities...
(sqlboiler output showing table scanning)
```

**Step 3: Verify entities were regenerated**

```bash
ls -la entities/users.go
```

Should show recent timestamp.

**Step 4: Commit regenerated entities**

```bash
git add entities/
git commit -m "chore: regenerate entities after migration"
```

---

## Task 5: Update Matching Service

**Files:**
- Modify: `internal/service/matching_service.go:11-13` (interface)
- Modify: `internal/service/matching_service.go:15-27` (struct and constructor)
- Modify: `internal/service/matching_service.go:29-80` (CheckAndUpdateMatch implementation)

**Step 1: Update MatchingService interface**

Replace interface (lines 11-13) with:

```go
type MatchingService interface {
	CheckAndUpdateMatch(ctx context.Context, currentUser *model.User) (matched bool, matchedUser *model.User, err error)
}
```

**Step 2: Update matchingService struct and constructor**

Replace struct and constructor (lines 15-27) with:

```go
type matchingService struct {
	userRepo repository.UserRepository
}

// NewMatchingService は MatchingService の新しいインスタンスを作成する
func NewMatchingService(userRepo repository.UserRepository) MatchingService {
	return &matchingService{
		userRepo: userRepo,
	}
}
```

**Step 3: Replace CheckAndUpdateMatch implementation**

Replace entire CheckAndUpdateMatch method (lines 29-80) with:

```go
// CheckAndUpdateMatch は相互マッチングをチェックし、マッチした場合は両方の matched_with_user_id を更新する
//
// 処理の流れ:
// 1. 相互にcrushしているユーザーを検索（FindMatchingUser）
// 2. 両方が真の場合、両方の matched_with_user_id を更新
//
// 戻り値:
//   - matched: マッチングが成立したかどうか
//   - matchedUser: マッチング相手のUserオブジェクト（マッチング成立時のみ）
//   - err: エラー（あれば）
func (s *matchingService) CheckAndUpdateMatch(
	ctx context.Context,
	currentUser *model.User,
) (matched bool, matchedUser *model.User, err error) {
	// 1. 相互にcrushしているユーザーを検索
	matchedUser, err = s.userRepo.FindMatchingUser(ctx, currentUser)
	if err != nil {
		return false, nil, err
	}

	// マッチング相手が見つからない場合
	if matchedUser == nil {
		return false, nil, nil
	}

	// 2. 両方の matched_with_user_id を更新
	currentUser.MatchedWithUserID = matchedUser.LineID
	matchedUser.MatchedWithUserID = currentUser.LineID

	if err := s.userRepo.Update(ctx, currentUser); err != nil {
		return false, nil, err
	}

	if err := s.userRepo.Update(ctx, matchedUser); err != nil {
		return false, nil, err
	}

	return true, matchedUser, nil
}
```

**Step 4: Commit matching service changes**

```bash
git add internal/service/matching_service.go
git commit -m "refactor: update matching service to use users table only"
```

---

## Task 6: Update User Service - RegisterCrush

**Files:**
- Modify: `internal/service/user_service.go:22-29` (userService struct)
- Modify: `internal/service/user_service.go:31-41` (NewUserService)
- Modify: `internal/service/user_service.go:96-163` (RegisterCrush)

**Step 1: Remove likeRepo from userService struct**

Replace userService struct (lines 22-29) with:

```go
type userService struct {
	userRepo        repository.UserRepository
	userLiffURL     string
	crushLiffURL    string
	matchingService MatchingService
	lineBotClient   linebot.Client
}
```

**Step 2: Update NewUserService constructor**

Replace NewUserService (lines 31-41) with:

```go
// NewUserService は UserService の新しいインスタンスを作成する
func NewUserService(userRepo repository.UserRepository, userLiffURL string, crushLiffURL string, matchingService MatchingService, lineBotClient linebot.Client) UserService {
	return &userService{
		userRepo:        userRepo,
		userLiffURL:     userLiffURL,
		crushLiffURL:    crushLiffURL,
		matchingService: matchingService,
		lineBotClient:   lineBotClient,
	}
}
```

**Step 3: Replace RegisterCrush implementation**

Replace entire RegisterCrush method (lines 96-163) with:

```go
// RegisterCrush は好きな人を登録し、マッチング判定を行う
func (s *userService) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string) (matched bool, matchedUserName string, err error) {
	// 1. 現在のユーザー情報を取得
	currentUser, err := s.userRepo.FindByLineID(ctx, userID)
	if err != nil {
		return false, "", err
	}
	if currentUser == nil {
		return false, "", fmt.Errorf("user not found: %s", userID)
	}

	// 2. 自己登録チェック（domain method使用）
	if currentUser.IsSamePerson(crushName, crushBirthday) {
		return false, "", fmt.Errorf("cannot register yourself")
	}

	// 3. 名前のバリデーション
	if valid, errMsg := model.IsValidName(crushName); !valid {
		return false, "", fmt.Errorf("%s", errMsg)
	}

	// 4. 好きな人を登録（usersテーブルに直接保存）
	currentUser.CrushName = crushName
	currentUser.CrushBirthday = crushBirthday

	// 5. RegistrationStepを2に更新（domain method使用）
	currentUser.CompleteCrushRegistration()

	if err := s.userRepo.Update(ctx, currentUser); err != nil {
		return false, "", err
	}

	// 6. マッチング判定（MatchingService に委譲）
	var matchedUser *model.User
	matched, matchedUser, err = s.matchingService.CheckAndUpdateMatch(ctx, currentUser)
	if err != nil {
		return false, "", fmt.Errorf("matching check failed: %w", err)
	}

	// マッチした場合、両方のユーザーにLINE通知を送信
	if matched {
		// 現在のユーザーに通知
		if err := s.sendMatchNotification(ctx, currentUser, matchedUser); err != nil {
			log.Printf("Failed to send match notification to %s: %v", currentUser.LineID, err)
			// エラーをログに記録するが、処理は継続
		}

		// 相手ユーザーに通知
		if err := s.sendMatchNotification(ctx, matchedUser, currentUser); err != nil {
			log.Printf("Failed to send match notification to %s: %v", matchedUser.LineID, err)
			// エラーをログに記録するが、処理は継続
		}
	} else {
		// マッチしなかった場合も登録完了を通知
		if err := s.sendCrushRegistrationComplete(ctx, currentUser); err != nil {
			log.Printf("Failed to send crush registration complete notification to %s: %v", currentUser.LineID, err)
			// エラーをログに記録するが、処理は継続
		}
	}

	matchedUserName = ""
	if matchedUser != nil {
		matchedUserName = matchedUser.Name
	}

	return matched, matchedUserName, nil
}
```

**Step 4: Commit RegisterCrush changes**

```bash
git add internal/service/user_service.go
git commit -m "refactor: update RegisterCrush to use users table directly"
```

---

## Task 7: Add Unmatch Methods to User Service

**Files:**
- Modify: `internal/service/user_service.go` (add after sendCrushRegistrationComplete)

**Step 1: Add unmatchUsers method**

Add after sendCrushRegistrationComplete method (around line 325):

```go
// unmatchUsers はマッチングを解除し、両方のユーザーに通知を送信する
func (s *userService) unmatchUsers(ctx context.Context, initiatorUser *model.User, partnerUserID string) error {
	// 相手のユーザー情報を取得
	partnerUser, err := s.userRepo.FindByLineID(ctx, partnerUserID)
	if err != nil {
		return fmt.Errorf("failed to find partner user: %w", err)
	}
	if partnerUser == nil {
		return fmt.Errorf("partner user not found: %s", partnerUserID)
	}

	// 両方の matched_with_user_id を NULL に
	initiatorUser.MatchedWithUserID = ""
	partnerUser.MatchedWithUserID = ""

	if err := s.userRepo.Update(ctx, initiatorUser); err != nil {
		return fmt.Errorf("failed to update initiator user: %w", err)
	}

	if err := s.userRepo.Update(ctx, partnerUser); err != nil {
		return fmt.Errorf("failed to update partner user: %w", err)
	}

	// 両方のユーザーに解除通知を送信
	if err := s.sendUnmatchNotification(ctx, initiatorUser, partnerUser, true); err != nil {
		log.Printf("Failed to send unmatch notification to initiator %s: %v", initiatorUser.LineID, err)
	}

	if err := s.sendUnmatchNotification(ctx, partnerUser, initiatorUser, false); err != nil {
		log.Printf("Failed to send unmatch notification to partner %s: %v", partnerUser.LineID, err)
	}

	return nil
}
```

**Step 2: Add sendUnmatchNotification method**

Add after unmatchUsers method:

```go
// sendUnmatchNotification はマッチング解除時にLINE Push通知を送信する
func (s *userService) sendUnmatchNotification(ctx context.Context, toUser *model.User, partnerUser *model.User, isInitiator bool) error {
	var reason string
	if isInitiator {
		reason = "あなたが情報を変更しました"
	} else {
		reason = "相手が情報を変更しました"
	}

	message := fmt.Sprintf("マッチングが解除されました。\n\n理由：%s\n相手：%s", reason, partnerUser.Name)

	request := &messaging_api.PushMessageRequest{
		To: toUser.LineID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: message,
			},
		},
		NotificationDisabled: false,
	}

	_, err := s.lineBotClient.PushMessage(request)
	return err
}
```

**Step 3: Commit unmatch methods**

```bash
git add internal/service/user_service.go
git commit -m "feat: add unmatch methods for match dissolution"
```

---

## Task 8: Add Confirm Unmatch to RegisterFromLIFF

**Files:**
- Modify: `internal/service/user_service.go:16-19` (UserService interface)
- Modify: `internal/service/user_service.go:73-94` (RegisterFromLIFF)
- Modify: `internal/service/user_service.go:191-214` (updateUserInfo)

**Step 1: Update UserService interface**

Replace RegisterFromLIFF method signature in interface (line 17) with:

```go
RegisterFromLIFF(ctx context.Context, userID, name, birthday string, confirmUnmatch bool) error
```

**Step 2: Update RegisterFromLIFF signature and add unmatch check**

Replace RegisterFromLIFF method (lines 73-94) with:

```go
// RegisterFromLIFF はLIFFフォームから送信された登録情報を保存する
func (s *userService) RegisterFromLIFF(ctx context.Context, userID, name, birthday string, confirmUnmatch bool) error {
	// 1. バリデーション
	if ok, errMsg := model.IsValidName(name); !ok {
		return fmt.Errorf("%s", errMsg)
	}

	// 2. ユーザー検索
	user, err := s.userRepo.FindByLineID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// 3. 初回登録 vs 再登録で分岐
	if user == nil {
		// 初回登録
		return s.registerNewUser(ctx, userID, name, birthday)
	} else {
		// 再登録（情報更新）
		return s.updateUserInfo(ctx, user, name, birthday, confirmUnmatch)
	}
}
```

**Step 3: Update updateUserInfo to handle unmatch**

Replace updateUserInfo method (lines 191-214) with:

```go
// updateUserInfo は再登録時に既存ユーザーの情報を更新する
func (s *userService) updateUserInfo(ctx context.Context, user *model.User, name, birthday string, confirmUnmatch bool) error {
	// 1. マッチング中かチェック
	if user.IsMatched() && !confirmUnmatch {
		return fmt.Errorf("matched_user_exists")
	}

	// 2. マッチング解除処理
	if user.IsMatched() && confirmUnmatch {
		if err := s.unmatchUsers(ctx, user, user.MatchedWithUserID); err != nil {
			log.Printf("Failed to unmatch users: %v", err)
			// エラーをログに記録するが、処理は継続（情報更新は実施）
		}
	}

	// 3. ユーザー情報を更新
	user.Name = name
	user.Birthday = birthday

	// 4. registration_step が 0 の場合のみ 1 に更新（通常はありえないが念のため）
	if user.RegistrationStep == 0 {
		user.CompleteUserRegistration()
	}

	// 5. DBに保存
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// 6. 更新完了メッセージを送信
	if err := s.sendUserInfoUpdateConfirmation(ctx, user); err != nil {
		log.Printf("Failed to send update confirmation to %s: %v", user.LineID, err)
		// エラーをログに記録するが、更新処理は成功として扱う
	}

	return nil
}
```

**Step 4: Commit RegisterFromLIFF unmatch changes**

```bash
git add internal/service/user_service.go
git commit -m "feat: add confirm_unmatch parameter to RegisterFromLIFF"
```

---

## Task 9: Add Confirm Unmatch to RegisterCrush

**Files:**
- Modify: `internal/service/user_service.go:18` (UserService interface)
- Modify: `internal/service/user_service.go:96-163` (RegisterCrush)

**Step 1: Update UserService interface**

Replace RegisterCrush method signature in interface (line 18) with:

```go
RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string, confirmUnmatch bool) (matched bool, matchedUserName string, err error)
```

**Step 2: Update RegisterCrush to handle unmatch**

Replace RegisterCrush method (lines 96-163) with:

```go
// RegisterCrush は好きな人を登録し、マッチング判定を行う
func (s *userService) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string, confirmUnmatch bool) (matched bool, matchedUserName string, err error) {
	// 1. 現在のユーザー情報を取得
	currentUser, err := s.userRepo.FindByLineID(ctx, userID)
	if err != nil {
		return false, "", err
	}
	if currentUser == nil {
		return false, "", fmt.Errorf("user not found: %s", userID)
	}

	// 2. マッチング中かチェック
	if currentUser.IsMatched() && !confirmUnmatch {
		return false, "", fmt.Errorf("matched_user_exists")
	}

	// 3. マッチング解除処理
	if currentUser.IsMatched() && confirmUnmatch {
		if err := s.unmatchUsers(ctx, currentUser, currentUser.MatchedWithUserID); err != nil {
			log.Printf("Failed to unmatch users: %v", err)
			// エラーをログに記録するが、処理は継続（Crush更新は実施）
		}
	}

	// 4. 自己登録チェック（domain method使用）
	if currentUser.IsSamePerson(crushName, crushBirthday) {
		return false, "", fmt.Errorf("cannot register yourself")
	}

	// 5. 名前のバリデーション
	if valid, errMsg := model.IsValidName(crushName); !valid {
		return false, "", fmt.Errorf("%s", errMsg)
	}

	// 6. 好きな人を登録（usersテーブルに直接保存）
	currentUser.CrushName = crushName
	currentUser.CrushBirthday = crushBirthday

	// 7. RegistrationStepを2に更新（domain method使用）
	currentUser.CompleteCrushRegistration()

	if err := s.userRepo.Update(ctx, currentUser); err != nil {
		return false, "", err
	}

	// 8. マッチング判定（MatchingService に委譲）
	var matchedUser *model.User
	matched, matchedUser, err = s.matchingService.CheckAndUpdateMatch(ctx, currentUser)
	if err != nil {
		return false, "", fmt.Errorf("matching check failed: %w", err)
	}

	// マッチした場合、両方のユーザーにLINE通知を送信
	if matched {
		// 現在のユーザーに通知
		if err := s.sendMatchNotification(ctx, currentUser, matchedUser); err != nil {
			log.Printf("Failed to send match notification to %s: %v", currentUser.LineID, err)
			// エラーをログに記録するが、処理は継続
		}

		// 相手ユーザーに通知
		if err := s.sendMatchNotification(ctx, matchedUser, currentUser); err != nil {
			log.Printf("Failed to send match notification to %s: %v", matchedUser.LineID, err)
			// エラーをログに記録するが、処理は継続
		}
	} else {
		// マッチしなかった場合も登録完了を通知
		if err := s.sendCrushRegistrationComplete(ctx, currentUser); err != nil {
			log.Printf("Failed to send crush registration complete notification to %s: %v", currentUser.LineID, err)
			// エラーをログに記録するが、処理は継続
		}
	}

	matchedUserName = ""
	if matchedUser != nil {
		matchedUserName = matchedUser.Name
	}

	return matched, matchedUserName, nil
}
```

**Step 3: Commit RegisterCrush unmatch changes**

```bash
git add internal/service/user_service.go
git commit -m "feat: add confirm_unmatch parameter to RegisterCrush"
```

---

## Task 10: Update Registration API Handler

**Files:**
- Modify: `internal/handler/registration_api.go:25-28` (RegisterRequest struct)
- Modify: `internal/handler/registration_api.go:30-77` (Register handler)

**Step 1: Add ConfirmUnmatch to RegisterRequest**

Replace RegisterRequest struct (lines 25-28) with:

```go
type RegisterRequest struct {
	Name           string `json:"name"`
	Birthday       string `json:"birthday"`
	ConfirmUnmatch bool   `json:"confirm_unmatch"`
}
```

**Step 2: Update Register handler to handle matched_user_exists error**

Replace Register handler (lines 30-77) with:

```go
func (h *RegistrationAPIHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Authorizationヘッダーからトークン取得
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "認証が必要です"})
		return
	}

	// "Bearer {token}" 形式からトークン抽出
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader { // Bearerプレフィックスがない
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "無効な認証形式です"})
		return
	}

	// トークン検証してuser_id取得
	userID, err := h.verifier.VerifyIDToken(token)
	if err != nil {
		log.Printf("Token verification failed: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "認証に失敗しました"})
		return
	}

	// リクエストボディからname, birthday, confirm_unmatchを取得
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// user_idはトークンから取得したものを使用
	if err := h.userService.RegisterFromLIFF(r.Context(), userID, req.Name, req.Birthday, req.ConfirmUnmatch); err != nil {
		log.Printf("Failed to register user: %v", err)

		// matched_user_existsエラーの場合は特別なレスポンス
		if err.Error() == "matched_user_exists" {
			// 相手の名前を取得するためにユーザー情報を取得
			// TODO: サービスからエラーと一緒に相手の名前を返すようにリファクタリング
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "matched_user_exists",
				"message": "現在マッチング中です。変更するとマッチングが解除されます。",
			})
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("Registration successful for user %s: name=%s, birthday=%s", userID, req.Name, req.Birthday)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

**Step 3: Commit registration API handler changes**

```bash
git add internal/handler/registration_api.go
git commit -m "feat: add confirm_unmatch handling to registration API"
```

---

## Task 11: Update Crush Registration API Handler

**Files:**
- Modify: `internal/handler/crush_registration_api.go:25-28` (RegisterCrushRequest struct)
- Modify: `internal/handler/crush_registration_api.go:36-112` (RegisterCrush handler)

**Step 1: Add ConfirmUnmatch to RegisterCrushRequest**

Replace RegisterCrushRequest struct (lines 25-28) with:

```go
type RegisterCrushRequest struct {
	CrushName      string `json:"crush_name"`
	CrushBirthday  string `json:"crush_birthday"`
	ConfirmUnmatch bool   `json:"confirm_unmatch"`
}
```

**Step 2: Update RegisterCrush handler**

Replace RegisterCrush handler (lines 36-112) with:

```go
func (h *CrushRegistrationAPIHandler) RegisterCrush(w http.ResponseWriter, r *http.Request) {
	// Authorizationヘッダーからトークンを取得
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "認証が必要です"})
		return
	}

	// "Bearer {token}" 形式からトークン抽出
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader { // Bearerプレフィックスがない
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "無効な認証形式です"})
		return
	}

	// トークン検証してuser_id取得
	userID, err := h.verifier.VerifyIDToken(token)
	if err != nil {
		log.Printf("Token verification failed: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "認証に失敗しました"})
		return
	}

	// リクエストボディをデコード
	var req RegisterCrushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// バリデーション
	if req.CrushName == "" || req.CrushBirthday == "" {
		log.Println("Missing crush_name or crush_birthday in request")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "crush_name and crush_birthday are required"})
		return
	}

	// サービス呼び出し（user_idはトークンから取得したものを使用）
	matched, matchedName, err := h.userService.RegisterCrush(r.Context(), userID, req.CrushName, req.CrushBirthday, req.ConfirmUnmatch)
	if err != nil {
		log.Printf("Failed to register crush: %v", err)

		// matched_user_existsエラーの場合は特別なレスポンス
		if err.Error() == "matched_user_exists" {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "matched_user_exists",
				"message": "現在マッチング中です。変更するとマッチングが解除されます。",
			})
			return
		}

		// 自己登録エラーの場合は400を返す
		if err.Error() == "cannot register yourself" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "自分自身は登録できません"})
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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

	log.Printf("Crush registration successful for user %s: crush=%s, matched=%t", userID, req.CrushName, matched)
}
```

**Step 3: Commit crush registration API handler changes**

```bash
git add internal/handler/crush_registration_api.go
git commit -m "feat: add confirm_unmatch handling to crush registration API"
```

---

## Task 12: Update Main to Remove LikeRepository

**Files:**
- Modify: `main.go` (remove like repository initialization and pass)

**Step 1: Find and remove like repository initialization**

Find the line that initializes likeRepo (should be around line 70-80):
```go
likeRepo := repository.NewLikeRepository(db)
```

Delete it.

**Step 2: Update MatchingService initialization**

Find the line that creates matchingService (should be around line 85):
```go
matchingService := service.NewMatchingService(userRepo, likeRepo)
```

Replace with:
```go
matchingService := service.NewMatchingService(userRepo)
```

**Step 3: Update UserService initialization**

Find the line that creates userService (should be around line 90):
```go
userService := service.NewUserService(userRepo, likeRepo, userLiffURL, crushLiffURL, matchingService, lineBotClient)
```

Replace with:
```go
userService := service.NewUserService(userRepo, userLiffURL, crushLiffURL, matchingService, lineBotClient)
```

**Step 4: Build to verify**

```bash
go build -o cupid
```

Expected: No compilation errors.

**Step 5: Commit main.go changes**

```bash
git add main.go
git commit -m "refactor: remove like repository from dependency injection"
```

---

## Task 13: Delete Like Model and Repository

**Files:**
- Delete: `internal/model/like.go`
- Delete: `internal/model/like_test.go`
- Delete: `internal/repository/like_repo.go`
- Delete: `internal/repository/like_repo_test.go`

**Step 1: Delete like model files**

```bash
git rm internal/model/like.go internal/model/like_test.go
```

**Step 2: Delete like repository files**

```bash
git rm internal/repository/like_repo.go internal/repository/like_repo_test.go
```

**Step 3: Build to verify**

```bash
go build -o cupid
```

Expected: No compilation errors (like references should be gone).

**Step 4: Commit deletions**

```bash
git commit -m "refactor: delete like model and repository (consolidated into users)"
```

---

## Task 14: Run All Tests

**Files:**
- No file changes

**Step 1: Run all tests**

```bash
make test
```

Expected output: Some tests may fail due to missing mocks or outdated test data. We'll fix them in next task.

**Step 2: Identify failing tests**

Look for test failures related to:
- Missing likeRepo parameter in service constructors
- Missing confirmUnmatch parameter in RegisterFromLIFF/RegisterCrush calls
- MatchingService interface changes

Note: Do NOT commit yet. We need to fix tests first.

---

## Task 15: Fix User Service Tests

**Files:**
- Modify: `internal/service/user_service_test.go`

**Step 1: Read current test file**

```bash
cat internal/service/user_service_test.go
```

**Step 2: Update all NewUserService calls**

Find all lines like:
```go
userService := service.NewUserService(mockUserRepo, mockLikeRepo, ...)
```

Replace with:
```go
userService := service.NewUserService(mockUserRepo, ...)
```

(Remove mockLikeRepo parameter)

**Step 3: Update all RegisterFromLIFF test calls**

Find all lines like:
```go
err := userService.RegisterFromLIFF(ctx, userID, name, birthday)
```

Replace with:
```go
err := userService.RegisterFromLIFF(ctx, userID, name, birthday, false)
```

(Add `false` for confirmUnmatch parameter in normal cases)

**Step 4: Update all RegisterCrush test calls**

Find all lines like:
```go
matched, name, err := userService.RegisterCrush(ctx, userID, crushName, crushBirthday)
```

Replace with:
```go
matched, name, err := userService.RegisterCrush(ctx, userID, crushName, crushBirthday, false)
```

(Add `false` for confirmUnmatch parameter in normal cases)

**Step 5: Run user service tests**

```bash
go test ./internal/service/user_service_test.go -v
```

Expected: Tests should pass now.

**Step 6: Commit test fixes**

```bash
git add internal/service/user_service_test.go
git commit -m "test: fix user service tests after like removal"
```

---

## Task 16: Fix Matching Service Tests

**Files:**
- Modify: `internal/service/matching_service_test.go`

**Step 1: Read current test file**

```bash
cat internal/service/matching_service_test.go
```

**Step 2: Update all NewMatchingService calls**

Find all lines like:
```go
matchingService := service.NewMatchingService(mockUserRepo, mockLikeRepo)
```

Replace with:
```go
matchingService := service.NewMatchingService(mockUserRepo)
```

**Step 3: Update all CheckAndUpdateMatch calls**

Find all lines like:
```go
matched, user, err := matchingService.CheckAndUpdateMatch(ctx, currentUser, currentLike)
```

Replace with:
```go
matched, user, err := matchingService.CheckAndUpdateMatch(ctx, currentUser)
```

(Remove currentLike parameter)

**Step 4: Update test logic to not create Like objects**

Replace any test code that creates Like objects with direct User updates:

Old:
```go
like := &model.Like{...}
```

New:
```go
currentUser.CrushName = "..."
currentUser.CrushBirthday = "..."
```

**Step 5: Remove like repository mocks**

Delete any MockLikeRepository definitions and expectations.

**Step 6: Run matching service tests**

```bash
go test ./internal/service/matching_service_test.go -v
```

Expected: Tests should pass now.

**Step 7: Commit test fixes**

```bash
git add internal/service/matching_service_test.go
git commit -m "test: fix matching service tests after like removal"
```

---

## Task 17: Fix Handler Tests

**Files:**
- Modify: `internal/handler/registration_api_test.go`
- Modify: `internal/handler/crush_registration_api_test.go`

**Step 1: Update MockUserServiceForAPI interface in registration_api_test.go**

Find RegisterFromLIFF method in MockUserServiceForAPI:
```go
func (m *MockUserServiceForAPI) RegisterFromLIFF(ctx context.Context, userID, name, birthday string) error {
```

Replace with:
```go
func (m *MockUserServiceForAPI) RegisterFromLIFF(ctx context.Context, userID, name, birthday string, confirmUnmatch bool) error {
```

**Step 2: Update test expectations in registration_api_test.go**

Find all mock expectations like:
```go
mockUserService.On("RegisterFromLIFF", mock.Anything, "U-test-user", "田中太郎", "2000-01-15").Return(nil)
```

Replace with:
```go
mockUserService.On("RegisterFromLIFF", mock.Anything, "U-test-user", "田中太郎", "2000-01-15", false).Return(nil)
```

**Step 3: Update MockUserServiceForAPI interface in crush_registration_api_test.go (if exists)**

Find RegisterCrush method:
```go
func (m *MockUserServiceForAPI) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string) (matched bool, matchedUserName string, err error) {
```

Replace with:
```go
func (m *MockUserServiceForAPI) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string, confirmUnmatch bool) (matched bool, matchedUserName string, err error) {
```

**Step 4: Update test expectations in crush_registration_api_test.go**

Find all mock expectations like:
```go
mockUserService.On("RegisterCrush", mock.Anything, userID, crushName, crushBirthday).Return(...)
```

Replace with:
```go
mockUserService.On("RegisterCrush", mock.Anything, userID, crushName, crushBirthday, false).Return(...)
```

**Step 5: Run handler tests**

```bash
go test ./internal/handler/... -v
```

Expected: Tests should pass now.

**Step 6: Commit test fixes**

```bash
git add internal/handler/registration_api_test.go internal/handler/crush_registration_api_test.go
git commit -m "test: fix handler tests after adding confirm_unmatch parameter"
```

---

## Task 18: Fix Webhook Handler Tests

**Files:**
- Modify: `internal/handler/webhook_test.go`

**Step 1: Update MockUserService interface**

Find RegisterFromLIFF method in MockUserService:
```go
func (m *MockUserService) RegisterFromLIFF(ctx context.Context, userID, name, birthday string) error {
```

Replace with:
```go
func (m *MockUserService) RegisterFromLIFF(ctx context.Context, userID, name, birthday string, confirmUnmatch bool) error {
```

**Step 2: Find RegisterCrush method**

Find RegisterCrush method:
```go
func (m *MockUserService) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string) (matched bool, matchedUserName string, err error) {
```

Replace with:
```go
func (m *MockUserService) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string, confirmUnmatch bool) (matched bool, matchedUserName string, err error) {
```

**Step 3: Run webhook tests**

```bash
go test ./internal/handler/webhook_test.go -v
```

Expected: Tests should pass now.

**Step 4: Commit test fixes**

```bash
git add internal/handler/webhook_test.go
git commit -m "test: fix webhook tests after interface changes"
```

---

## Task 19: Run All Tests Again

**Files:**
- No file changes

**Step 1: Run all tests**

```bash
make test
```

Expected output: All tests should pass now.

**Step 2: Verify test coverage**

```bash
go test -cover ./internal/...
```

Expected: Good coverage maintained after changes.

**Step 3: No commit needed**

Tests are passing. Ready for build and deploy.

---

## Task 20: Build and Deploy

**Files:**
- Binary: `cupid` (rebuilt)

**Step 1: Clean previous build**

```bash
rm -f cupid
```

**Step 2: Build new binary**

```bash
go build -o cupid
```

Expected: Successful build with no errors.

**Step 3: Test binary locally** (optional but recommended)

```bash
./cupid
```

Press Ctrl+C after verifying it starts without crashes.

**Step 4: Deploy to EC2**

```bash
make deploy
```

Expected output:
```
Building cupid binary...
Deploying to EC2...
Restarting service...
Deployment complete!
```

**Step 5: Verify service is running on EC2**

```bash
ssh cupid-bot "sudo systemctl status cupid"
```

Expected: Service should be active (running).

**Step 6: Check logs for errors**

```bash
ssh cupid-bot "sudo journalctl -u cupid -n 50 --no-pager"
```

Expected: No startup errors. Should see "Server started on :8080" or similar.

**Step 7: Commit deployment note** (optional)

```bash
git tag v1.0.0-likes-migration
git push origin v1.0.0-likes-migration
```

---

## Task 21: Verify Production Functionality

**Files:**
- No file changes

**Step 1: Test user registration via LIFF**

Open LIFF URL in browser and register a test user.

Expected: Registration succeeds without errors.

**Step 2: Test crush registration via LIFF**

Register a crush for the test user.

Expected: Registration succeeds. No match yet.

**Step 3: Test mutual matching**

Register second user and have them register first user as crush.

Expected: Match notification sent to both users.

**Step 4: Test re-registration**

Try to change first user's name via LIFF.

Expected: If not matched, update succeeds. If matched, error asking for confirmation.

**Step 5: Document any issues**

If issues found, create GitHub issues and fix in follow-up tasks.

---

## Success Criteria

- ✅ Migration applied successfully (likesテーブル削除、usersテーブル再作成)
- ✅ All tests passing
- ✅ Application builds without errors
- ✅ Service running on EC2
- ✅ User registration works
- ✅ Crush registration works
- ✅ Matching works with new schema
- ✅ Re-registration blocked when matched (requires confirmation)
- ✅ Unmatch flow works with notifications

## Rollback Plan

If deployment fails:

```bash
# 1. Rollback migration
make migrate-down

# 2. Checkout previous commit
git checkout HEAD~N  # where N is number of commits to rollback

# 3. Rebuild and redeploy
make deploy
```
