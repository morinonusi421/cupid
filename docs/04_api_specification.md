# API仕様

## 使用するLINE Messaging API機能

このプロジェクトで使用するLINE Messaging APIの機能は以下の3つに限定される、よ。

### 1. Follow Event（友だち追加イベント）
**用途**: ユーザーがBotを友だち追加したときの初期登録
- 新規ユーザーをDBに登録
- 初回メッセージ「はじめまして。あなたの名前を教えて、ね（ひらがなで）」を送信

### 2. Message Event（テキストメッセージ受信）
**用途**: ユーザーからのテキスト入力を受け取る
- 名前、生年月日、好きな人の名前、好きな人の生年月日を受信
- `registration_step`に応じて状態遷移

### 3. Reply Message API
**用途**: ユーザーのメッセージに即座に返信
- Follow Event、Message Eventに対する返信
- `replyToken`を使用（1回のみ有効）

### 4. Push Message API
**用途**: マッチング成立時の通知送信
- 相手ユーザーに「相思相愛、みたい。おめでとう。」を送信
- `replyToken`を使わず、User IDで直接送信

**使用しない機能**: 画像、動画、音声、位置情報、スタンプなどのメッセージタイプは使用しない。

---

## エンドポイント

### POST /webhook
LINE Platformからのメッセージを受け取るエンドポイント

**URL**: `https://cupid.click/webhook`
**プロトコル**: HTTPS（必須）
**メソッド**: POST

---

## リクエスト仕様

### Headers
```
Content-Type: application/json
X-Line-Signature: {HMAC-SHA256 signature}
```

#### X-Line-Signature
```
LINEが送信するリクエストの署名
Channel Secretを使ったHMAC-SHA256ハッシュ
必ず検証すること（セキュリティ）
```

### Body (LINE Webhook Event)

LINE Platformから送られてくるWebhookイベントは、主に以下の2種類：
1. **Follow Event**: ユーザーがBotを友だち追加したとき
2. **Message Event**: ユーザーがメッセージを送信したとき

#### Follow Event（友だち追加時）
```json
{
  "destination": "Uxxxx...",
  "events": [
    {
      "type": "follow",
      "timestamp": 1609459200000,
      "source": {
        "type": "user",
        "userId": "U1234567890abcdef"
      },
      "replyToken": "xxxxxxxxxxxxxxxxxxx",
      "mode": "active"
    }
  ]
}
```

**用途**: 新規ユーザーの初期登録と初回メッセージ送信

#### Message Event（テキストメッセージ受信時）
```json
{
  "destination": "Uxxxx...",
  "events": [
    {
      "type": "message",
      "message": {
        "type": "text",
        "id": "123456789",
        "text": "しのざわひろ"
      },
      "timestamp": 1609459200000,
      "source": {
        "type": "user",
        "userId": "U1234567890abcdef"
      },
      "replyToken": "xxxxxxxxxxxxxxxxxxx",
      "mode": "active"
    }
  ]
}
```

**用途**: ユーザーの入力（名前、生年月日など）を受け取る

#### 主要フィールド
| フィールド | 型 | 説明 |
|-----------|-----|------|
| events[].type | String | イベントタイプ（`follow`, `message`など） |
| events[].message.type | String | メッセージタイプ（`text`のみ使用） |
| events[].message.text | String | ユーザーが送信したテキスト |
| events[].source.userId | String | LINE User ID（Uで始まる一意な文字列） |
| events[].replyToken | String | 返信用トークン（1回のみ有効、リクエスト受信後すぐ使用） |
| events[].timestamp | Number | イベント発生時刻（Unixタイムミリ秒） |

---

## レスポンス仕様

### 成功時
```
HTTP/1.1 200 OK
Content-Type: text/plain

OK
```

**重要**: LINEは200応答を受け取ると、正常処理と判断します。エラーが起きてもまず200を返し、非同期でエラー処理することを推奨。

