# 好きな人登録機能 設計書

**作成日**: 2026-02-03
**ステータス**: 設計完了、実装待ち

## 概要

ユーザーが好きな人を登録し、相手も自分を登録していた場合に自動的にマッチング通知を送る機能。

## 確定した仕様

### 基本方針
1. Web登録フォームで好きな人の名前・誕生日を入力
2. 相手も自分を登録していたら自動的に通知
3. 通知はLINEメッセージで相手の名前を含める
4. 再登録機能（好きな人の変更）はTODO

### 制約条件
- 1人のユーザーは1人だけ好きな人を登録可能（UNIQUE制約）
- マッチング判定は名前と誕生日の完全一致
- 自己登録は防ぐ（バリデーション）
- 好きな人が未登録でもエラーにしない（後でマッチング可能）

### ユーザーフロー
1. ユーザーがLINE Botにメッセージ送信
2. Bot が登録状態を確認（RegistrationStep = 1？）
3. 登録済みなら好きな人登録フォームURLを送信
4. ユーザーがWebフォームで名前・誕生日を入力
5. サーバーがマッチング判定
6. マッチング時は両方のユーザーにLINE通知

## アーキテクチャ

### コンポーネント構成

既存の3層アーキテクチャを維持：

#### 1. フロントエンド層
- `static/crush/register.html` - 好きな人登録フォーム
- `static/crush/register.css` - スタイル
- `static/crush/register.js` - フォーム送信とバリデーション

#### 2. API層（handler）
- `CrushRegistrationAPIHandler` 新規作成
- エンドポイント: `POST /api/register-crush`
- 責務: リクエスト検証、Service呼び出し、レスポンス返却

#### 3. ビジネスロジック層（service）
- `UserService`に新メソッド: `RegisterCrush(ctx, userID, crushName, crushBirthday) (matchedUserID string, error)`
- マッチング判定ロジック
- LINE通知送信

#### 4. データアクセス層（repository）
- `LikeRepository` 新規作成
- メソッド: `Create`, `FindByFromUserID`, `FindMatchingLike`, `UpdateMatched`

#### 5. インフラ
- Nginx: `/crush/` パスで `static/crush/` を配信

## データフロー

### 好きな人登録フロー
```
User → Web Form → POST /api/register-crush
  → CrushRegistrationAPIHandler.RegisterCrush()
  → UserService.RegisterCrush()
    → バリデーション（自己登録チェック）
    → LikeRepository.Create() (likesテーブルにINSERT)
    → マッチング判定
      → UserRepository.FindByNameAndBirthday() (相手がusersに存在？)
      → LikeRepository.FindMatchingLike() (相手も自分を登録済み？)
      → マッチング時:
        - 両方のlike.matchedを1に更新
        - 両方のユーザーにLINE通知
  → レスポンス返却 { matched: bool, message: string }
```

### マッチング判定ロジック

```go
// 1. 好きな人Bが users テーブルに存在するか確認
crushUser := userRepo.FindByNameAndBirthday(crushName, crushBirthday)
if crushUser == nil {
    // 存在しない → マッチング不可（エラーにはしない）
    return "", nil
}

// 2. Bさんが Aさん（自分）を登録しているか確認
reverseLike := likeRepo.FindMatchingLike(
    fromUserID: crushUser.LineID,
    toName: currentUser.Name,
    toBirthday: currentUser.Birthday
)

if reverseLike != nil {
    // マッチング成立！
    // 両方のlikeレコードのmatchedを1に更新
    likeRepo.UpdateMatched(currentLike.ID, 1)
    likeRepo.UpdateMatched(reverseLike.ID, 1)

    // 両方にLINE通知
    linebot.PushMessage(currentUser.LineID, "山田太郎さんとマッチしました！💘")
    linebot.PushMessage(crushUser.LineID, "佐藤花子さんとマッチしました！💘")

    return crushUser.LineID, nil
}

return "", nil // マッチングなし
```

## API設計

### POST /api/register-crush

**リクエスト:**
```json
{
  "user_id": "U1234567890",
  "crush_name": "山田太郎",
  "crush_birthday": "1990-01-01"
}
```

**バリデーション:**
- `user_id`: 必須、空文字禁止
- `crush_name`: 必須、1-50文字
- `crush_birthday`: 必須、YYYY-MM-DD形式
- 自己登録チェック: `crush_name == user.Name && crush_birthday == user.Birthday` ならエラー

