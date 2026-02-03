# モデル層リファクタリング設計書

**作成日**: 2026-02-03
**目的**: 薄いモデル層を充実させ、役割分担とDIを改善する

## 概要

現状のコードは「貧血ドメインモデル（Anemic Domain Model）」問題を抱えている。モデルがデータ構造のみで、ビジネスロジックがService層に散在している。

このリファクタリングでは、以下を実現する：
1. モデルにドメインロジックを追加
2. MatchingServiceを導入して責務を分離
3. DIを改善してテスタビリティ向上

**方針**: オーバーエンジニアリングを避け、シンプルで実用的な設計を目指す。

---

## 設計方針

### 選択した方針

1. **ハイブリッドアプローチ（Value Object最小限）**
   - `Name`, `Birthday`などを全てstructにはしない
   - モデルにメソッドを追加するだけで十分

2. **MatchingServiceの導入**
   - 複数モデルにまたがるロジックを分離
   - テスタビリティ向上

3. **Aggregate境界は現状維持**
   - UserとLikeは別々のAggregate
   - 既存のRepository構造を維持

4. **serviceディレクトリは1つ**
   - `domain/service`と`service`を分けない
   - Goの慣習に沿ってシンプルに

---

## ディレクトリ構造

### 変更前
```
internal/
├── model/           # 薄い（データ構造のみ）
├── repository/
├── service/         # 肥大化
└── handler/
```

### 変更後
```
internal/
├── model/
│   ├── user.go      # + ドメインメソッド
│   └── like.go      # + ドメインメソッド
├── repository/      # 変更なし
├── service/
│   ├── user_service.go       # 軽量化（オーケストレーション）
│   └── matching_service.go   # NEW（マッチングロジック）
└── handler/         # 変更なし
```

---

## 役割分担

### Model層（internal/model/）
**責務**: 単一モデル内のドメインロジック

- バリデーション
- 状態遷移
- ドメインルール判定

**例**:
- `User.IsSamePerson(name, birthday)` - 自己登録チェック
- `User.CompleteCrushRegistration()` - 状態遷移
- `Like.MarkAsMatched()` - マッチング状態更新

### MatchingService（internal/service/matching_service.go）
**責務**: 複数モデルにまたがるドメインロジック

- UserとLikeを協調させる
- マッチング判定アルゴリズム
- 双方向チェックロジック

**例**:
- `CheckAndUpdateMatch(user, like)` - マッチング判定と状態更新

### UserService（internal/service/user_service.go）
**責務**: ユースケースのオーケストレーション

- Repository呼び出し
- MatchingServiceへの委譲
- トランザクション管理
- エラーハンドリング

**例**:
- `RegisterCrush()` - フロー全体の制御

### Repository層（internal/repository/）
**責務**: データアクセスのみ

- 変更なし

---

## 詳細設計

### 1. User Model

```go
// internal/model/user.go

type User struct {
    LineID           string
    Name             string
    Birthday         string
    RegistrationStep int
    RegisteredAt     string
    UpdatedAt        string
}

// IsSamePerson は自己登録チェック用
func (u *User) IsSamePerson(name, birthday string) bool {
    return u.Name == name && u.Birthday == birthday
}

// CanRegisterCrush はcrush登録可能かチェック
func (u *User) CanRegisterCrush() bool {
    return u.RegistrationStep >= 1
}

// CompleteCrushRegistration はcrush登録完了時の状態遷移
func (u *User) CompleteCrushRegistration() {
    u.RegistrationStep = 2
}

// IsRegistrationComplete はユーザー登録が完了しているかチェック
func (u *User) IsRegistrationComplete() bool {
    return u.RegistrationStep >= 1
}

// CompleteUserRegistration はユーザー登録完了時の状態遷移
func (u *User) CompleteUserRegistration() {
    u.RegistrationStep = 1
}
```

---

### 2. Like Model

```go
// internal/model/like.go

type Like struct {
    ID         int64
    FromUserID string
    ToName     string
    ToBirthday string
    Matched    bool
    CreatedAt  string
}

// MarkAsMatched はマッチング成立時にフラグを立てる
func (l *Like) MarkAsMatched() {
    l.Matched = true
}

// IsMatched はマッチング済みかチェック
func (l *Like) IsMatched() bool {
    return l.Matched
}

// MatchesCrush は指定された名前・誕生日と一致するかチェック
func (l *Like) MatchesCrush(name, birthday string) bool {
    return l.ToName == name && l.ToBirthday == birthday
}

// NewLike はLikeを生成するファクトリ関数
func NewLike(fromUserID, toName, toBirthday string) *Like {
    return &Like{
        FromUserID:  fromUserID,
        ToName:      toName,
        ToBirthday:  toBirthday,
        Matched:     false,
    }
}
```

---

### 3. MatchingService