### エラー時
```
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "Invalid signature"
}
```

---

## 内部処理フロー

### 1. Webhook署名検証（必須）

#### 目的
```
リクエストが本当にLINE Platformから送られたものか確認
第三者による不正リクエストを防ぐ
```

#### 検証方法
```go
import (
    "github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// リクエストボディを取得
body, _ := io.ReadAll(r.Body)

// 署名取得
signature := r.Header.Get("X-Line-Signature")

// 検証
channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
if !webhook.ValidateSignature(channelSecret, signature, body) {
    // 署名が無効
    http.Error(w, "Invalid signature", http.StatusBadRequest)
    return
}
```

#### 注意点
```
- 署名検証失敗時は必ず400を返す
- ボディは一度しか読めないので保存しておく
- Channel Secretは環境変数で管理
```

---

### 2. イベントタイプのチェック

```go
var request webhook.CallbackRequest
json.Unmarshal(body, &request)

for _, event := range request.Events {
    switch event.GetType() {
    case webhook.EventTypeFollow:
        // 友だち追加イベント
        followEvent, ok := event.(*webhook.FollowEvent)
        if !ok {
            continue
        }
        handleFollowEvent(followEvent)

    case webhook.EventTypeMessage:
        // メッセージイベント
        messageEvent, ok := event.(*webhook.MessageEvent)
        if !ok {
            continue
        }

        // テキストメッセージのみ処理
        if messageEvent.Message.GetType() != webhook.MessageTypeText {
            continue
        }

        textMessage := messageEvent.Message.(*webhook.TextMessageContent)
        handleMessage(messageEvent.Source.UserId, textMessage.Text, messageEvent.ReplyToken)

    default:
        // その他のイベントは無視
        continue
    }
}
```

---

### 3. Follow Event処理（友だち追加時）

ユーザーがBotを友だち追加したときに呼ばれる処理。

```go
func handleFollowEvent(event *webhook.FollowEvent) {
    userID := event.Source.UserId
    replyToken := event.ReplyToken

    // 既に登録済みかチェック
    user, err := db.GetUser(userID)
    if err != nil {
        log.Printf("DB error: %v", err)
        replyMessage(replyToken, "エラーが発生しました")
        return
    }

    if user != nil {
        // 既に登録済み（ブロック解除など）
        replyMessage(replyToken, "おかえり、ね。")
        return
    }

    // 新規ユーザー登録
    err = db.Exec(
        "INSERT INTO users (line_user_id, name, birthday, registration_step) VALUES (?, '', '', 'awaiting_name')",
        userID,
    )
    if err != nil {
        log.Printf("Failed to create user: %v", err)
        replyMessage(replyToken, "登録エラーが発生しました")
        return
    }

    // 初回メッセージ送信
    replyMessage(replyToken, "はじめまして。あなたの名前を教えて、ね（ひらがなで）")
}
```

**重要な点**:
- Follow Eventは友だち追加時に1回だけ発生する
- ブロック解除時にも発生する可能性があるため、既存ユーザーのチェックが必要
- `registration_step`を`awaiting_name`にして、名前入力待ち状態にする

---

### 4. ユーザー状態の確認（メッセージ受信時）

```go
func handleMessage(userID, text, replyToken string) {
    // SQLiteからユーザー取得
    user, err := db.GetUser(userID)

    if err != nil {
        log.Printf("DB error: %v", err)
        replyMessage(replyToken, "エラーが発生しました")
        return
    }

    if user == nil {
        // 通常はFollow Eventで登録されるため、ここには来ない
        // 念のため新規登録処理
        createNewUser(userID, replyToken)
        return
    }

    // 既存ユーザーの状態に応じた処理
    handleUserState(user, text, replyToken)
}
```

---

### 5. 状態に応じた処理