**レスポンス（成功）:**
```json
{
  "status": "ok",
  "matched": true,
  "message": "山田太郎さんとマッチしました！💘"
}
```

**レスポンス（マッチングなし）:**
```json
{
  "status": "ok",
  "matched": false,
  "message": "登録しました。相手があなたを登録したらマッチングします。"
}
```

**エラーレスポンス:**
```json
{
  "error": "自分自身は登録できません"
}
```

## データベース

### likesテーブル（既存）

```sql
CREATE TABLE likes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  from_user_id TEXT NOT NULL,
  to_name TEXT NOT NULL,
  to_birthday TEXT NOT NULL,
  matched INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (from_user_id) REFERENCES users(line_user_id),
  UNIQUE(from_user_id)
);

CREATE INDEX idx_likes_to_name_birthday ON likes(to_name, to_birthday);
```

**変更不要** - 既存のスキーマで実装可能

### 必要なクエリ

1. **好きな人の登録（INSERT）**
```sql
INSERT INTO likes (from_user_id, to_name, to_birthday)
VALUES (?, ?, ?)
ON CONFLICT(from_user_id) DO UPDATE SET
  to_name = excluded.to_name,
  to_birthday = excluded.to_birthday,
  matched = 0
```

2. **マッチング相手の検索（SELECT）**
```sql
SELECT * FROM likes
WHERE from_user_id = ?
  AND to_name = ?
  AND to_birthday = ?
```

3. **マッチングフラグ更新（UPDATE）**
```sql
UPDATE likes SET matched = 1 WHERE id = ?
```

## フロントエンド

### HTML構成

`static/crush/register.html` は `static/liff/register.html` とほぼ同じ構成：
- タイトル: 「好きな人登録」
- フィールド: 名前、生年月日
- バリデーション: クライアント側で基本チェック
- URLパラメータで`user_id`取得（セキュリティTODO）

### JavaScript

```javascript
async function registerCrush(name, birthday) {
    const userId = getUserIdFromURL();
    if (!userId) {
        throw new Error('ユーザーIDが見つかりません');
    }

    const response = await fetch('/api/register-crush', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
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

    return data;
}
```

## エラーハンドリング

### バリデーションエラー
- 自己登録: 「自分自身は登録できません」
- 空フィールド: 「名前と誕生日を入力してください」
- 不正な日付形式: 「正しい日付を入力してください」

### システムエラー
- DB接続エラー: 500 Internal Server Error
- LINE API エラー: ログに記録、ユーザーには通知済みと表示

### エッジケース
- 好きな人が未登録: エラーにせず、登録のみ実行
- すでに登録済み: 上書き（TODO: 再登録機能を後で実装）
- 同時マッチング: トランザクション不要（SQLiteの制約で自然に解決）

## セキュリティ

### 既存の問題（TODO）
- URLパラメータでuser_idを受け取る（なりすまし可能）
- 将来的にワンタイムトークン方式に変更

### 新規の考慮点
- 自己登録の防止（バリデーション実装）
- SQL injection: プリペアドステートメント使用
- XSS: フォーム入力のエスケープ

## テスト戦略

### ユニットテスト
1. `UserService.RegisterCrush()`
   - 正常系: マッチング成立
   - 正常系: マッチングなし（相手未登録）
   - 異常系: 自己登録エラー
   - 異常系: バリデーションエラー

2. `LikeRepository`
   - Create, FindByFromUserID, FindMatchingLike, UpdateMatched

### 統合テスト
- エンドツーエンド: フォーム送信 → マッチング → LINE通知

## 実装の順序

1. Model: `Like` struct 作成
2. Repository: `LikeRepository` 実装
3. Service: `UserService.RegisterCrush()` 実装
4. Handler: `CrushRegistrationAPIHandler` 実装
5. フロントエンド: HTML/CSS/JS
6. main.go: ルート追加
7. Nginx: 設定追加
8. テスト: ユニットテスト、統合テスト
9. デプロイ

## TODO（将来の改善）

- [ ] 再登録機能（好きな人の変更）
- [ ] ワンタイムトークン方式でセキュリティ改善
- [ ] マッチング履歴の表示
- [ ] マッチング解除機能
- [ ] 複数人の好きな人登録（UNIQUE制約削除）

## 参考

- 既存実装: `static/liff/register.html`, `internal/handler/registration_api.go`
- LINE Messaging API: Push Message
- データベース: `db/migrations/20260117000001-initial_schema.sql`