```go
// internal/service/matching_service.go

package service

import (
    "context"
    "fmt"

    "github.com/morinonusi421/cupid/internal/model"
    "github.com/morinonusi421/cupid/internal/repository"
)

// MatchingService はマッチング判定ロジックを提供する
type MatchingService struct {
    userRepo repository.UserRepository
    likeRepo repository.LikeRepository
}

// NewMatchingService はMatchingServiceの新しいインスタンスを作成
func NewMatchingService(
    userRepo repository.UserRepository,
    likeRepo repository.LikeRepository,
) *MatchingService {
    return &MatchingService{
        userRepo: userRepo,
        likeRepo: likeRepo,
    }
}

// CheckAndUpdateMatch はマッチング判定と状態更新を行う
//
// 処理フロー:
// 1. crushがusersテーブルに存在するか確認
// 2. crushも自分を登録しているか確認（双方向チェック）
// 3. マッチング成立なら両方のmatchedフラグを更新
//
// 戻り値:
// - matched: マッチング成立したか
// - matchedUserName: マッチングした相手の名前
// - error: エラー
func (m *MatchingService) CheckAndUpdateMatch(
    ctx context.Context,
    currentUser *model.User,
    currentLike *model.Like,
) (matched bool, matchedUserName string, err error) {

    // 1. crushがusersテーブルに存在するか確認
    crushUser, err := m.userRepo.FindByNameAndBirthday(
        ctx,
        currentLike.ToName,
        currentLike.ToBirthday,
    )
    if err != nil {
        return false, "", fmt.Errorf("failed to find crush user: %w", err)
    }
    if crushUser == nil {
        // 相手が未登録 → マッチング不可
        return false, "", nil
    }

    // 2. crushも自分を登録しているか確認（双方向チェック）
    reverseLike, err := m.likeRepo.FindMatchingLike(
        ctx,
        crushUser.LineID,
        currentUser.Name,
        currentUser.Birthday,
    )
    if err != nil {
        return false, "", fmt.Errorf("failed to find reverse like: %w", err)
    }
    if reverseLike == nil {
        // 相手は自分を登録していない → マッチング不可
        return false, "", nil
    }

    // 3. マッチング成立！両方のmatchedフラグを更新
    currentLike.MarkAsMatched()
    reverseLike.MarkAsMatched()

    if err := m.likeRepo.UpdateMatched(ctx, currentLike.ID, true); err != nil {
        return false, "", fmt.Errorf("failed to update current like: %w", err)
    }
    if err := m.likeRepo.UpdateMatched(ctx, reverseLike.ID, true); err != nil {
        return false, "", fmt.Errorf("failed to update reverse like: %w", err)
    }

    return true, currentLike.ToName, nil
}
```

---

### 4. UserService（リファクタ後）

```go
// internal/service/user_service.go（一部抜粋）

type userService struct {
    userRepo        repository.UserRepository
    likeRepo        repository.LikeRepository
    matchingService *MatchingService  // NEW
    liffVerifier    *liff.Verifier
    liffRegisterURL string
}

func NewUserService(
    userRepo repository.UserRepository,
    likeRepo repository.LikeRepository,
    matchingService *MatchingService,  // NEW
    liffVerifier *liff.Verifier,
    liffRegisterURL string,
) UserService {
    return &userService{
        userRepo:        userRepo,
        likeRepo:        likeRepo,
        matchingService: matchingService,  // NEW
        liffVerifier:    liffVerifier,
        liffRegisterURL: liffRegisterURL,
    }
}

// RegisterCrush は好きな人を登録し、マッチング判定を行う
func (s *userService) RegisterCrush(
    ctx context.Context,
    userID, crushName, crushBirthday string,
) (matched bool, matchedUserName string, err error) {

    // 1. ユーザー取得
    currentUser, err := s.userRepo.FindByLineID(ctx, userID)
    if err != nil {
        return false, "", err
    }
    if currentUser == nil {
        return false, "", fmt.Errorf("user not found: %s", userID)
    }

    // 2. 自己登録チェック（Modelメソッド使用）
    if currentUser.IsSamePerson(crushName, crushBirthday) {
        return false, "", fmt.Errorf("cannot register yourself")
    }

    // 3. Like登録（ファクトリ関数使用）
    like := model.NewLike(userID, crushName, crushBirthday)
    if err := s.likeRepo.Create(ctx, like); err != nil {
        return false, "", err
    }

    // 4. 状態遷移（Modelメソッド使用）
    currentUser.CompleteCrushRegistration()
    if err := s.userRepo.Update(ctx, currentUser); err != nil {
        return false, "", err
    }

    // 5. マッチング判定（MatchingServiceに委譲）
    matched, name, err := s.matchingService.CheckAndUpdateMatch(ctx, currentUser, like)
    if err != nil {
        return false, "", err
    }

    return matched, name, nil
}
```

---

### 5. Dependency Injection（main.go）