```go
func handleUserState(user *User, text, replyToken string) {
    switch user.RegistrationStep {
    case "awaiting_name":
        // 名前として保存（ひらがなで）
        db.Exec(
            "UPDATE users SET name = ?, registration_step = ? WHERE line_user_id = ?",
            text, "awaiting_birthday", user.LineUserID,
        )
        replyMessage(replyToken, fmt.Sprintf("%s、ね。生年月日は？（例：2009-12-21）", text))

    case "awaiting_birthday":
        // 生年月日として保存
        db.Exec(
            "UPDATE users SET birthday = ?, registration_step = ? WHERE line_user_id = ?",
            text, "completed", user.LineUserID,
        )
        replyMessage(replyToken, "登録できた、よ。好きな人の名前は？（ひらがなで）")

    case "completed":
        // 好きな人の名前をtemp_crush_nameに一時保存
        db.Exec(
            "UPDATE users SET temp_crush_name = ?, registration_step = ? WHERE line_user_id = ?",
            text, "awaiting_crush_birthday", user.LineUserID,
        )
        replyMessage(replyToken, "その人の生年月日は？")

    case "awaiting_crush_birthday":
        // temp_crush_nameから好きな人の名前を取得
        var crushName string
        db.QueryRow(
            "SELECT temp_crush_name FROM users WHERE line_user_id = ?",
            user.LineUserID,
        ).Scan(&crushName)

        // マッチング処理
        handleCrushRegistration(user, crushName, text, replyToken)

        // temp_crush_nameをクリア
        db.Exec(
            "UPDATE users SET temp_crush_name = NULL, registration_step = ? WHERE line_user_id = ?",
            "completed", user.LineUserID,
        )
    }
}
```

**状態遷移の説明**:
- `awaiting_name` → 名前入力 → `awaiting_birthday`
- `awaiting_birthday` → 生年月日入力 → `completed`
- `completed` → 好きな人の名前入力（`temp_crush_name`に保存） → `awaiting_crush_birthday`
- `awaiting_crush_birthday` → 好きな人の生年月日入力 → マッチング処理 → `completed`に戻る

**temp_crush_nameフィールド**:
好きな人の名前を一時保存するフィールド。生年月日入力待ちの間だけ使用し、マッチング処理後にクリアされる。

---

### 6. マッチング処理

```go
func handleCrushRegistration(user *User, crushName, crushBirthday, replyToken string) {
    tx, _ := db.Begin()
    defer tx.Rollback()

    // likes登録
    _, err := tx.Exec(
        "INSERT INTO likes (from_user_id, to_name, to_birthday) VALUES (?, ?, ?)",
        user.LineUserID, crushName, crushBirthday,
    )
    if err != nil {
        replyMessage(replyToken, "登録エラーが発生しました")
        return
    }

    // マッチング検索
    var matchedUserID string
    err = tx.QueryRow(`
        SELECT l.from_user_id
        FROM likes l
        WHERE l.to_name = ? AND l.to_birthday = ? AND l.matched = 0
    `, user.Name, user.Birthday).Scan(&matchedUserID)

    if err == sql.ErrNoRows {
        // マッチングなし
        tx.Commit()
        replyMessage(replyToken, "登録した、よ。相思相愛なら通知する、ね。")
        return
    }

    if err != nil {
        replyMessage(replyToken, "エラーが発生しました")
        return
    }

    // マッチング成立
    tx.Exec("UPDATE likes SET matched = 1 WHERE from_user_id = ?", user.LineUserID)
    tx.Exec("UPDATE likes SET matched = 1 WHERE from_user_id = ?", matchedUserID)
    tx.Commit()

    // 両方に通知
    replyMessage(replyToken, "相思相愛、みたい。おめでとう。")
    pushMessage(matchedUserID, "相思相愛、みたい。おめでとう。")
}
```

---

## LINE Messaging API

### Reply Message API
ユーザーからのメッセージに対する即座の返信

#### エンドポイント
```
POST https://api.line.me/v2/bot/message/reply
```

