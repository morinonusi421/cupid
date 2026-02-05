# LINE LIFF認証によるなりすまし対策設計

## 背景と問題

### 現在の脆弱性

現在の登録フローでは、URLパラメータに直接`user_id`を含めている：

```
https://cupid-linebot.click/liff/register.html?user_id=U1234567890abcdef
```

**問題点：**
- URLを知っていれば、誰でも他人のuser_idを使って登録できる
- user_idを書き換えるだけでなりすまし可能
- URLが漏洩すると、本人以外が登録できてしまう

### 影響範囲

以下の5箇所にTODOコメントが存在：
- `static/liff/register.js`
- `internal/service/user_service.go`
- `internal/handler/registration_api.go`
- `internal/handler/crush_registration_api.go`
- `docs/plans/2026-02-03-crush-registration.md`

すべて「ワンタイムトークン方式に変更する」という内容。

## 選択したアプローチ

### 検討した3つのアプローチ

1. **LINE LIFF + アクセストークン検証**（採用）
2. サーバー生成のワンタイムトークン + DB管理
3. HMAC署名付きトークン（JWT風）

### 採用理由

**LINE LIFF方式を選択した理由：**
- 既に`internal/liff/verifier.go`が実装済み
- LINE公式の仕組みで最もセキュア
- URLにuser_idやトークンを含めないため、URL漏洩リスクなし
- なりすまし不可能（トークンはLINEが発行・検証可能）
- 将来的にLIFFの他の機能（プロフィール取得など）も使える

## アーキテクチャ概要

### 現在のフロー（脆弱）

```
1. ユーザー → LINE Bot: メッセージ送信
2. Bot → ユーザー: URL返信（user_id含む）
   例: https://cupid-linebot.click/liff/register.html?user_id=U123
3. ユーザー → フォーム: アクセス＆入力
4. フォーム → API: POST /api/register { user_id: "U123", ... }
   ↑ここでなりすまし可能（URLを書き換えれば別人になれる）
```

### 新しいフロー（セキュア）

```
1. ユーザー → LINE Bot: メッセージ送信
2. Bot → ユーザー: LIFF URL返信
   例: https://liff.line.me/{liff-id}
3. LIFF SDK起動 → LINE認証
4. フォーム: liff.getAccessToken() でトークン取得
5. フォーム → API: POST /api/register
   Header: Authorization: Bearer {access_token}
   Body: { name: "...", birthday: "..." }
   ↑ user_idは含めない
6. API: トークン検証 → user_id取得 → 登録処理
```

**重要な変更点：**
- URLにuser_idを含めない
- フロントエンドでLINE公式のアクセストークンを取得
- バックエンドでトークンを検証してuser_idを安全に取得
- なりすまし不可能（トークンはLINEが発行、検証可能）

## LIFF設定

### LINE Developers設定（必要な作業）

**2つのLIFF appを登録：**

#### 1. ユーザー登録用LIFF app
- 名前: `Cupid - ユーザー登録`
- サイズ: Full（全画面）
- エンドポイントURL: `https://cupid-linebot.click/liff/register.html`
- Scope: `profile` (ユーザー情報取得)

#### 2. 好きな人登録用LIFF app
- 名前: `Cupid - 好きな人登録`
- サイズ: Full（全画面）
- エンドポイントURL: `https://cupid-linebot.click/crush/register.html`
- Scope: `profile`

登録すると、それぞれ**LIFF ID**が発行される（例: `1234567890-AbCdEfGh`）

### 環境変数の追加

`.env`に以下を追加：

```bash
LINE_LIFF_REGISTER_ID=1234567890-AbCdEfGh      # ユーザー登録用
LINE_LIFF_CRUSH_REGISTER_ID=9876543210-XyZwVu  # 好きな人登録用
LINE_LIFF_CHANNEL_ID=2008809168                # 既存（検証用）
```

### Bot側のURL生成

`internal/service/user_service.go`の修正：

#### handleInitialMessage()