```go
// cmd/server/main.go（変更部分のみ）

// Repository層
userRepo := repository.NewUserRepository(db)
likeRepo := repository.NewLikeRepository(db)

// Service層
matchingService := service.NewMatchingService(userRepo, likeRepo)  // NEW
userService := service.NewUserService(
    userRepo,
    likeRepo,
    matchingService,  // NEW
    liffVerifier,
    registerURL,
)

// Handler層（変更なし）
webhookHandler := handler.NewWebhookHandler(channelSecret, lineBotClient, userService)
```

---

## テスト戦略

### テスト対象

1. **Model層のテスト（NEW）**
   - `User.IsSamePerson()`
   - `User.CanRegisterCrush()`
   - `Like.MarkAsMatched()`
   - など

2. **MatchingServiceのテスト（NEW）**
   - マッチングなしケース
   - マッチング成立ケース
   - エラーケース

3. **UserServiceのテスト（変更）**
   - MatchingServiceをモック化
   - オーケストレーションのテスト

4. **Repository/Handlerのテスト（変更なし）**

---

## 変更前後の比較

### Before（変更前）

```go
// Service層に散在するドメインロジック
func (s *userService) RegisterCrush(...) {
    // 80行のロジック

    // 自己登録チェック
    if currentUser.Name == crushName && currentUser.Birthday == crushBirthday {
        return false, "", fmt.Errorf("cannot register yourself")
    }

    // 状態遷移
    currentUser.RegistrationStep = 2

    // マッチング判定（長いロジック）
    crushUser, err := s.userRepo.FindByNameAndBirthday(...)
    reverseLike, err := s.likeRepo.FindMatchingLike(...)
    s.likeRepo.UpdateMatched(...)
    s.likeRepo.UpdateMatched(...)
    // ...
}
```

**問題点:**
- Service層が肥大化（80行）
- ドメインロジックが散在
- テストしづらい

---

### After（変更後）

```go
// Model層にドメインロジック
func (u *User) IsSamePerson(name, birthday string) bool {
    return u.Name == name && u.Birthday == birthday
}

func (u *User) CompleteCrushRegistration() {
    u.RegistrationStep = 2
}

// MatchingServiceにマッチングロジック
func (m *MatchingService) CheckAndUpdateMatch(...) (bool, string, error) {
    // 30行のマッチング判定ロジック
}

// UserServiceはオーケストレーション（30行）
func (s *userService) RegisterCrush(...) {
    // バリデーション
    if currentUser.IsSamePerson(crushName, crushBirthday) {
        return error
    }

    // Like登録
    like := model.NewLike(userID, crushName, crushBirthday)
    s.likeRepo.Create(ctx, like)

    // 状態遷移
    currentUser.CompleteCrushRegistration()
    s.userRepo.Update(ctx, currentUser)

    // マッチング判定
    matched, name, err := s.matchingService.CheckAndUpdateMatch(ctx, currentUser, like)
    return matched, name, err
}
```

**改善点:**
- Service層が軽量化（30行）
- ドメインロジックがModel層に集約
- MatchingServiceが独立してテスト可能
- 意図が明確

---

## 実装の優先順位

### Phase 1: Model層のメソッド追加
1. `User.IsSamePerson()`, `CompleteCrushRegistration()` など追加
2. `Like.MarkAsMatched()`, `NewLike()` など追加
3. Model層のテスト追加

### Phase 2: MatchingServiceの導入
1. `matching_service.go` 作成
2. MatchingServiceのテスト追加
3. `NewMatchingService()` 実装

### Phase 3: UserServiceのリファクタ
1. `NewUserService()` にMatchingServiceを追加
2. `RegisterCrush()` をリファクタ（Modelメソッド使用、MatchingService委譲）
3. `RegisterFromLIFF()` をリファクタ（Modelメソッド使用）
4. UserServiceのテストを更新（MatchingServiceモック化）

### Phase 4: DIの更新
1. `main.go` でMatchingServiceを初期化
2. 既存のテストを更新

---

## 期待される効果

1. **コードの可読性向上**
   - ドメインロジックがModel層に集約
   - 意図が明確なメソッド名

2. **保守性向上**
   - 責務が明確に分離
   - 変更影響範囲が限定的

3. **テスタビリティ向上**
   - Model層が単独でテスト可能
   - MatchingServiceが単独でテスト可能
   - UserServiceのテストでMatchingServiceをモック化

4. **再利用性向上**
   - ドメインロジックが再利用可能
   - MatchingServiceが他のServiceからも利用可能

---

## まとめ

このリファクタリングでは、オーバーエンジニアリングを避けつつ、実用的な改善を実現する。

**キーポイント:**
- Value Objectは導入しない（シンプルさ重視）
- serviceディレクトリは1つ（Go的）
- Aggregate境界は現状維持（複雑化を避ける）
- テスタビリティを重視

**次のステップ:**
実装計画の作成 → git worktreeでの実装 → テスト → マージ
