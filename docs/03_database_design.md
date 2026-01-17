# データベース設計

## SQLite データベース構成

### データベース情報
- **エンジン**: SQLite 3
- **ファイル**: `/home/ec2-user/cupid/cupid.db` (本番), `cupid.db` (開発)
- **文字コード**: UTF-8
- **特徴**: ファイルベース、サーバープロセス不要
- **マイグレーション管理**: sql-migrate

### マイグレーション
- **ツール**: [sql-migrate](https://github.com/rubenv/sql-migrate)
- **設定ファイル**: `dbconfig.yml`
- **マイグレーションディレクトリ**: `db/migrations/`
- **実行コマンド**: `sql-migrate up -env=development` (ローカル), `sql-migrate up -env=production` (EC2)

---

## テーブル定義

### 1. usersテーブル

#### テーブル作成SQL
```sql
CREATE TABLE users (
  line_user_id TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  birthday TEXT NOT NULL DEFAULT '',
  registration_step INTEGER NOT NULL DEFAULT 0,
  temp_crush_name TEXT,
  registered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 名前と誕生日の組み合わせで検索するためのインデックス
CREATE INDEX idx_users_name_birthday ON users(name, birthday);
```

**注意**: `updated_at` の自動更新は、SQLite + SQLBoilerの組み合わせではトリガーが必要。SQLiteの`DEFAULT CURRENT_TIMESTAMP`はINSERT時のみ適用され、UPDATE時は適用されないため。

#### フィールド説明
| カラム名 | 型 | 制約 | 説明 |
|---------|-----|------|------|
| line_user_id | TEXT | PRIMARY KEY | LINE User ID（Uで始まる一意な文字列） |
| name | TEXT | NOT NULL | ユーザーの名前（**ひらがな**で登録） |
| birthday | TEXT | NOT NULL | 生年月日（YYYY-MM-DD形式の文字列） |
| registration_step | INTEGER | NOT NULL | 登録ステップ（後述、0〜3の整数） |
| temp_crush_name | TEXT | NULL | 好きな人の名前を一時保存（ステップ3時に使用） |
| registered_at | TEXT | NOT NULL | 登録日時（ISO8601形式） |
| updated_at | TEXT | NOT NULL | 更新日時（ISO8601形式、SQLBoilerが自動更新） |

#### registration_stepの値と状態遷移

`registration_step` はINTEGER型で、以下の値を取る：

| 値 | 状態（Go定数） | 説明 | 次の入力 |
|----|---------------|------|---------|
| 0 | `StepAwaitingName` | 名前入力待ち | ユーザー自身の名前 |
| 1 | `StepAwaitingBirthday` | 生年月日入力待ち | ユーザー自身の生年月日 |
| 2 | `StepCompleted` | 登録完了（好きな人の名前入力待ち） | 好きな人の名前 → `temp_crush_name`に保存 |
| 3 | `StepAwaitingCrushBirthday` | 好きな人の生年月日入力待ち | 好きな人の生年月日 → `likes`テーブルに登録 |

**Go定数定義例**:
```go
const (
    StepAwaitingName = 0
    StepAwaitingBirthday = 1
    StepCompleted = 2
    StepAwaitingCrushBirthday = 3
)
```

**状態遷移図**:
```
[新規] → 0 (awaiting_name) → 1 (awaiting_birthday) → 2 (completed) → 3 (awaiting_crush_birthday) → 2 (completed)
```

**注意**: `completed`状態は2つの意味を持つ、ね：
1. 自分の情報登録完了後、好きな人の名前入力待ち
2. 好きな人の登録完了後（再度`completed`に戻る）

#### 使用パターン
```sql
-- ユーザー情報の取得
SELECT * FROM users WHERE line_user_id = 'U1234567890abcdef';

-- ユーザー情報の更新（名前はひらがなで登録）
UPDATE users
SET name = 'しのざわひろ', registration_step = 1
WHERE line_user_id = 'U1234567890abcdef';

-- 名前と誕生日で検索（マッチング用）
SELECT line_user_id FROM users
WHERE name = 'しのざわひろ' AND birthday = '2009-12-21';
```

---

### 2. likesテーブル

#### テーブル作成SQL
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

-- マッチング検索用のインデックス
CREATE INDEX idx_likes_to_name_birthday ON likes(to_name, to_birthday);
```

#### フィールド説明
| カラム名 | 型 | 制約 | 説明 |
|---------|-----|------|------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT | 自動採番のID |
| from_user_id | TEXT | FOREIGN KEY, UNIQUE | 好きな人を登録したユーザーのLINE User ID（1ユーザー1レコードのみ） |
| to_name | TEXT | NOT NULL | 好きな人の名前（**ひらがな**） |
| to_birthday | TEXT | NOT NULL | 好きな人の生年月日（YYYY-MM-DD形式） |
| matched | INTEGER | NOT NULL | マッチング済みフラグ（0=未、1=済） |
| created_at | TEXT | NOT NULL | 登録日時（ISO8601形式） |

#### UNIQUE制約の挙動

`from_user_id`に`UNIQUE`制約があるため、1ユーザーは1レコードしか持てない、ね。

**再登録時の動作**:
- 同じユーザーが別の人を好きになって再登録
- → `UNIQUE constraint failed`エラー
- → アプリケーション側で`INSERT OR REPLACE`または`UPDATE`で対応が必要

**実装での対応**:
```sql
-- 既存レコードがあれば削除してから挿入
DELETE FROM likes WHERE from_user_id = ?;
INSERT INTO likes (from_user_id, to_name, to_birthday) VALUES (?, ?, ?);

-- または、INSERT OR REPLACEを使用
INSERT OR REPLACE INTO likes (from_user_id, to_name, to_birthday, matched)
VALUES (?, ?, ?, 0);
```

#### 制約
- **UNIQUE(from_user_id)**: 1人のユーザーは1人しか登録できない
- **FOREIGN KEY**: from_user_idはusersテーブルのline_user_idを参照

#### SQLiteのBoolean型について
```
SQLiteにはBoolean型が存在しない
→ INTEGER型で 0=FALSE, 1=TRUE を表現
```

#### 使用パターン
```sql
-- 好きな人の登録（名前はひらがなで）
INSERT INTO likes (from_user_id, to_name, to_birthday)
VALUES ('U1234567890abcdef', 'つきむらてまり', '2010-04-04');

-- マッチング検索（重要）
SELECT l.from_user_id, u.name, u.birthday
FROM likes l
JOIN users u ON l.from_user_id = u.line_user_id
WHERE l.to_name = 'しのざわひろ'
  AND l.to_birthday = '2009-12-21'
  AND l.matched = 0;

-- マッチング済みに更新
UPDATE likes
SET matched = 1
WHERE from_user_id = 'U1234567890abcdef';
```

---

## SQLiteの特徴と注意点

### 1. データ型の柔軟性
```
SQLiteは動的型付け
→ TEXT, INTEGER, REAL, BLOB, NULL

日付型が存在しない:
- DATE → TEXT型で 'YYYY-MM-DD' 形式保存
- TIMESTAMP → TEXT型で ISO8601形式保存
```

### 2. 外部キー制約
```sql
-- デフォルトで無効
-- 有効化が必要（接続時に毎回実行）
PRAGMA foreign_keys = ON;
```

Goのコード例：
```go
db, _ := sql.Open("sqlite3", "cupid.db?_foreign_keys=on")
```

### 3. AUTO_INCREMENT
```sql
-- PostgreSQL/MySQL: SERIAL
-- SQLite: INTEGER PRIMARY KEY AUTOINCREMENT
```

### 4. トリガー（SQLite特有の要件）
```
updated_at の自動更新: SQLiteではトリガーが必要
→ DEFAULT CURRENT_TIMESTAMP はINSERT時のみ適用
→ UPDATE時は適用されない
→ SQLBoilerのAuto Timestamp機能だけでは不十分
→ トリガーでUPDATE時の自動更新を実現

FOR EACH ROW は必須
BEGIN ... END で囲む
```

---

## マッチング検索ロジック

### シナリオ
1. ユーザーA（篠澤広, 2009-12-21, line_user_id: `UA111...`）がユーザーB（月村手毬, 2010-04-04）を好きと登録
2. システムは以下をチェック：
   - likesテーブルに「篠澤広 & 2009-12-21」を好きとして登録した人（ユーザーB）がいるか？
   - その人のLINE User IDは何か？
   - その人はまだマッチングしていないか？（matched = 0）

### SQLクエリ
```sql
-- ユーザーAの情報を取得
SELECT line_user_id, name, birthday
FROM users
WHERE line_user_id = 'UA111...';
-- 結果: name='しのざわひろ', birthday='2009-12-21'

-- ユーザーAを好きな人を検索
SELECT l.from_user_id, u.name
FROM likes l
JOIN users u ON l.from_user_id = u.line_user_id
WHERE l.to_name = 'しのざわひろ'
  AND l.to_birthday = '2009-12-21'
  AND l.matched = 0;
-- 結果があれば相思相愛
-- 例: from_user_id='UB222...', name='つきむらてまり'
```

### マッチング成立後の処理
```sql
-- トランザクション開始
BEGIN TRANSACTION;

-- ユーザーAのlikesレコードを更新
UPDATE likes
SET matched = 1
WHERE from_user_id = 'UA111...';

-- ユーザーBのlikesレコードを更新
UPDATE likes
SET matched = 1
WHERE from_user_id = 'UB222...';

-- コミット
COMMIT;

-- その後、両方のユーザーにPush Message送信
```

---

## データフロー例

### ユーザー登録フロー
```sql
-- 1. 初回メッセージ（新規ユーザー）
INSERT INTO users (line_user_id, name, registration_step)
VALUES ('U1234567890abcdef', '', 0);

-- 2. 名前送信（ひらがなで登録）
UPDATE users
SET name = 'しのざわひろ', registration_step = 1
WHERE line_user_id = 'U1234567890abcdef';

-- 3. 生年月日送信
UPDATE users
SET birthday = '2009-12-21', registration_step = 2
WHERE line_user_id = 'U1234567890abcdef';
```

### マッチング登録フロー
```sql
-- 1. 好きな人の名前送信（temp_crush_nameに保存、ひらがなで）
UPDATE users
SET temp_crush_name = 'つきむらてまり',
    registration_step = 3
WHERE line_user_id = 'U1234567890abcdef';

-- 2. 好きな人の生年月日送信
BEGIN TRANSACTION;

-- 2-1. likesテーブルに登録（名前はひらがなで）
INSERT INTO likes (from_user_id, to_name, to_birthday)
VALUES ('U1234567890abcdef', 'つきむらてまり', '2010-04-04');

-- 2-2. マッチング検索
SELECT l.from_user_id, u.name
FROM likes l
JOIN users u ON l.from_user_id = u.line_user_id
WHERE l.to_name = 'しのざわひろ'
  AND l.to_birthday = '2009-12-21'
  AND l.matched = 0;

-- 2-3. マッチングがあれば両方を更新
-- (結果がある場合のみ実行)
UPDATE likes SET matched = 1 WHERE from_user_id IN ('UA111...', 'UB222...');

-- 2-4. 登録ステップを戻し、temp_crush_nameをクリア
UPDATE users
SET registration_step = 2,
    temp_crush_name = NULL
WHERE line_user_id = 'U1234567890abcdef';

COMMIT;
```

---

## Goでの実装例

### データベース接続
```go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

// 接続
db, err := sql.Open("sqlite3", "cupid.db?_foreign_keys=on")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// WALモード有効化（同時アクセス性能向上）
db.Exec("PRAGMA journal_mode=WAL")
```

### ユーザー取得
```go
func GetUser(lineUserID string) (*User, error) {
    var user User
    err := db.QueryRow(
        "SELECT line_user_id, name, birthday, registration_step FROM users WHERE line_user_id = ?",
        lineUserID,
    ).Scan(&user.LineUserID, &user.Name, &user.Birthday, &user.RegistrationStep)

    if err == sql.ErrNoRows {
        return nil, nil // ユーザーが存在しない
    }
    return &user, err
}
```

### マッチング検索
```go
func CheckMatch(name, birthday string) (string, error) {
    var matchedUserID string
    err := db.QueryRow(`
        SELECT l.from_user_id
        FROM likes l
        JOIN users u ON l.from_user_id = u.line_user_id
        WHERE l.to_name = ? AND l.to_birthday = ? AND l.matched = 0
    `, name, birthday).Scan(&matchedUserID)

    if err == sql.ErrNoRows {
        return "", nil // マッチングなし
    }
    return matchedUserID, err
}
```

### トランザクション
```go
func RegisterLike(fromUserID, toName, toBirthday string) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback() // エラー時は自動ロールバック

    // likes登録
    _, err = tx.Exec(
        "INSERT INTO likes (from_user_id, to_name, to_birthday) VALUES (?, ?, ?)",
        fromUserID, toName, toBirthday,
    )
    if err != nil {
        return err
    }

    // マッチング検索など...

    return tx.Commit()
}
```

---

## データベース初期化（マイグレーション）

### sql-migrateによるマイグレーション

マイグレーションファイルは `db/migrations/` ディレクトリに配置。

**設定ファイル**: `dbconfig.yml`
```yaml
development:
  dialect: sqlite3
  datasource: cupid.db
  dir: db/migrations
  table: schema_migrations