#### Headers
```
Authorization: Bearer {Channel Access Token}
Content-Type: application/json
```

#### Body
```json
{
  "replyToken": "xxxxxxxxxxxxxxxxxxx",
  "messages": [
    {
      "type": "text",
      "text": "しのざわひろ、ね。生年月日は？（例：2009-12-21）"
    }
  ]
}
```

#### 制約
- `replyToken`は1回のみ使用可能
- Webhook受信後すぐに使用すること（タイムアウトあり）
- 最大5メッセージまで同時送信可能

#### line-bot-sdk-goでの実装
```go
import (
    "github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func replyMessage(replyToken, text string) error {
    bot, _ := messaging_api.NewMessagingApiAPI(os.Getenv("LINE_CHANNEL_TOKEN"))

    _, err := bot.ReplyMessage(
        &messaging_api.ReplyMessageRequest{
            ReplyToken: replyToken,
            Messages: []messaging_api.MessageInterface{
                &messaging_api.TextMessage{
                    Text: text,
                },
            },
        },
    )
    return err
}
```

---

### Push Message API
任意のタイミングでユーザーにメッセージを送信（マッチング通知用）

#### エンドポイント
```
POST https://api.line.me/v2/bot/message/push
```

#### Headers
```
Authorization: Bearer {Channel Access Token}
Content-Type: application/json
```

#### Body
```json
{
  "to": "U1234567890abcdef",
  "messages": [
    {
      "type": "text",
      "text": "相思相愛、みたい。おめでとう。"
    }
  ]
}
```

#### 制約
- 送信先のユーザーは事前にBotを友だち追加している必要がある
- Rate Limitあり（通常使用では問題なし）
- 無制限に送信可能（ただし月間メッセージ数に制限がある場合あり）

#### line-bot-sdk-goでの実装
```go
func pushMessage(userID, text string) error {
    bot, _ := messaging_api.NewMessagingApiAPI(os.Getenv("LINE_CHANNEL_TOKEN"))

    _, err := bot.PushMessage(
        &messaging_api.PushMessageRequest{
            To: userID,
            Messages: []messaging_api.MessageInterface{
                &messaging_api.TextMessage{
                    Text: text,
                },
            },
        },
    )
    return err
}
```

---

## データベース操作

### ユーザー取得
```go
type User struct {
    LineUserID       string
    Name             string
    Birthday         string
    RegistrationStep string
}

func GetUser(lineUserID string) (*User, error) {
    var user User
    err := db.QueryRow(
        "SELECT line_user_id, name, birthday, registration_step FROM users WHERE line_user_id = ?",
        lineUserID,
    ).Scan(&user.LineUserID, &user.Name, &user.Birthday, &user.RegistrationStep)

    if err == sql.ErrNoRows {
        return nil, nil // ユーザーが存在しない
    }
    if err != nil {
        return nil, err
    }
    return &user, nil
}
```

### 新規ユーザー作成
```go
func CreateUser(lineUserID string) error {
    _, err := db.Exec(
        "INSERT INTO users (line_user_id, name, registration_step) VALUES (?, '', 'awaiting_name')",
        lineUserID,
    )
    return err
}
```

### ユーザー情報更新
```go
func UpdateUser(lineUserID, name, birthday, step string) error {
    _, err := db.Exec(
        "UPDATE users SET name = ?, birthday = ?, registration_step = ? WHERE line_user_id = ?",
        name, birthday, step, lineUserID,
    )
    return err
}
```

### マッチング検索
```go
func CheckMatch(toName, toBirthday string) (string, error) {
    var matchedUserID string
    err := db.QueryRow(`
        SELECT l.from_user_id
        FROM likes l
        WHERE l.to_name = ? AND l.to_birthday = ? AND l.matched = 0
        LIMIT 1
    `, toName, toBirthday).Scan(&matchedUserID)

    if err == sql.ErrNoRows {
        return "", nil // マッチングなし
    }
    if err != nil {
        return "", err
    }
    return matchedUserID, nil
}
```