```go
// 変更前（脆弱）
registerURL := fmt.Sprintf("%s?user_id=%s", s.liffRegisterURL, user.LineID)
return fmt.Sprintf("初めまして！💘\n\n下のリンクから登録してね。\n\n%s", registerURL), nil

// 変更後（セキュア）
liffURL := fmt.Sprintf("https://liff.line.me/%s", os.Getenv("LINE_LIFF_REGISTER_ID"))
return fmt.Sprintf("初めまして！💘\n\n下のリンクから登録してね。\n\n%s", liffURL), nil
```

#### ProcessTextMessage() - 好きな人登録案内

```go
// 変更前（脆弱）
crushRegisterURL := fmt.Sprintf("https://cupid-linebot.click/crush/register.html?user_id=%s", userID)

// 変更後（セキュア）
liffURL := fmt.Sprintf("https://liff.line.me/%s", os.Getenv("LINE_LIFF_CRUSH_REGISTER_ID"))
return fmt.Sprintf("次に、好きな人を登録してください💘\n\n%s", liffURL), nil
```

**重要な違い：**
- 従来: `https://cupid-linebot.click/liff/register.html?user_id=U123`（脆弱）
- LIFF: `https://liff.line.me/1234567890-AbCdEfGh`（セキュア）

LIFF URLにアクセスすると、LINEが自動的に認証して、設定したエンドポイントURL（`https://cupid-linebot.click/liff/register.html`）にリダイレクトされる。

## フロントエンド実装

### LIFF SDKの導入

`static/liff/register.html`と`static/crush/register.html`の両方に追加：

```html
<head>
    <!-- 既存の内容 -->
    <script charset="utf-8" src="https://static.line-scdn.net/liff/edge/2/sdk.js"></script>
</head>
```

### register.js の変更

#### 1. LIFF初期化

```javascript
// ページ読み込み時
window.addEventListener('load', async () => {
    try {
        // LIFF ID取得（環境変数から読み込む想定、または直接埋め込み）
        const liffId = 'LINE_LIFF_REGISTER_ID'; // 実際の値に置き換え

        await liff.init({ liffId: liffId });

        if (!liff.isLoggedIn()) {
            liff.login(); // 未ログインならLINEログイン画面へ
            return;
        }

        setupForm(); // ログイン済みならフォーム表示
    } catch (error) {
        console.error('LIFF initialization failed', error);
        showMessage('LINE認証に失敗しました。再度お試しください。', 'error');
    }
});
```

#### 2. アクセストークン取得とAPI呼び出し

```javascript
async function registerUser(name, birthday) {
    try {
        showLoading(true);
        submitButton.disabled = true;

        // アクセストークン取得
        const accessToken = liff.getAccessToken();

        if (!accessToken) {
            throw new Error('認証情報が取得できませんでした');
        }

        // API呼び出し（user_idは送らない）
        const response = await fetch('/api/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${accessToken}` // ★トークンをヘッダーで送信
            },
            body: JSON.stringify({ name, birthday }) // user_id削除
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || '登録に失敗しました。');
        }

        // 成功
        showMessage('登録が完了しました！LINEに戻って話しかけてね。', 'success');

    } catch (error) {
        console.error('Registration failed', error);
        showMessage(error.message || '登録に失敗しました。', 'error');
        submitButton.disabled = false;
    } finally {
        showLoading(false);
    }
}
```

#### 3. 不要なコード削除

```javascript
// 以下の関数を削除
// function getUserIdFromURL() { ... }

// TODOコメントも削除
```

### crush/register.js の変更

同様の変更を適用：
- LIFF初期化（`LINE_LIFF_CRUSH_REGISTER_ID`を使用）
- `liff.getAccessToken()`でトークン取得
- Authorizationヘッダーでトークン送信
- `user_id`をリクエストボディから削除

## バックエンド実装

### registration_api.go の変更

#### 1. Authorizationヘッダーからトークン取得

```go
package handler

import (
    "encoding/json"
    "log"
    "net/http"
    "strings"

    "github.com/morinonusi421/cupid/internal/service"
)

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
    userID, err := h.userService.VerifyLIFFToken(token)
    if err != nil {
        log.Printf("Token verification failed: %v", err)
        w.WriteHeader(http.StatusUnauthorized)
        json.NewEncoder(w).Encode(map[string]string{"error": "認証に失敗しました"})
        return
    }

    // リクエストボディからname, birthdayのみ取得
    var req RegisterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("Failed to decode request: %v", err)
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
        return
    }

    // user_idはトークンから取得したものを使用
    if err := h.userService.RegisterFromLIFF(r.Context(), userID, req.Name, req.Birthday); err != nil {
        log.Printf("Failed to register user: %v", err)
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
        return
    }

    log.Printf("Registration successful for user %s: name=%s, birthday=%s", userID, req.Name, req.Birthday)

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

