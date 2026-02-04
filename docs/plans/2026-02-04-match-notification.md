# マッチング通知機能 実装計画

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** 相思相愛成立時に両ユーザーへLINE Push通知を送信する機能を実装する

**Architecture:** lineBotClientインターフェースにPushMessageメソッドを追加し、UserServiceからMatchingService経由で取得した相手ユーザー情報を使って両方に通知を送信する

**Tech Stack:** Go 1.25.5, LINE Messaging API v8, SQLite

---

## Task 1: lineBotClient インターフェース拡張

**Files:**
- Modify: `internal/linebot/client.go:6-8` (Client interface)
- Modify: `internal/linebot/client.go:19-21` (clientReal implementation)
- Modify: `e2e/integration_test.go:62-66` (mockLineBotClient)

**参考資料:**
- LINE Bot SDK Go v8: `/Users/takahashi.hikaru/line-bot-sdk-go/linebot/messaging_api/api_messaging.go`
- PushMessageRequest構造: `/Users/takahashi.hikaru/line-bot-sdk-go/linebot/messaging_api/model_push_message_request.go`

**Step 1: Client インターフェースに PushMessage メソッドを追加**

File: `internal/linebot/client.go`

```go
type Client interface {
	ReplyMessage(request *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error)
	PushMessage(request *messaging_api.PushMessageRequest) (*messaging_api.PushMessageResponse, error)
}
```

**Step 2: clientReal に PushMessage 実装を追加**

File: `internal/linebot/client.go`

```go
func (c *clientReal) PushMessage(request *messaging_api.PushMessageRequest) (*messaging_api.PushMessageResponse, error) {
	return c.messagingAPI.PushMessage(request)
}
```

**Step 3: e2e テストのモックに PushMessage を追加**

File: `e2e/integration_test.go`

```go
func (m *mockLineBotClient) PushMessage(request *messaging_api.PushMessageRequest) (*messaging_api.PushMessageResponse, error) {
	return &messaging_api.PushMessageResponse{}, nil
}
```

**Step 4: コンパイル確認**

Run: `go build ./...`
Expected: SUCCESS (no compilation errors)

**Step 5: Commit**

```bash
git add internal/linebot/client.go e2e/integration_test.go
git commit -m "feat: add PushMessage method to lineBotClient interface"
```

---

## Task 2: MatchingService 戻り値を拡張

**Files:**
- Modify: `internal/service/matching_service.go:11-16` (MatchingService interface)
- Modify: `internal/service/matching_service.go:33-93` (CheckAndUpdateMatch implementation)

**Step 1: MatchingService インターフェースの戻り値を変更**

File: `internal/service/matching_service.go`

変更前:
```go
CheckAndUpdateMatch(ctx context.Context, currentUser *model.User, currentLike *model.Like) (matched bool, matchedUserName string, err error)
```

変更後:
```go
CheckAndUpdateMatch(ctx context.Context, currentUser *model.User, currentLike *model.Like) (matched bool, matchedUser *model.User, err error)
```

**Step 2: CheckAndUpdateMatch 実装を修正（相手ユーザー取得部分）**

File: `internal/service/matching_service.go` (Line 62付近)

変更前:
```go
if mutualLike != nil {
	// マッチステータスを更新
	// ...
	return true, mutualLike.FromUserName, nil
}

return false, "", nil
```

変更後:
```go
if mutualLike != nil {
	// 相手ユーザー情報を取得
	matchedUser, err := s.userRepo.FindByLineID(ctx, currentLike.LikedUserLineID)
	if err != nil {
		return false, nil, fmt.Errorf("matched user not found: %w", err)
	}

	// マッチステータスを更新
	// ...

	return true, matchedUser, nil
}

return false, nil, nil
```

**Step 3: コンパイルエラー確認**

Run: `go build ./...`
Expected: FAIL (UserService が matchedUserName を期待しているためコンパイルエラー)

**Step 4: UserService の RegisterCrush を一時的に修正（コンパイル通すため）**

File: `internal/service/user_service.go` (Line 199付近)

変更前:
```go
matched, matchedUserName, err := s.matchingService.CheckAndUpdateMatch(ctx, currentUser, like)
```

変更後:
```go
matched, matchedUser, err := s.matchingService.CheckAndUpdateMatch(ctx, currentUser, like)
if err != nil {
	return false, "", fmt.Errorf("matching check failed: %w", err)
}

matchedUserName := ""
if matchedUser != nil {
	matchedUserName = matchedUser.Name
}
```

**Step 5: コンパイル確認**

Run: `go build ./...`
Expected: SUCCESS

**Step 6: 既存テスト実行**

Run: `go test ./e2e -v`
Expected: PASS (既存の動作を壊していない)

**Step 7: Commit**

```bash
git add internal/service/matching_service.go internal/service/user_service.go
git commit -m "refactor: change MatchingService to return User object instead of name"
```

---

## Task 3: UserService に通知ロジック追加

**Files:**
- Modify: `internal/service/user_service.go:199-206` (RegisterCrush method)
- Modify: `internal/service/user_service.go` (add sendMatchNotification helper)

**Step 1: sendMatchNotification ヘルパーメソッドを追加**