### Like登録（トランザクション）
```go
func RegisterLike(fromUserID, toName, toBirthday string) (bool, string, error) {
    tx, err := db.Begin()
    if err != nil {
        return false, "", err
    }
    defer tx.Rollback()

    // likes登録
    _, err = tx.Exec(
        "INSERT INTO likes (from_user_id, to_name, to_birthday) VALUES (?, ?, ?)",
        fromUserID, toName, toBirthday,
    )
    if err != nil {
        return false, "", err
    }

    // ユーザー情報取得（自分の名前と誕生日）
    var userName, userBirthday string
    err = tx.QueryRow(
        "SELECT name, birthday FROM users WHERE line_user_id = ?",
        fromUserID,
    ).Scan(&userName, &userBirthday)
    if err != nil {
        return false, "", err
    }

    // マッチング検索
    var matchedUserID string
    err = tx.QueryRow(`
        SELECT from_user_id FROM likes
        WHERE to_name = ? AND to_birthday = ? AND matched = 0
    `, userName, userBirthday).Scan(&matchedUserID)

    if err == sql.ErrNoRows {
        // マッチングなし
        tx.Commit()
        return false, "", nil
    }
    if err != nil {
        return false, "", err
    }

    // マッチング成立：両方を更新
    _, err = tx.Exec("UPDATE likes SET matched = 1 WHERE from_user_id = ?", fromUserID)
    if err != nil {
        return false, "", err
    }

    _, err = tx.Exec("UPDATE likes SET matched = 1 WHERE from_user_id = ?", matchedUserID)
    if err != nil {
        return false, "", err
    }

    tx.Commit()
    return true, matchedUserID, nil
}
```

---

## エラーハンドリング

### 考慮すべきエラー

#### 1. 署名検証失敗
```go
if !ValidateSignature(...) {
    log.Printf("Invalid signature from IP: %s", r.RemoteAddr)
    http.Error(w, "Invalid signature", http.StatusBadRequest)
    return
}
```

#### 2. データベースエラー
```go
user, err := GetUser(userID)
if err != nil {
    log.Printf("DB error: %v", err)
    replyMessage(replyToken, "エラーが発生しました。しばらくしてからもう一度試してください。")
    return
}
```

#### 3. LINE APIエラー
```go
err := replyMessage(replyToken, text)
if err != nil {
    log.Printf("LINE API error: %v", err)
    // ユーザーには通知できない（replyTokenが使えない）
    // ログに記録するのみ
}
```

#### 4. UNIQUE制約違反（好きな人の重複登録）
```go
_, err = tx.Exec("INSERT INTO likes ...")
if err != nil {
    if strings.Contains(err.Error(), "UNIQUE constraint failed") {
        replyMessage(replyToken, "すでに登録済みです")
        return
    }
    // その他のエラー
}
```

### エラーレスポンス方針
```
原則：
- LINEには常に200を返す（LINEからのリトライを避ける）
- エラーはログに記録
- ユーザーにはわかりやすいメッセージを返信
```

---

## 環境変数

### 必要な環境変数
```bash
# LINE Bot認証情報
LINE_CHANNEL_SECRET=your_channel_secret
LINE_CHANNEL_TOKEN=your_channel_access_token

# データベース
DATABASE_PATH=/home/ec2-user/cupid/cupid.db

# サーバー設定
PORT=8080
```

### 環境変数の設定方法

#### systemdサービスファイル
```ini
[Service]
Environment="LINE_CHANNEL_SECRET=xxx"
Environment="LINE_CHANNEL_TOKEN=yyy"
Environment="DATABASE_PATH=/home/ec2-user/cupid/cupid.db"
Environment="PORT=8080"
```