#### 2. RegisterRequest構造体の変更

```go
// 変更前
type RegisterRequest struct {
    UserID   string `json:"user_id"`   // 削除
    Name     string `json:"name"`
    Birthday string `json:"birthday"`
}

// 変更後
type RegisterRequest struct {
    Name     string `json:"name"`
    Birthday string `json:"birthday"`
}
// user_idはAuthorizationヘッダーから取得するため不要
```

#### 3. TODOコメント削除

```go
// 以下のコメントを削除
// TODO: セキュリティ改善 - ワンタイムトークン方式に変更する
// 現在はリクエストボディに直接user_idを含めているが、なりすまし可能
// 将来的にはサーバー生成のワンタイムトークンを使用すべき
```

### crush_registration_api.go の変更

同様の変更を適用：

```go
func (h *CrushRegistrationAPIHandler) RegisterCrush(w http.ResponseWriter, r *http.Request) {
    // Authorizationヘッダーからトークン取得
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        w.WriteHeader(http.StatusUnauthorized)
        json.NewEncoder(w).Encode(map[string]string{"error": "認証が必要です"})
        return
    }

    token := strings.TrimPrefix(authHeader, "Bearer ")
    if token == authHeader {
        w.WriteHeader(http.StatusUnauthorized)
        json.NewEncoder(w).Encode(map[string]string{"error": "無効な認証形式です"})
        return
    }

    // トークン検証してuser_id取得
    userID, err := h.userService.VerifyLIFFToken(token)
    if err != nil {
        log.Printf("Token verification failed: %v", err)
        w.WriteHeader(http.StatusUnauthorized)
        json.NewEncoder(w).Encode(map[string]string{"error": "認証に失敗しました"})
        return
    }

    // リクエストボディをデコード（UserIDフィールド削除）
    var req RegisterCrushRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("Failed to decode request: %v", err)
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
        return
    }

    // バリデーション（UserIDチェック削除）
    if req.CrushName == "" || req.CrushBirthday == "" {
        log.Println("Missing crush_name or crush_birthday in request")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "crush_name and crush_birthday are required"})
        return
    }

    // サービス呼び出し（userIDはトークンから取得）
    matched, matchedName, err := h.userService.RegisterCrush(r.Context(), userID, req.CrushName, req.CrushBirthday)

    // 以下同じ
    // ...
}
```

```go
// 変更前
type RegisterCrushRequest struct {
    UserID        string `json:"user_id"`   // 削除
    CrushName     string `json:"crush_name"`
    CrushBirthday string `json:"crush_birthday"`
}

// 変更後
type RegisterCrushRequest struct {
    CrushName     string `json:"crush_name"`
    CrushBirthday string `json:"crush_birthday"`
}
```

### 既存のLIFF Verifierを活用

- `internal/liff/verifier.go`は既に実装済み
- `internal/service/user_service.go`の`VerifyLIFFToken()`メソッドも既に存在
- これらをそのまま使える（変更不要）

**既存実装の確認：**

```go
// internal/liff/verifier.go
func (v *Verifier) VerifyAccessToken(accessToken string) (string, error) {
    // LINE APIでトークン検証
    url := "https://api.line.me/oauth2/v2.1/verify?access_token=" + accessToken

    resp, err := http.Get(url)
    if err != nil {
        return "", fmt.Errorf("failed to verify token: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("token verification failed: %s", string(body))
    }

    var verifyResp VerifyResponse
    if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
        return "", fmt.Errorf("failed to decode response: %w", err)
    }

    // Channel ID検証
    if verifyResp.ClientID != v.channelID {
        return "", fmt.Errorf("channel ID mismatch")
    }

    return verifyResp.Sub, nil // user_idを返す
}
```

## エラーハンドリング

### フロントエンド

#### 1. LIFF初期化失敗