production:
  dialect: sqlite3
  datasource: /home/ec2-user/cupid/cupid.db
  dir: db/migrations
  table: schema_migrations
```

### マイグレーション実行方法

```bash
# ローカル環境
sql-migrate up -env=development

# 本番環境（EC2）
sql-migrate up -env=production

# ステータス確認
sql-migrate status -env=development

# ロールバック（最後の1つ）
sql-migrate down -env=development

# すべてロールバック
sql-migrate down -limit=0 -env=development
```

**注意**:
- `updated_at` 自動更新トリガーを使用（SQLite + SQLBoilerの組み合わせで必要）。
- 外部キー制約（`PRAGMA foreign_keys = ON`）はGoコード接続時に設定（`?_foreign_keys=on`）。
- `idx_likes_matched` インデックスは削除（YAGNI原則、必要になってから追加）。

---

## 拡張性のための設計

### 将来追加できる機能

#### 1. マッチング履歴
```sql
CREATE TABLE match_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_a_id TEXT NOT NULL,
  user_b_id TEXT NOT NULL,
  matched_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_a_id) REFERENCES users(line_user_id),
  FOREIGN KEY (user_b_id) REFERENCES users(line_user_id)
);

-- 誰と何回マッチングしたか
SELECT u.name, COUNT(*) as match_count
FROM match_history mh
JOIN users u ON mh.user_b_id = u.line_user_id
WHERE mh.user_a_id = 'U1234567890abcdef'
GROUP BY u.name;
```

#### 2. 人気ランキング
```sql
-- 誰が一番人気か
SELECT u.name, COUNT(l.id) as like_count
FROM users u
LEFT JOIN likes l ON u.name = l.to_name AND u.birthday = l.to_birthday
GROUP BY u.name
ORDER BY like_count DESC
LIMIT 10;
```

#### 3. 相思相愛率の統計
```sql
-- 全体のマッチング率
SELECT
  COUNT(*) as total_likes,
  SUM(matched) as matched_likes,
  ROUND(100.0 * SUM(matched) / COUNT(*), 2) as match_rate