#### .envファイル（開発環境）
```bash
# .env
LINE_CHANNEL_SECRET=xxx
LINE_CHANNEL_TOKEN=yyy
DATABASE_PATH=./cupid.db
PORT=8080
```

---

## セキュリティ

### 1. Webhook署名検証（必須）
```
すべてのリクエストで署名を検証
検証失敗時は400を返す
```

### 2. SQLインジェクション対策
```go
// NG: 文字列結合
query := "SELECT * FROM users WHERE line_user_id = '" + userID + "'"

// OK: プレースホルダー使用
db.Query("SELECT * FROM users WHERE line_user_id = ?", userID)
```

### 3. 環境変数での秘密情報管理
```
Channel SecretとAccess Tokenは環境変数
ハードコーディング厳禁
Gitにコミット厳禁
```

### 4. HTTPS必須
```
LINEはHTTPSエンドポイントのみサポート
Let's Encryptで証明書取得
```

---

## パフォーマンス

### 想定負荷
```
同時アクセス: 1-2人
1日のリクエスト: 100回程度
レスポンスタイム: < 1秒
```

### ボトルネック
```
今回の規模では特になし
SQLiteの性能で十分

将来的にユーザーが増えたら:
- 接続プーリング（database/sql で自動）
- インデックスの最適化
- MySQL移行検討
```

---

## テスト

複数の携帯端末を用意するのは現実的じゃない、ね。以下の方法でテストできる、よ。

### 方法1: curlで直接Webhookを叩く（推奨）

#### 開発時の署名検証スキップ

開発時は署名検証を環境変数で制御する方法が簡単、ね。

```go
// main.goの署名検証部分
skipSignatureValidation := os.Getenv("SKIP_SIGNATURE_VALIDATION") == "true"

if !skipSignatureValidation {
    if !webhook.ValidateSignature(channelSecret, signature, body) {
        http.Error(w, "Invalid signature", http.StatusBadRequest)
        return
    }
} else {
    log.Println("⚠️ Signature validation skipped (development mode)")
}
```

**.envファイル（開発時）**:
```bash
SKIP_SIGNATURE_VALIDATION=true  # 開発時のみtrue、本番は削除
```

#### curlでのテスト例

**Follow Event（友だち追加）のテスト**:
```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "events": [
      {
        "type": "follow",
        "timestamp": 1609459200000,
        "source": {
          "type": "user",
          "userId": "U1234567890abcdef"
        },
        "replyToken": "test-reply-token-001",
        "mode": "active"
      }
    ]
  }'
```

**ユーザーA（しのざわひろ）の登録フロー**:
```bash
# 1. 名前入力
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "events": [{
      "type": "message",
      "message": {"type": "text", "id": "1", "text": "しのざわひろ"},
      "timestamp": 1609459200000,
      "source": {"type": "user", "userId": "U1111111111111111"},
      "replyToken": "test-reply-token-002",
      "mode": "active"
    }]
  }'

# 2. 生年月日入力
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "events": [{
      "type": "message",
      "message": {"type": "text", "id": "2", "text": "2009-12-21"},
      "timestamp": 1609459200000,
      "source": {"type": "user", "userId": "U1111111111111111"},
      "replyToken": "test-reply-token-003",
      "mode": "active"
    }]
  }'

# 3. 好きな人の名前入力
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "events": [{
      "type": "message",
      "message": {"type": "text", "id": "3", "text": "つきむらてまり"},
      "timestamp": 1609459200000,
      "source": {"type": "user", "userId": "U1111111111111111"},
      "replyToken": "test-reply-token-004",
      "mode": "active"
    }]
  }'

# 4. 好きな人の生年月日入力
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "events": [{
      "type": "message",
      "message": {"type": "text", "id": "4", "text": "2010-04-04"},
      "timestamp": 1609459200000,
      "source": {"type": "user", "userId": "U1111111111111111"},
      "replyToken": "test-reply-token-005",
      "mode": "active"
    }]
  }'
```

