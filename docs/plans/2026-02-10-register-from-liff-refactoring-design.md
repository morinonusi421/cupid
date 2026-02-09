# RegisterFromLIFF リファクタリング設計

## 概要

RegisterFromLIFFメソッドを、初回登録と再登録（情報更新）を明確に分離する設計にリファクタリングする。

## 現状の問題点

### 1. 初回登録と再登録が混在
- `GetOrCreateUser` + `UpdateUser` の流れで処理
- 初回と再登録の違いが不明確

### 2. 不適切なメッセージ送信
- 再登録時にも「好きな人を登録してね💘」が送られる
- ユーザーは既に登録済みなので不適切

### 3. 不要な処理の実行
- `CompleteUserRegistration()` が再登録時にも呼ばれる
- 既に `registration_step = 1` なので不要

### 4. 複雑な依存関係
- `RegisterUser` → `GetOrCreateUser` → `RegisterFromLIFF`
- `RegisterUser` は「不完全なユーザー」を作成（Birthday空、step=0）
- LIFFからの登録には不適切

## 要件

### 機能要件

1. **初回登録**
   - Name, Birthday, RegistrationStep=1 の完全なユーザーを作成
   - 「登録完了！次に、好きな人を登録してね💘」メッセージ送信（QuickReply付き）

2. **再登録（情報更新）**
   - Name, Birthday のみ更新
   - RegisteredAt は保持
   - 「情報を更新しました✨」メッセージ送信（シンプルな確認）

3. **registration_step の扱い**
   - 初回登録：最初から 1 に設定
   - 再登録：通常は既に 1（念のため 0 なら 1 に更新）

### 非機能要件

- コードの意図を明確にする
- テストしやすい設計
- YAGNI原則に従う（使われないメソッドは削除）

## ユースケース

### 初回登録
- ユーザーがLIFFフォームを初めて送信
- 頻度：各ユーザー1回のみ

### 再登録（情報更新）
- 名前や誕生日を間違えたので訂正したい
- 頻度：レアケース
- 注：カタカナフルネーム強制なので、表記ゆれによる変更はない

## 設計

### 全体構造

```go
func (s *userService) RegisterFromLIFF(ctx context.Context, userID, name, birthday string) error {
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
        return s.updateUserInfo(ctx, user, name, birthday)
    }
}
```

**変更点：**
- `GetOrCreateUser` を使わず、`FindByLineID` で明示的に検索
- `user == nil` で初回か再登録かを判断
- 2つのプライベートメソッドに委譲

### 初回登録（registerNewUser）

```go
// registerNewUser は初回登録時に新規ユーザーを作成する
func (s *userService) registerNewUser(ctx context.Context, userID, name, birthday string) error {
    // 1. 完全なユーザーオブジェクトを作成
    user := &model.User{
        LineID:           userID,
        Name:             name,
        Birthday:         birthday,
        RegistrationStep: 1,  // 最初から登録完了状態
        RegisteredAt:     "", // DBのDEFAULT（現在時刻）を使用
        UpdatedAt:        "", // DBのDEFAULT（現在時刻）を使用
    }

    // 2. DBに保存
    if err := s.userRepo.Create(ctx, user); err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }

    // 3. 好きな人登録を促すメッセージを送信
    if err := s.sendCrushRegistrationPrompt(ctx, user); err != nil {
        log.Printf("Failed to send crush registration prompt to %s: %v", user.LineID, err)
        // エラーをログに記録するが、登録処理は成功として扱う
    }

    return nil
}
```

**ポイント：**
- `RegistrationStep` を最初から 1 に設定（`CompleteUserRegistration` 不要）
- `RegisterUser` や `GetOrCreateUser` を経由しない
- 初回専用のメッセージを送信

### 再登録（updateUserInfo）

```go
// updateUserInfo は再登録時に既存ユーザーの情報を更新する
func (s *userService) updateUserInfo(ctx context.Context, user *model.User, name, birthday string) error {
    // 1. ユーザー情報を更新
    user.Name = name
    user.Birthday = birthday

    // 2. registration_step が 0 の場合のみ 1 に更新（通常はありえないが念のため）
    if user.RegistrationStep == 0 {
        user.CompleteUserRegistration()
    }

    // 3. DBに保存
    if err := s.userRepo.Update(ctx, user); err != nil {
        return fmt.Errorf("failed to update user: %w", err)
    }

    // 4. 更新完了メッセージを送信
    if err := s.sendUserInfoUpdateConfirmation(ctx, user); err != nil {
        log.Printf("Failed to send update confirmation to %s: %v", user.LineID, err)
        // エラーをログに記録するが、更新処理は成功として扱う
    }

    return nil
}
```