```javascript
try {
    await liff.init({ liffId: liffId });
} catch (error) {
    console.error('LIFF initialization failed', error);
    showMessage('LINE認証の初期化に失敗しました。再度お試しください。', 'error');
}
```

#### 2. ログインしていない

```javascript
if (!liff.isLoggedIn()) {
    liff.login(); // 自動的にLINEログイン画面へリダイレクト
    return;
}
```

#### 3. トークン取得失敗

```javascript
const accessToken = liff.getAccessToken();
if (!accessToken) {
    showMessage('認証情報が取得できませんでした', 'error');
    return;
}
```

#### 4. API呼び出し失敗

```javascript
if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.error || '登録に失敗しました。');
}
```

### バックエンド

#### 1. Authorizationヘッダーなし

```go
if authHeader == "" {
    w.WriteHeader(http.StatusUnauthorized)
    json.NewEncoder(w).Encode(map[string]string{
        "error": "認証が必要です"
    })
    return
}
```

#### 2. Bearer形式でない

```go
token := strings.TrimPrefix(authHeader, "Bearer ")
if token == authHeader {
    w.WriteHeader(http.StatusUnauthorized)
    json.NewEncoder(w).Encode(map[string]string{
        "error": "無効な認証形式です"
    })
    return
}
```

#### 3. トークン検証失敗

```go
userID, err := h.userService.VerifyLIFFToken(token)
if err != nil {
    log.Printf("Token verification failed: %v", err)
    w.WriteHeader(http.StatusUnauthorized)
    json.NewEncoder(w).Encode(map[string]string{
        "error": "認証に失敗しました。LINEからやり直してください。"
    })
    return
}
```

## テスト戦略

### 1. 単体テスト（registration_api_test.go）

モックでトークン検証をシミュレート：

```go
func TestRegistrationAPI_Register_Success(t *testing.T) {
    // モック設定
    mockUserService := &MockUserService{
        verifyTokenFunc: func(token string) (string, error) {
            if token == "valid-token" {
                return "U1234567890abcdef", nil
            }
            return "", fmt.Errorf("invalid token")
        },
    }

    handler := NewRegistrationAPIHandler(mockUserService)

    // リクエスト作成
    req := httptest.NewRequest(http.MethodPost, "/api/register",
        strings.NewReader(`{"name":"ヤマダタロウ","birthday":"1990-01-01"}`))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer valid-token")

    rec := httptest.NewRecorder()
    handler.Register(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRegistrationAPI_Register_Unauthorized(t *testing.T) {
    // Authorizationヘッダーなし
    req := httptest.NewRequest(http.MethodPost, "/api/register",
        strings.NewReader(`{"name":"ヤマダタロウ","birthday":"1990-01-01"}`))
    req.Header.Set("Content-Type", "application/json")

    rec := httptest.NewRecorder()
    handler.Register(rec, req)

    assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
```

### 2. E2Eテスト（integration_test.go）

**問題：**
- LIFF環境のモック化が複雑
- 実際のLIFF初期化が必要

**対応策：**
- E2Eテストでは従来のmockLineBotClientを使用
- LIFF関連のテストは手動テストで実施
- または、テスト用のLIFF appを用意

### 3. 手動テスト

**手順：**
1. LINE DevelopersでLIFF app登録
2. `.env`にLIFF ID設定
3. コードデプロイ
4. 実際のLINE環境からbotにメッセージ送信
5. LIFF URLをタップ
6. LINE認証画面が表示されることを確認
7. フォーム入力・送信
8. 登録成功を確認
9. DB確認（正しいuser_idで登録されているか）

## マイグレーション

### DBスキーマ変更

**なし：**
- 既存のテーブル構造は変更不要
- user_idの取得方法が変わるだけ

### 既存データへの影響

**なし：**
- 既存ユーザーデータはそのまま
- 新しい登録から新フローを使用
- 既存ユーザーが再登録する場合も新フローで問題なし

## デプロイ手順

### 1. LINE DevelopersでLIFF app登録

1. LINE Developers Consoleにログイン
2. Messaging API Channelを選択
3. 「LIFF」タブを開く
4. 「追加」ボタンをクリック

