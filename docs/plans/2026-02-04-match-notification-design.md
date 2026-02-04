# マッチング通知機能 設計書

## 概要

相思相愛が成立した際、両方のユーザーにLINE Push通知を送信する機能を追加する。

## 背景

現在の実装では、マッチング成立時にHTTPレスポンスで結果を返すのみで、LINE通知は送信されていない。ユーザーがブラウザを閉じた後でも通知を受け取れるよう、LINE Messaging APIのPush Message機能を使用する。

## 要件

### 機能要件

1. **マッチング成立時の通知**
   - 相思相愛が成立したら、両方のユーザーにLINE Push通知を送信
   - 通知内容：「相思相愛が成立しました！\n相手：[相手の名前]」

2. **フロントエンド表示**
   - Crush登録完了時、画面には「登録が完了しました。結果はLINEでお知らせします。」と表示
   - マッチ結果の画面表示は削除（LINE通知に一本化）

3. **エラーハンドリング**
   - Push通知送信失敗時もユーザー登録は成功扱い
   - エラーはログに記録して処理継続

### 非機能要件

- 既存のAPI仕様（HTTPレスポンス形式）は維持
- テストでは実際のLINE APIを呼ばずモックで検証

## アーキテクチャ

### コンポーネント変更

```
┌─────────────────┐
│  UserService    │
│  ┌───────────┐  │
│  │RegisterCrush│ │
│  └─────┬───────┘ │
│        │         │
│        ├─────────┼─→ MatchingService.CheckAndUpdateMatch()
│        │         │     戻り値: (matched, matchedUser, err)
│        │         │
│        └─────────┼─→ lineBotClient.PushMessage() ×2
│                  │     （現在のユーザー + 相手ユーザー）
└─────────────────┘
```

### 通知フロー

```
1. User A が User B を Crush 登録
   ↓
2. MatchingService がマッチング判定
   ↓
3. マッチ成立（User B も User A を登録済み）
   ↓
4. UserService が両方に通知送信
   ├─→ User A に Push通知
   └─→ User B に Push通知
   ↓
5. HTTPレスポンス返却（matched: true）
```

## 実装詳細

### 1. lineBotClient インターフェース拡張

**ファイル:** `internal/linebot/client.go`

```go
type Client interface {
    ReplyMessage(request *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error)
    PushMessage(request *messaging_api.PushMessageRequest) (*messaging_api.PushMessageResponse, error)  // 追加
}

// 実装
func (c *clientReal) PushMessage(request *messaging_api.PushMessageRequest) (*messaging_api.PushMessageResponse, error) {
    return c.messagingAPI.PushMessage(request)
}

// モック
func (m *mockLineBotClient) PushMessage(request *messaging_api.PushMessageRequest) (*messaging_api.PushMessageResponse, error) {
    return &messaging_api.PushMessageResponse{}, nil
}
```

**PushMessageRequest構造:**

```go
&messaging_api.PushMessageRequest{
    To: userLineID,  // 送信先のLINE User ID
    Messages: []messaging_api.MessageInterface{
        messaging_api.TextMessage{
            Type: "text",
            Text: "相思相愛が成立しました！\n相手：[名前]",
        },
    },
    NotificationDisabled: false,  // プッシュ通知を有効化
}
```

### 2. MatchingService 戻り値拡張

**ファイル:** `internal/service/matching_service.go`

**変更前:**

```go
func (s *matchingService) CheckAndUpdateMatch(
    ctx context.Context,
    currentUser *model.User,
    currentLike *model.Like,
) (matched bool, matchedUserName string, err error)
```

**変更後:**

```go
func (s *matchingService) CheckAndUpdateMatch(
    ctx context.Context,
    currentUser *model.User,
    currentLike *model.Like,
) (matched bool, matchedUser *model.User, err error)
```

**実装:**

```go
if mutualLike != nil {
    // 相手ユーザー情報を取得
    matchedUser, err := s.userRepo.FindByLineID(ctx, currentLike.LikedUserLineID)
    if err != nil {
        return false, nil, fmt.Errorf("matched user not found: %w", err)
    }

    // マッチステータス更新
    // ...

    return true, matchedUser, nil
}

return false, nil, nil
```

### 3. UserService 通知ロジック

**ファイル:** `internal/service/user_service.go`

**RegisterCrush メソッド内:**

```go
// マッチング判定（MatchingService に委譲）
matched, matchedUser, err := s.matchingService.CheckAndUpdateMatch(ctx, currentUser, like)
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
}

return matched, matchedUser.Name, nil
```

**sendMatchNotification ヘルパー:**

```go
func (s *userService) sendMatchNotification(ctx context.Context, toUser *model.User, matchedWithUser *model.User) error {
    message := fmt.Sprintf("相思相愛が成立しました！\n相手：%s", matchedWithUser.Name)

    request := &messaging_api.PushMessageRequest{
        To: toUser.LineID,
        Messages: []messaging_api.MessageInterface{
            messaging_api.TextMessage{
                Type: "text",
                Text: message,
            },
        },
        NotificationDisabled: false,
    }

    _, err := s.lineBotClient.PushMessage(request)
    return err
}
```

### 4. フロントエンド変更

**ファイル:** `liff/register-crush.html`

**変更前:**

```javascript
if (data.matched) {
    alert(`相思相愛が成立しました！相手：${data.matched_user_name}`);
} else {
    alert('登録が完了しました');
}
```

**変更後:**

```javascript
alert('登録が完了しました。結果はLINEでお知らせします。');
```

### 5. バックエンドAPIレスポンス

**ファイル:** `internal/handler/crush_registration_api_handler.go`

**変更なし（互換性維持）:**

```go
response := map[string]interface{}{
    "matched":           matched,
    "matched_user_name": matchedUserName,
}
```

## エラーハンドリング

| エラー箇所 | 処理 | 理由 |
|-----------|------|------|
| PushMessage送信失敗 | ログ記録、処理継続 | 通知失敗は致命的エラーではない |
| MatchingService失敗 | エラー返却、登録中断 | マッチング判定は必須処理 |
| DB操作失敗 | エラー返却、ロールバック | データ整合性を保つ |

## テスト戦略

### 既存テスト（e2e/integration_test.go）

- `TestIntegration_CrushRegistrationMatch`: マッチ時にPushMessageが呼ばれることを確認
- モックを使用して実際のLINE APIは呼ばない
- インターフェースメソッドの呼び出しを検証

## 参考資料

実装時に以下を十分に調査すること：

- LINE Bot SDK Go v8: `/Users/takahashi.hikaru/line-bot-sdk-go/`
  - `linebot/messaging_api/model_push_message_request.go`
  - `linebot/messaging_api/api_messaging.go`
- LINE Messaging API Reference: https://developers.line.biz/en/docs/messaging-api/
- Push Message API: https://developers.line.biz/en/reference/messaging-api/#send-push-message

## 実装順序

1. lineBotClient インターフェースとモック追加
2. MatchingService 戻り値変更
3. UserService 通知ロジック実装
4. フロントエンド表示変更
5. テスト追加・修正
6. 動作確認（ローカル → EC2）
