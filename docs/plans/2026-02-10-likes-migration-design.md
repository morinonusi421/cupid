# Likes Migration Design - likesテーブル削除とusersテーブル統合

## 概要

likesテーブルを削除し、好きな人の情報をusersテーブルに統合する。UNIQUE(from_user_id)制約により1人につき1つのlikeレコードしか持てないため、likesテーブルは冗長であり、usersテーブルにcrush情報を統合することでシンプルな設計にする。

## 現状の問題点

### 1. アーキテクチャの冗長性
- UNIQUE(from_user_id)制約で1対1の関係
- 1対1なのに別テーブルは不要
- JOIN が必要でクエリが複雑

### 2. 再登録の実装が困難
- ProcessTextMessageは「再登録できます」と案内
- RegisterCrushは常にCreateを呼ぶため、UNIQUE制約でエラー
- 再登録が実装されていない（バグ）

### 3. データ整合性の懸念
- ユーザーが名前を変更した場合
- 他の人が登録したlikes.to_nameは古い名前のまま
- マッチング検索時に不整合が発生する可能性

## 要件

### 機能要件

#### 自分の情報変更
- **目的**: 間違い修正（主）、本名変更（副）
- **変更可能項目**: name, birthday
- **既存データへの影響**: なし（他の人が登録した古い名前はそのまま）
- **マッチング中の変更**: 許可、確認メッセージ表示、マッチング解除

#### 好きな人の変更
- **目的**: 間違い修正、気持ちの変化
- **変更方法**: UPDATEで上書き
- **マッチング中の変更**: 許可、確認メッセージ表示、マッチング解除

#### マッチング解除
- **通知**: 両方のユーザーにメッセージ配信
- **タイミング**: 自分の情報変更時、または好きな人変更時
- **確認フロー**: LIFF確認画面で明示的に確認

### 非機能要件

- YAGNI原則に従う（複数の好きな人は実装しない）
- シンプルな設計（1テーブルで完結）
- 再登録を簡単に実装できる

## ユースケース

### 自分の情報変更
1. ユーザーAが名前を「タナカタロウ」で登録
2. 登録時に「タナカタロ」とタイポに気づく
3. LIFFフォームから「タナカタロウ」に修正
4. マッチング中なら確認画面表示 → 解除通知

### 好きな人の変更
1. ユーザーAが「ヤマダハナコ」を好きな人として登録
2. 気持ちが変わった、または間違いに気づいた
3. LIFFフォームから「サトウアキコ」に変更
4. マッチング中なら確認画面表示 → 解除通知

### 名前変更時のマッチング
1. ユーザーAが名前を「タナカタロ」（タイポ）で登録
2. ユーザーBが「タナカタロウ」（正しい名前）を好きな人として登録
3. マッチングしない
4. ユーザーAが「タナカタロウ」に修正
5. マッチング成立

## 設計

### スキーマ変更

#### 新しいusersテーブル

```sql
CREATE TABLE users (
  line_user_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  birthday TEXT NOT NULL,
  registration_step INTEGER NOT NULL DEFAULT 1,  -- 0を削除
  crush_name TEXT,                                -- 好きな人の名前
  crush_birthday TEXT,                            -- 好きな人の誕生日
  matched_with_user_id TEXT,                      -- マッチング相手のID
  registered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (matched_with_user_id) REFERENCES users(line_user_id)
);

-- マッチング検索用インデックス
CREATE INDEX idx_users_name_birthday ON users(name, birthday);
CREATE INDEX idx_users_crush ON users(crush_name, crush_birthday);

-- updated_at自動更新トリガー
CREATE TRIGGER update_users_updated_at
AFTER UPDATE ON users
FOR EACH ROW
BEGIN
  UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE line_user_id = NEW.line_user_id;
END;
```

#### 変更点

1. **crush_name, crush_birthdayを追加**: 好きな人の情報をusersに統合
2. **matched_with_user_idを追加**: マッチング相手のID（NULL=未マッチ）
3. **matchedフラグを削除**: matched_with_user_id IS NOT NULL で判定可能（冗長データ削除）
4. **registration_stepのDEFAULTを1に変更**: 0を削除（正常フローでは使わないため）
5. **likesテーブルを削除**: 完全に不要

### registration_stepの定義

#### 新しい定義