**ユーザー登録用：**
- LIFF app name: `Cupid - ユーザー登録`
- Size: Full
- Endpoint URL: `https://cupid-linebot.click/liff/register.html`
- Scope: `profile`
- Bot link feature: On（推奨）

**好きな人登録用：**
- LIFF app name: `Cupid - 好きな人登録`
- Size: Full
- Endpoint URL: `https://cupid-linebot.click/crush/register.html`
- Scope: `profile`
- Bot link feature: On（推奨）

5. LIFF IDをコピー（例: `1234567890-AbCdEfGh`）

### 2. 環境変数設定

ローカルの`.env`とEC2の`.env`に追加：

```bash
LINE_LIFF_REGISTER_ID=1234567890-AbCdEfGh
LINE_LIFF_CRUSH_REGISTER_ID=9876543210-XyZwVu
```

### 3. コード修正

以下のファイルを修正：
- `static/liff/register.html`（LIFF SDK追加）
- `static/liff/register.js`（LIFF初期化、トークン取得）
- `static/crush/register.html`（LIFF SDK追加）
- `static/crush/register.js`（LIFF初期化、トークン取得）
- `internal/handler/registration_api.go`（Authorizationヘッダー検証）
- `internal/handler/crush_registration_api.go`（Authorizationヘッダー検証）
- `internal/service/user_service.go`（URL生成変更）

### 4. テスト実行

```bash
# ローカルでテスト
make test

# 手動テスト用に一時的にローカル起動
make run

# ngrokでトンネル（手動テストの場合）
# ngrok http 8080
```

### 5. EC2にデプロイ

```bash
# コミット＆プッシュ
git add .
git commit -m "feat: implement LIFF authentication to prevent impersonation"
git push origin main

# EC2にデプロイ
make deploy
```

### 6. 手動テスト（実際のLINE環境）

1. LINEアプリでbotにメッセージ送信
2. botから返ってきたLIFF URLをタップ
3. LINE認証画面が表示されることを確認
4. 認証後、登録フォームが表示されることを確認
5. フォーム入力・送信
6. 登録成功メッセージを確認
7. EC2でDB確認：
   ```bash
   ssh cupid-bot
   cd ~/cupid
   sqlite3 cupid.db "SELECT * FROM users ORDER BY id DESC LIMIT 1;"
   ```
8. 正しいuser_idで登録されているか確認

### 7. 動作確認チェックリスト

- [ ] LIFF URLをタップするとLINE認証画面が表示される
- [ ] 認証後、登録フォームが表示される
- [ ] 名前のカタカナバリデーションが動作する
- [ ] 登録完了メッセージが表示される
- [ ] DBに正しいuser_idで登録されている
- [ ] なりすまし不可能（他人のURLでは登録できない）
- [ ] 既存機能（マッチング通知など）が正常動作する

## セキュリティ強化ポイント

### 1. なりすまし防止

**変更前：**
- URLパラメータのuser_idを信頼
- 誰でもURLを書き換えてなりすまし可能

**変更後：**
- LINE公式のアクセストークンで認証
- トークンはLINEが発行、改ざん不可能
- トークン検証で正しいuser_idを取得

### 2. URL漏洩対策

**変更前：**
- URLにuser_idが含まれる
- URLが漏洩すると他人が登録できる

**変更後：**
- URLにuser_idやトークンを含めない
- LIFF URLは公開されても問題なし
- アクセストークンは動的に取得、URLには含まれない

### 3. トークン検証

**実装：**
- LINE公式API（`https://api.line.me/oauth2/v2.1/verify`）で検証
- Channel IDも検証（他のLIFF appのトークンを拒否）
- トークン期限切れも自動的に検出

## まとめ

### 解決した問題

✅ なりすまし脆弱性を完全に解決
✅ URLパラメータからuser_idを削除
✅ LINE公式の仕組みでセキュア認証
✅ URL漏洩リスクを排除

### 追加の利点

✅ 既存のLIFF Verifierを活用（実装済み）
✅ 将来的にLIFFの他の機能も使える
✅ LINE公式のベストプラクティスに準拠

### 今後の拡張性

- LIFF Profile APIでユーザー名を自動取得
- LIFF Send Messagesでトーク画面にメッセージ送信
- LIFF Shareで友達招待機能
- LIFF CloseWindowで自動的にトーク画面に戻る