**ポイント：**
- Name, Birthday のみ更新（RegisteredAt は保持）
- `registration_step` は条件付きで更新（通常は既に 1）
- 再登録専用のメッセージを送信

### メッセージ送信（sendUserInfoUpdateConfirmation）

```go
// sendUserInfoUpdateConfirmation は情報更新完了のメッセージを送信する
func (s *userService) sendUserInfoUpdateConfirmation(ctx context.Context, user *model.User) error {
    message := "情報を更新しました✨"

    request := &messaging_api.PushMessageRequest{
        To: user.LineID,
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

**既存のメッセージとの使い分け：**
- `sendCrushRegistrationPrompt`: 初回登録時（QuickReply付き、好きな人登録を促す）
- `sendUserInfoUpdateConfirmation`: 再登録時（シンプルな確認メッセージ）

## 削除するメソッド

### 1. RegisterUser
- GetOrCreateUserからのみ呼ばれている
- 完全に不要になる
- インターフェース定義も削除
- テストも削除

### 2. GetOrCreateUser
- RegisterFromLIFFでしか使われていない
- RegisterFromLIFFで使わなくなる
- 完全に不要になる
- インターフェース定義も削除
- テスト4つも削除（TestUserService_GetOrCreateUser_*）

**理由：YAGNI原則**
- 使われていないメソッドは削除する
- 将来必要になったら、その時に追加すればいい

## 影響を受けるファイル

### 実装
1. `internal/service/user_service.go`
   - RegisterFromLIFF を修正
   - registerNewUser を追加
   - updateUserInfo を追加
   - sendUserInfoUpdateConfirmation を追加
   - RegisterUser を削除
   - GetOrCreateUser を削除

### テスト
2. `internal/service/user_service_test.go`
   - TestUserService_RegisterFromLIFF を修正
   - TestUserService_RegisterFromLIFF_NewUser を追加（初回登録）
   - TestUserService_RegisterFromLIFF_UpdateExisting を追加（再登録）
   - TestUserService_GetOrCreateUser_* を削除（4つ）

### モック
3. `internal/handler/webhook_test.go`
   - MockUserService から RegisterUser, GetOrCreateUser を削除

4. `internal/handler/registration_api_test.go`
   - MockUserServiceForAPI から RegisterUser, GetOrCreateUser を削除

## テストケース

### 新規追加

1. **TestUserService_RegisterFromLIFF_NewUser**
   - 初回登録のテスト
   - user == nil の場合
   - Create が呼ばれることを確認
   - sendCrushRegistrationPrompt が呼ばれることを確認

2. **TestUserService_RegisterFromLIFF_UpdateExisting**
   - 再登録のテスト
   - user != nil の場合
   - Update が呼ばれることを確認
   - sendUserInfoUpdateConfirmation が呼ばれることを確認

### 既存（そのまま）

3. **TestUserService_RegisterFromLIFF_InvalidName**
   - バリデーションエラーのテスト

### 削除

- TestUserService_GetOrCreateUser_ExistingUser
- TestUserService_GetOrCreateUser_NewUser
- TestUserService_GetOrCreateUser_FindError
- TestUserService_GetOrCreateUser_CreateError

## エラーハンドリング

### FindByLineID のエラー
```go
if err != nil {
    return fmt.Errorf("failed to find user: %w", err)
}
```

### Create のエラー
```go
if err := s.userRepo.Create(ctx, user); err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}
```

### Update のエラー
```go
if err := s.userRepo.Update(ctx, user); err != nil {
    return fmt.Errorf("failed to update user: %w", err)
}
```

### メッセージ送信のエラー
- ログに記録するが、処理は成功として扱う
- ユーザー登録/更新は完了しているため

## 実装順序

1. 新しいメソッド追加（registerNewUser, updateUserInfo, sendUserInfoUpdateConfirmation）
2. RegisterFromLIFF を修正（新しいメソッドを使用）
3. テスト追加（初回登録、再登録）
4. テスト実行・確認
5. 不要なメソッド削除（RegisterUser, GetOrCreateUser）
6. 不要なテスト削除
7. モック修正
8. 全テスト実行・確認
9. ビルド確認
10. デプロイ

## 期待される効果

### コードの明確化
- 初回登録と再登録の違いが明確
- 意図が分かりやすい

### バグの修正
- 再登録時に不適切なメッセージが送られる問題を解決

### テスタビリティの向上
- 初回と再登録を別々にテストできる

### コードの簡潔化
- 不要なメソッド（RegisterUser, GetOrCreateUser）を削除
- 依存関係がシンプルになる
