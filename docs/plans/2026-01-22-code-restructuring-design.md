# コード構造リファクタリング設計

**日付**: 2026-01-22
**目的**: テスタビリティ向上と綺麗な構造への移行

## 概要

現在の平坦なファイル構造から、レイヤー分離された構造に移行する。DI（依存性注入）を導入してテストしやすくする。

## 現状の問題点

- DI がない → テストが難しい
- handler が bot クライアントに直接依存 → モックできない
- テストが db_test.go だけ
- 全てが main パッケージに存在

## 新しいディレクトリ構造

```
cupid/
├── cmd/
│   └── server/
│       └── main.go          # エントリーポイント
├── internal/
│   ├── handler/             # HTTPハンドラー層
│   │   ├── webhook.go
│   │   └── webhook_test.go
│   ├── service/             # ビジネスロジック層
│   │   ├── user_service.go
│   │   ├── user_service_test.go
│   │   ├── match_service.go
│   │   └── match_service_test.go
│   ├── repository/          # データアクセス層
│   │   ├── user_repo.go
│   │   ├── user_repo_test.go
│   │   ├── like_repo.go
│   │   └── like_repo_test.go
│   └── model/               # ドメインモデル
│       ├── user.go
│       └── like.go
├── pkg/
│   └── database/            # DB初期化
│       ├── database.go
│       └── database_test.go
└── entities/                # SQLBoiler生成コード（変更なし）
```

## 各層の責務

### handler 層
- HTTP/LINE リクエストを受け取る
- service を呼び出す
- レスポンスを返す
- エラーを HTTP ステータスコードに変換

### service 層
- ビジネスロジック
- 「ユーザー登録」「マッチング判定」など
- repository を使ってデータを操作
- エラーをラップして context を追加

### repository 層
- DB 操作（CRUD）
- SQLBoiler の entities を使用
- DB エラーをそのまま返す

### model 層
- アプリケーション内で使うドメインモデル
- entities とは別に定義
- ビジネスロジックで扱いやすい形

### pkg/database
- DB 接続の初期化
- PRAGMA 設定など

## 依存性注入（DI）設計

各層を interface で抽象化し、テスト時に mock を注入できるようにする。

### repository interface 例

```go
type UserRepository interface {
    FindByLineID(ctx context.Context, lineID string) (*model.User, error)
    Create(ctx context.Context, user *model.User) error
    Update(ctx context.Context, user *model.User) error
}

type userRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
    return &userRepository{db: db}
}
```

### service interface 例

```go
type UserService interface {
    RegisterUser(ctx context.Context, lineID, displayName string) error
    GetOrCreateUser(ctx context.Context, lineID, displayName string) (*model.User, error)
}

type userService struct {
    userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
    return &userService{userRepo: userRepo}
}
```

### handler での DI

```go
type WebhookHandler struct {
    channelSecret string
    bot          *messaging_api.MessagingApiAPI
    userService  service.UserService
}

func NewWebhookHandler(secret string, bot *messaging_api.MessagingApiAPI, userSvc service.UserService) *WebhookHandler {
    return &WebhookHandler{
        channelSecret: secret,
        bot:          bot,
        userService:  userSvc,
    }
}
```

### main.go での組み立て

```go
// DB初期化
db := database.New("cupid.db")
defer db.Close()

// 依存関係を下から上に構築
userRepo := repository.NewUserRepository(db)
userService := service.NewUserService(userRepo)
handler := handler.NewWebhookHandler(channelSecret, bot, userService)

// HTTPハンドラー登録
http.HandleFunc("/webhook", handler.ServeHTTP)
```

## テスト戦略

### repository のテスト
- **実際の SQLite DB を使用**（インメモリ）
- migration を実行してスキーマを準備
- CRUD 操作を検証
- トランザクションのロールバックでクリーンアップ

```go
func TestUserRepository_Create(t *testing.T) {
    db := setupTestDB(t)  // migration実行済み
    defer db.Close()

    repo := NewUserRepository(db)
    user := &model.User{LineID: "U123", DisplayName: "Test"}

    err := repo.Create(context.Background(), user)
    require.NoError(t, err)

    found, err := repo.FindByLineID(context.Background(), "U123")
    require.NoError(t, err)
    assert.Equal(t, "Test", found.DisplayName)
}
```