FROM likes;
```

---

## MySQLへの移行

### 将来的にユーザーが増えたら

#### データエクスポート
```bash
# SQLiteからダンプ
sqlite3 cupid.db .dump > cupid.sql

# または
sqlite3 cupid.db "SELECT * FROM users;" > users.csv
```

#### MySQL用に変換
```sql
-- データ型変更
TEXT → VARCHAR(255)
INTEGER → INT
matched (0/1) → BOOLEAN

-- AUTO_INCREMENT構文変更
AUTOINCREMENT → AUTO_INCREMENT
```

#### Goコードの変更
```go
// SQLite
db, _ := sql.Open("sqlite3", "cupid.db")

// MySQL
db, _ := sql.Open("mysql", "user:pass@tcp(localhost:3306)/cupid")

// クエリは基本的に同じ（database/sql互換）
```

---

## パフォーマンスチューニング

### WALモード
```sql
PRAGMA journal_mode=WAL;
```
- 読み込みと書き込みが同時実行可能
- パフォーマンス向上
- デフォルトで有効化推奨

### インデックス確認
```sql
-- インデックスの使用状況確認
EXPLAIN QUERY PLAN
SELECT * FROM likes WHERE to_name = '篠澤広' AND to_birthday = '2009-12-21';
```

### VACUUM
```bash
# データベース最適化（定期的に実行）
sqlite3 cupid.db "VACUUM;"
```

---

## 注意事項

### 同姓同名・同じ生年月日の問題
- **現状**: 名前（ひらがな）と生年月日の組み合わせで同一人物を判定
- **問題**: 同じ名前（ひらがな）で誕生日も同じ人がいた場合、後から登録した人の情報で上書きされる
- **対応**: 意図的な仕様（単純に上書きでOK、確認メッセージなし）
- **例**: 「しのざわひろ 2009-12-21」が2人いた場合、後から登録した人が有効
- **ひらがな化の効果**: 表記ゆれ（「篠澤」vs「しのざわ」）を防ぐことで衝突確率を下げる
- **将来の対策案**: ユーザーIDを好きな人として登録する仕組みに変更（スコープ外）

### 好きな人の変更
- **現状**: UNIQUE制約により1人しか登録できない
- **変更方法**: 既存レコードをDELETEしてからINSERT、またはUPDATE
- **実装**: 後で考える（スコープ外）

### データ整合性
- **外部キー制約**: from_user_idはusersテーブルを参照
- **重要**: `PRAGMA foreign_keys = ON` を忘れずに
- **メリット**: 存在しないユーザーをlikesに登録できない

### 同時書き込み
```
SQLiteは書き込み時にファイルロック
→ 同時書き込みは直列化される