```
1: ユーザー登録完了（好きな人未登録）
2: 好きな人登録完了
```

#### 状態遷移

```
LIFF登録 → step=1 → 好きな人登録 → step=2
             ↓                    ↓
        [情報変更]          [好きな人変更]
      （stepは変わらず）   （stepは変わらず）
```

### マッチング判定ロジック

#### マッチング条件

```sql
SELECT * FROM users
WHERE name = ?                          -- A.crush_name
  AND birthday = ?                      -- A.crush_birthday
  AND crush_name = ?                    -- A.name
  AND crush_birthday = ?                -- A.birthday
  AND matched_with_user_id IS NULL      -- まだマッチしていない
LIMIT 1
```

#### マッチング成立時の処理

```go
1. A.matched_with_user_id = B.line_user_id
2. B.matched_with_user_id = A.line_user_id
3. 両方のユーザーに通知送信
```

### マッチング解除ロジック

#### マッチング解除の処理フロー

```go
// マッチング中かチェック
if user.MatchedWithUserID != "" {
    // 1. 確認メッセージ表示（LIFF側）
    "現在マッチング中です。変更するとマッチングが解除されますが、本当によろしいですか？"

    // 2. ユーザーがOKした場合（サーバー側）
    a. 相手のユーザー情報を取得（matched_with_user_id）
    b. 自分の matched_with_user_id を NULL に
    c. 相手の matched_with_user_id を NULL に
    d. 両方のユーザーに解除通知を送信
}

// 3. 情報を変更
user.Name = newName  // または crush_name など
```

### API設計（変更時の確認フロー）

#### RegisterFromLIFF（自分の情報変更）

**リクエスト：**
```json
{
  "name": "新しい名前",
  "birthday": "2000-01-15",
  "confirm_unmatch": false  // 初回送信時はfalse
}
```

**レスポンス（マッチング中の場合）：**
```json
{
  "error": "matched_user_exists",
  "message": "現在マッチング中です。変更するとマッチングが解除されます。",
  "matched_user_name": "相手の名前"
}
```

**再送信（確認後）：**
```json
{
  "name": "新しい名前",
  "birthday": "2000-01-15",
  "confirm_unmatch": true  // 確認済み
}
```

#### RegisterCrush（好きな人変更）

同様のフロー：
- 初回送信 → マッチング中ならエラー
- 確認後に `confirm_unmatch: true` で再送信

#### サーバー側の処理

```go
func (s *userService) RegisterFromLIFF(ctx, userID, name, birthday string, confirmUnmatch bool) error {
    user, _ := s.userRepo.FindByLineID(ctx, userID)

    // マッチング中かチェック
    if user.MatchedWithUserID != "" && !confirmUnmatch {
        return ErrMatchedUserExists  // 確認が必要
    }

    // マッチング解除処理
    if user.MatchedWithUserID != "" && confirmUnmatch {
        s.unmatchUsers(ctx, user.LineID, user.MatchedWithUserID)
    }

    // 情報を更新
    user.Name = name
    user.Birthday = birthday
    s.userRepo.Update(ctx, user)
}
```

### リポジトリ層の変更

#### 新しいマッチング検索メソッド

**現在（likesテーブル使用）：**
```go
FindMatchingLike(ctx, currentUser, like) (*model.User, error)
```

**新しい（usersテーブルのみ）：**
```go
FindMatchingUser(ctx, currentUser) (*model.User, error)
```

#### 実装

```go
func (r *userRepository) FindMatchingUser(ctx context.Context, currentUser *model.User) (*model.User, error) {
    // 相互にcrushしているユーザーを検索
    query := `
        SELECT * FROM users
        WHERE name = ?
          AND birthday = ?
          AND crush_name = ?
          AND crush_birthday = ?
          AND matched_with_user_id IS NULL
        LIMIT 1
    `

    var matchedUser model.User
    err := r.db.QueryRowContext(
        ctx,
        query,
        currentUser.CrushName,        // 自分が好きな人の名前
        currentUser.CrushBirthday,    // 自分が好きな人の誕生日
        currentUser.Name,              // 相手が好きな人として登録している名前
        currentUser.Birthday,          // 相手が好きな人として登録している誕生日
    ).Scan(&matchedUser)

    if err == sql.ErrNoRows {
        return nil, nil  // マッチングなし
    }
    if err != nil {
        return nil, err
    }

    return &matchedUser, nil
}
```