**ユーザーB（つきむらてまり）の登録フロー**:
```bash
# 同様の手順で userId: "U2222222222222222" として登録
# 好きな人: "しのざわひろ", "2009-12-21"
# → マッチング成立！
```

#### テスト用スクリプト（test.sh）

```bash
#!/bin/bash
API_URL="http://localhost:8080/webhook"

# ユーザーA登録
echo "=== ユーザーA（しのざわひろ）登録開始 ==="
curl -s -X POST $API_URL -H "Content-Type: application/json" -d '{"events":[{"type":"follow","timestamp":1609459200000,"source":{"type":"user","userId":"U1111111111111111"},"replyToken":"token-001","mode":"active"}]}'
sleep 1

curl -s -X POST $API_URL -H "Content-Type: application/json" -d '{"events":[{"type":"message","message":{"type":"text","id":"1","text":"しのざわひろ"},"timestamp":1609459200000,"source":{"type":"user","userId":"U1111111111111111"},"replyToken":"token-002","mode":"active"}]}'
sleep 1

curl -s -X POST $API_URL -H "Content-Type: application/json" -d '{"events":[{"type":"message","message":{"type":"text","id":"2","text":"2009-12-21"},"timestamp":1609459200000,"source":{"type":"user","userId":"U1111111111111111"},"replyToken":"token-003","mode":"active"}]}'
sleep 1

curl -s -X POST $API_URL -H "Content-Type: application/json" -d '{"events":[{"type":"message","message":{"type":"text","id":"3","text":"つきむらてまり"},"timestamp":1609459200000,"source":{"type":"user","userId":"U1111111111111111"},"replyToken":"token-004","mode":"active"}]}'
sleep 1

curl -s -X POST $API_URL -H "Content-Type: application/json" -d '{"events":[{"type":"message","message":{"type":"text","id":"4","text":"2010-04-04"},"timestamp":1609459200000,"source":{"type":"user","userId":"U1111111111111111"},"replyToken":"token-005","mode":"active"}]}'

echo "=== ユーザーB（つきむらてまり）登録開始 ==="
curl -s -X POST $API_URL -H "Content-Type: application/json" -d '{"events":[{"type":"follow","timestamp":1609459200000,"source":{"type":"user","userId":"U2222222222222222"},"replyToken":"token-101","mode":"active"}]}'
sleep 1

curl -s -X POST $API_URL -H "Content-Type: application/json" -d '{"events":[{"type":"message","message":{"type":"text","id":"11","text":"つきむらてまり"},"timestamp":1609459200000,"source":{"type":"user","userId":"U2222222222222222"},"replyToken":"token-102","mode":"active"}]}'
sleep 1

curl -s -X POST $API_URL -H "Content-Type: application/json" -d '{"events":[{"type":"message","message":{"type":"text","id":"12","text":"2010-04-04"},"timestamp":1609459200000,"source":{"type":"user","userId":"U2222222222222222"},"replyToken":"token-103","mode":"active"}]}'
sleep 1

curl -s -X POST $API_URL -H "Content-Type: application/json" -d '{"events":[{"type":"message","message":{"type":"text","id":"13","text":"しのざわひろ"},"timestamp":1609459200000,"source":{"type":"user","userId":"U2222222222222222"},"replyToken":"token-104","mode":"active"}]}'
sleep 1

curl -s -X POST $API_URL -H "Content-Type: application/json" -d '{"events":[{"type":"message","message":{"type":"text","id":"14","text":"2009-12-21"},"timestamp":1609459200000,"source":{"type":"user","userId":"U2222222222222222"},"replyToken":"token-105","mode":"active"}]}'

echo "=== テスト完了 ==="
```

実行方法：
```bash
chmod +x test.sh
./test.sh
```

---

### 方法2: 1台のスマホで複数アカウント切り替え

LINEアプリには複数アカウントを切り替える機能がある、ね。