今回の用途（同時アクセス1-2人）:
→ 全く問題なし
```

### バックアップ
```bash
# ファイルコピーだけ（簡単）
cp cupid.db cupid.db.backup

# WALモード使用時はWALファイルもコピー
cp cupid.db cupid.db.backup
cp cupid.db-wal cupid.db-wal.backup  # 存在する場合
cp cupid.db-shm cupid.db-shm.backup  # 存在する場合
```

---

## マイグレーションファイル

以下は、データベース初期化に使用する初回マイグレーションファイル、ね。

ファイルパス: `db/migrations/20260117000001-initial_schema.sql`

```sql
-- +migrate Up
-- ユーザーテーブル
-- registration_step: 0=awaiting_name, 1=awaiting_birthday, 2=completed, 3=awaiting_crush_birthday
CREATE TABLE users (
  line_user_id TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  birthday TEXT NOT NULL DEFAULT '',
  registration_step INTEGER NOT NULL DEFAULT 0,
  temp_crush_name TEXT,
  registered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 名前と誕生日の組み合わせで検索するためのインデックス
CREATE INDEX idx_users_name_birthday ON users(name, birthday);

-- 好きな人の登録テーブル
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

-- マッチング検索用のインデックス
CREATE INDEX idx_likes_to_name_birthday ON likes(to_name, to_birthday);

-- updated_at自動更新トリガー（SQLite + SQLBoilerの組み合わせで必要）
CREATE TRIGGER update_users_updated_at
AFTER UPDATE ON users
FOR EACH ROW
BEGIN
  UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE line_user_id = NEW.line_user_id;
END;

-- +migrate Down
DROP TRIGGER IF EXISTS update_users_updated_at;
DROP INDEX IF EXISTS idx_likes_to_name_birthday;
DROP TABLE IF EXISTS likes;
DROP INDEX IF EXISTS idx_users_name_birthday;
DROP TABLE IF EXISTS users;
```

**注意**:
- `updated_at` の自動更新トリガーは使用しない。SQLBoilerのAuto Timestamps機能を使用する。
- 外部キー制約（`PRAGMA foreign_keys = ON`）は、Goコード接続時に設定する（接続文字列に `?_foreign_keys=on`）。

### マイグレーション実行コマンド

```bash
# ローカル環境でマイグレーション実行
sql-migrate up -env=development

# EC2本番環境でマイグレーション実行
sql-migrate up -env=production

# マイグレーション状態確認
sql-migrate status -env=development

# スキーマ確認
sqlite3 cupid.db .schema
```

---

## まとめ

### SQLite選定の利点
- ファイルベース（管理簡単）
- サーバープロセス不要（メモリ節約）
- セットアップ不要（Goドライバで完結）
- 今回の規模には十分すぎる性能
- 将来MySQLに移行も可能

### データベース構成
- usersテーブル: ユーザー情報と登録状態
- likesテーブル: 好きな人情報とマッチング状態
- インデックス: マッチング検索の高速化
- トランザクション: データ整合性の保証