#### 削除されるコード

- `LikeRepository` 全体（インターフェース、実装、テスト）
- `internal/repository/like_repo.go`
- `internal/repository/like_repo_test.go`
- `internal/model/like.go`
- `internal/model/like_test.go`

### 通知メッセージ設計

#### マッチング成立時（変更なし）

```
相思相愛が成立しました！
相手：○○さん
```

#### マッチング解除時（新規）

**自分が情報を変更した場合：**
```
マッチングが解除されました。

理由：あなたが情報を変更しました
相手：○○さん
```

**相手が情報を変更した場合：**
```
マッチングが解除されました。

理由：相手が情報を変更しました
相手：○○さん
```

#### LIFF確認画面のメッセージ

**自分の情報変更時：**
```html
<h2>⚠️ マッチング中です</h2>
<p>現在、<strong>○○さん</strong>とマッチング中です。</p>
<p>情報を変更すると、マッチングが解除されます。</p>
<p>本当に変更しますか？</p>

<button>変更する</button>
<button>キャンセル</button>
```

**好きな人変更時：**
```html
<h2>⚠️ マッチング中です</h2>
<p>現在、<strong>○○さん</strong>とマッチング中です。</p>
<p>好きな人を変更すると、マッチングが解除されます。</p>
<p>本当に変更しますか？</p>

<button>変更する</button>
<button>キャンセル</button>
```

## マイグレーション計画

### マイグレーションファイル

**ファイル名：**
```
db/migrations/20260210000001-likes-to-users-migration.sql
```

### マイグレーション内容

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

## 影響を受けるファイル

### 実装

1. **スキーマ**
   - `db/migrations/20260210000001-likes-to-users-migration.sql` - 新規作成

2. **モデル**
   - `internal/model/user.go` - CrushName, CrushBirthday, MatchedWithUserID追加
   - `internal/model/like.go` - 削除
   - `internal/model/like_test.go` - 削除

3. **リポジトリ**
   - `internal/repository/user_repo.go` - FindMatchingUser追加
   - `internal/repository/like_repo.go` - 削除
   - `internal/repository/like_repo_test.go` - 削除

4. **サービス**
   - `internal/service/user_service.go` - RegisterFromLIFF, RegisterCrushにconfirm_unmatch追加
   - `internal/service/matching_service.go` - FindMatchingLikeをFindMatchingUserに変更
   - unmatchUsers処理追加
   - sendUnmatchNotification追加

5. **ハンドラ**
   - `internal/handler/registration_api.go` - confirm_unmatchパラメータ追加
   - `internal/handler/crush_registration_api.go` - confirm_unmatchパラメータ追加

6. **LIFF**
   - `public/liff/register.html` - 確認画面の実装
   - `public/crush/register.html` - 確認画面の実装

### テスト

7. **サービス層テスト**
   - `internal/service/user_service_test.go` - confirm_unmatchのテスト追加
   - `internal/service/matching_service_test.go` - FindMatchingUserのテスト追加

8. **リポジトリ層テスト**
   - `internal/repository/user_repo_test.go` - FindMatchingUserのテスト追加

## 実装順序

1. マイグレーションファイル作成
2. モデル変更（User構造体にCrush関連フィールド追加、Like削除）
3. リポジトリ変更（FindMatchingUser追加、LikeRepository削除）
4. サービス変更（confirm_unmatch対応、unmatchUsers追加）
5. ハンドラ変更（confirm_unmatchパラメータ追加）
6. LIFF変更（確認画面実装）
7. テスト追加・修正
8. マイグレーション実行
9. 全テスト実行・確認
10. ビルド確認
11. デプロイ

## 期待される効果

### アーキテクチャの改善
- シンプルな設計（1テーブルで完結）
- JOIN不要、クエリが高速
- 冗長データの削除（matched フラグ）

### 機能の改善
- 再登録が正しく実装される（バグ修正）
- マッチング解除の明確なフロー
- ユーザーへの適切な確認メッセージ

### 保守性の向上
- コードが簡潔になる（LikeRepository削除）
- データの一貫性が保ちやすい
- 将来の拡張が必要なら、その時にlikesテーブルを追加すればいい（YAGNI）