File: `internal/service/user_service.go` (ファイル末尾に追加)

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

**Step 2: RegisterCrush メソッドに通知送信処理を追加**

File: `internal/service/user_service.go` (Line 199-206)

変更前:
```go
matched, matchedUser, err := s.matchingService.CheckAndUpdateMatch(ctx, currentUser, like)
if err != nil {
	return false, "", fmt.Errorf("matching check failed: %w", err)
}

matchedUserName := ""
if matchedUser != nil {
	matchedUserName = matchedUser.Name
}
```

変更後:
```go
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

matchedUserName := ""
if matchedUser != nil {
	matchedUserName = matchedUser.Name
}
```

**Step 3: コンパイル確認**

Run: `go build ./...`
Expected: SUCCESS

**Step 4: 既存テスト実行**

Run: `go test ./e2e -v`
Expected: PASS (モックなので通知送信は実行されない)

**Step 5: Commit**

```bash
git add internal/service/user_service.go
git commit -m "feat: send LINE push notifications on match"
```

---

## Task 4: フロントエンド表示変更

**Files:**
- Modify: `liff/register-crush.html:100-104` (alert message)

**Step 1: マッチ結果の画面表示を削除**

File: `liff/register-crush.html` (Line 100-104付近)

変更前:
```javascript
if (data.matched) {
    alert(`相思相愛が成立しました！相手：${data.matched_user_name}`);
} else {
    alert('登録が完了しました');
}
```

変更後:
```javascript
alert('登録が完了しました。結果はLINEでお知らせします。');
```

**Step 2: ブラウザで動作確認**

1. `liff/register-crush.html` をブラウザで開く
2. フォーム送信
3. アラートメッセージが「登録が完了しました。結果はLINEでお知らせします。」になることを確認

**Step 3: Commit**

```bash
git add liff/register-crush.html
git commit -m "feat: update UI to indicate results via LINE notification"
```

---

## Task 5: 統合テスト実行と動作確認

**Files:**
- None (testing only)

**Step 1: 統合テストを実行（モック使用）**

Run: `SKIP_LINE_API=true go test ./e2e -v`
Expected: PASS (全テストが成功)

**Step 2: 統合テストを実行（実際のLINE API使用）**

Run: `go test ./e2e -v`
Expected: PASS (実際のLINE APIに通知が送信される)

**Step 3: ローカルサーバーで手動テスト**

```bash
# サーバー起動
go run cmd/server/main.go

# ブラウザで以下をテスト:
# 1. ユーザー登録
# 2. Crush登録（マッチなし）
# 3. 相手ユーザーがCrush登録（マッチ成立）
# 4. 両方のLINEアカウントに通知が届くことを確認
```

**Step 4: EC2にデプロイして本番環境テスト**

```bash
# デプロイ
make deploy

# サーバーログ確認
make logs

# 実際のLINEアカウントでエンドツーエンドテスト
```

**Step 5: 最終確認**

- [ ] マッチ成立時に両ユーザーにLINE通知が届く
- [ ] 通知内容が正しい（「相思相愛が成立しました！\n相手：[名前]」）
- [ ] フロントエンドで「登録が完了しました。結果はLINEでお知らせします。」と表示される
- [ ] 通知送信失敗してもユーザー登録は成功する
- [ ] エラーログが適切に記録される

---

## 実装上の注意事項

### LINE Bot SDK の使用方法

**PushMessageRequest の構造:**

```go
&messaging_api.PushMessageRequest{
    To: userLineID,  // string: 送信先のLINE User ID
    Messages: []messaging_api.MessageInterface{
        messaging_api.TextMessage{
            Type: "text",
            Text: "メッセージ本文",
        },
    },
    NotificationDisabled: false,  // bool: falseでプッシュ通知有効
}
```

**重要:**
- `To` は必ず `user.LineID` を使用（`liked_user_line_id` ではない）
- `Messages` は配列なので複数メッセージ送信可能（今回は1つ）
- `NotificationDisabled: false` でプッシュ通知を送る

### エラーハンドリング

**通知送信失敗時:**
- `log.Printf()` でエラーログ記録
- ユーザー登録処理は継続（通知失敗は致命的エラーではない）
- HTTPレスポンスは成功を返す

**MatchingService失敗時:**
- エラーを返却してユーザー登録を中断
- HTTPレスポンスはエラーを返す

### テスト戦略

**モック使用（CI/CD）:**
- `SKIP_LINE_API=true` で実行
- `mockLineBotClient.PushMessage()` が呼ばれる
- 実際のLINE APIは呼ばない

**実際のAPI使用（ローカル）:**
- `.env` に認証情報設定
- 実際のLINE APIに通知送信
- エンドツーエンドの動作確認

### 参考資料

実装時に以下を十分に調査すること:

- LINE Bot SDK Go v8: `/Users/takahashi.hikaru/line-bot-sdk-go/`
  - `linebot/messaging_api/model_push_message_request.go`
  - `linebot/messaging_api/api_messaging.go`
- LINE Messaging API Reference: https://developers.line.biz/en/docs/messaging-api/
- Push Message API: https://developers.line.biz/en/reference/messaging-api/#send-push-message