**手順**:
1. メインアカウントでログイン
2. 設定 → アカウント → サブアカウント追加
3. 2つのアカウントでそれぞれBotを友だち追加
4. アカウントを切り替えながらマッチングテスト

**メリット**: 本物のLINEアプリで動作確認できる
**デメリット**: 切り替えが面倒

---

### 方法3: PCとスマホで別アカウント

**手順**:
1. スマホで1つ目のアカウント
2. PCでLINEアプリをインストール、別アカウントでログイン
3. 両方でBotを友だち追加してテスト

**メリット**: 同時操作できる
**デメリット**: PCに別のLINEアカウントが必要

---

### 方法4: LINE Developers Consoleのテスト機能

LINE Developers Consoleには「Messaging API」タブに**Webhook URLの検証機能**がある、ね。

**手順**:
1. LINE Developers Console → Messaging API設定
2. Webhook URL: `https://cupid.click/webhook`
3. 「Verify」ボタンをクリック → 200が返ればOK

ただし、これは単純な疎通確認のみ。実際の登録フローはテストできない。

---

### 推奨テストフロー

#### Phase 1: ローカルでcurlテスト（開発時）
- `SKIP_SIGNATURE_VALIDATION=true`で署名検証スキップ
- `test.sh`スクリプトで登録フローとマッチングを自動テスト
- SQLiteでデータ確認: `sqlite3 cupid.db "SELECT * FROM users;"`

#### Phase 2: EC2でcurlテスト（デプロイ後）
- 同様のcurlコマンドを`https://cupid.click/webhook`に送信
- 署名検証をスキップしたまま動作確認

#### Phase 3: 実機テスト（本番前）
- `SKIP_SIGNATURE_VALIDATION=true`を削除して署名検証を有効化
- 自分のLINEアカウントでBotを友だち追加
- 登録フローを手動で確認

#### Phase 4: マッチング実機テスト
- 友人に協力してもらう（1人でOK）
- お互いに登録してマッチング成立を確認
- **または**: 自分でPCとスマホの2アカウントでテスト

---

### テスト時の注意点

#### Push Message APIのモック

curlテストでは、Push Message APIの呼び出しは実際には送信されない、ね。ログで確認する、よ。

```go
// Push Message送信前にログ出力
log.Printf("Sending push message to user: %s, message: %s", matchedUserID, "相思相愛、みたい。おめでとう。")

err := pushMessage(matchedUserID, "相思相愛、みたい。おめでとう。")
if err != nil {
    log.Printf("Failed to push message: %v", err)
}
```

#### replyTokenは無視される

curlテストでは`replyToken`はダミー値でOK。Reply Message APIも実際には送信されないので、ログで確認する。

```go
log.Printf("Replying to user: %s, message: %s", userID, text)
```

---

### データベースの確認

テスト後、SQLiteで結果を確認、ね。

```bash
# EC2にSSH接続後
sqlite3 ~/cupid/cupid.db

# ユーザー確認
SELECT line_user_id, name, birthday, registration_step FROM users;

# Likes確認
SELECT from_user_id, to_name, to_birthday, matched FROM likes;

# マッチング確認（matched = 1）
SELECT COUNT(*) FROM likes WHERE matched = 1;

.quit
```

---

## まとめ

### 実装すべきエンドポイント
```
POST /webhook
- 署名検証
- イベントタイプチェック（Follow Event, Message Event）
- Follow Event処理（新規ユーザー登録）
- ユーザー状態管理（registration_step）
- マッチング処理（相思相愛判定）
- Reply/Push Message送信
```

### データベース操作
```
- ユーザーCRUD
- likes登録
- マッチング検索
- トランザクション管理
```

### 外部API連携
```
- LINE Reply Message API
- LINE Push Message API
- line-bot-sdk-go使用
```

### セキュリティ
```
- Webhook署名検証（必須）
- SQLインジェクション対策
- 環境変数管理
- HTTPS通信
```