### service のテスト
- **repository を mock**
- ビジネスロジックだけを検証
- エラーハンドリングのテスト
- `github.com/stretchr/testify/mock` 使用

```go
func TestUserService_RegisterUser(t *testing.T) {
    mockRepo := new(MockUserRepository)
    service := NewUserService(mockRepo)

    mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
        return u.LineID == "U123"
    })).Return(nil)

    err := service.RegisterUser(context.Background(), "U123", "Test")
    require.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

### handler のテスト
- **service を mock**
- HTTP リクエスト/レスポンスを検証
- `httptest.NewRecorder()` 使用
- LINE の署名検証は後回し（最初は基本的なテストのみ）

## エラーハンドリング

各層で適切にエラーを処理する。

### repository 層
- DB エラーをそのまま返す
- エラーメッセージは最小限

### service 層
- ビジネスロジックのエラーをラップ
- `fmt.Errorf("failed to create user: %w", err)` で context 追加
- カスタムエラー型は必要に応じて追加

### handler 層
- エラーログを出力
- HTTP ステータスコードに変換
- クライアントには詳細なエラーメッセージを返さない

```go
if err := h.userService.RegisterUser(ctx, lineID, displayName); err != nil {
    log.Printf("Failed to register user: %v", err)
    w.WriteHeader(http.StatusInternalServerError)
    return
}
```

## 移行ステップ（4ステップ）

### ステップ1: ディレクトリ構造作成 + database 移動

**作業内容:**
- `cmd/server/` ディレクトリ作成
- `main.go` → `cmd/server/main.go` 移動
- `internal/` ディレクトリ作成
- `pkg/database/` ディレクトリ作成
- `db.go` → `pkg/database/database.go` 移動
- `db_test.go` → `pkg/database/database_test.go` 移動
- import パス修正

**確認:**
- `go build -o cupid ./cmd/server` でビルド成功
- `go test ./...` でテスト成功
- デプロイして動作確認

**コミットメッセージ:** `refactor: ディレクトリ構造を整理`

### ステップ2: repository 層作成 + テスト

**作業内容:**
- `internal/model/user.go` 作成（entities とは別のドメインモデル）
- `internal/repository/user_repo.go` 作成（interface + 実装）
- `internal/repository/user_repo_test.go` 作成
- テストヘルパー（setupTestDB）作成

**確認:**
- `go test ./internal/repository` でテスト成功
- デプロイ不要（まだ使われていない）

**コミットメッセージ:** `feat: repository層とドメインモデルを追加`

### ステップ3: service 層作成 + テスト

**作業内容:**
- `internal/service/user_service.go` 作成（interface + 実装）
- `internal/service/user_service_test.go` 作成
- mock repository 作成
- `go.mod` に testify 追加

**確認:**
- `go test ./internal/service` でテスト成功
- デプロイ不要（まだ使われていない）

**コミットメッセージ:** `feat: service層を追加（テスト含む）`

### ステップ4: handler リファクタリング + DI

**作業内容:**
- `handler.go` → `internal/handler/webhook.go` 移動
- 関数ベースから struct ベースに変更
- DI 導入（service を受け取る）
- `cmd/server/main.go` で依存関係を組み立て
- 既存のオウム返し機能を新しい構造に移行

**確認:**
- `go build -o cupid ./cmd/server` でビルド成功
- `go test ./...` で全テスト成功
- デプロイして動作確認（オウム返しが動くこと）

**コミットメッセージ:** `refactor: handler層をDI対応にリファクタリング`

## 今後の拡張

この構造により、以下が容易になる：

- 新しい機能追加時：service と repository を追加するだけ
- テスト追加：各層を独立してテストできる
- mock の利用：interface があるため容易
- エラーハンドリング：各層で適切に処理できる

## 参考資料

- Go の標準的なプロジェクト構造: https://github.com/golang-standards/project-layout
- testify/mock: https://github.com/stretchr/testify
