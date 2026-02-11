# Cupid - 相思相愛マッチングLINE Bot

[![Go Version](https://img.shields.io/badge/Go-1.25.5-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Production-success)](https://cupid-linebot.click/)

**Cupid**は、相思相愛を見つけるためのLINE Botアプリケーション。自分と好きな人の情報を登録すると、相手も自分を登録している場合のみ、両者にマッチング通知が届く。

🔗 **本番環境**: https://cupid-linebot.click/

---

## 📖 概要

### どんなアプリ？

- **匿名性**: 相思相愛でない限り、相手に情報は伝わらない
- **シンプル**: LINE上で名前と誕生日を登録するだけ
- **プライバシー保護**: 一方的な好意は相手に通知されない

### 主な機能

1. **ユーザー登録**: 自分の名前と誕生日をLIFF経由で登録
2. **好きな人登録**: 好きな人の名前と誕生日を登録
3. **自動マッチング**: 相思相愛の場合のみ両者に通知
4. **情報更新**: 登録情報の変更が可能
5. **マッチング解除**: 情報変更時に既存マッチングを解除

---

## 🎯 使い方（ユーザー視点）

### 1. LINE Botを友達追加

QRコードまたは検索でCupid LINE Botを友達追加。

### 2. 自分の情報を登録

トーク画面でメッセージを送ると、登録用のLIFF URLが送られてくる。

```
【入力内容】
- 名前（全角カタカナ、空白なし）
- 誕生日（YYYY-MM-DD形式）
```

### 3. 好きな人を登録

ユーザー登録完了後、好きな人の登録用URLが送られてくる。

```
【入力内容】
- 好きな人の名前（全角カタカナ、空白なし）
- 好きな人の誕生日（YYYY-MM-DD形式）
```

### 4. マッチング通知

相手も自分を登録している場合、両者にマッチング通知が届く。

```
🎉 相思相愛が成立しました！
相手：○○さん
```

### 5. 情報変更

トーク画面からメッセージを送ると、再登録用URLが送られてくる。

**注意**: マッチング中に情報を変更すると、マッチングが解除される。

---

## 🏗️ アーキテクチャ

### システム構成

```
[LINE Platform]
       ↓ Webhook
[Nginx (HTTPS)] ← Let's Encrypt
       ↓
[Cupid Go Server :8080]
       ↓
[SQLite Database]
```

### 技術スタック

| 項目 | 技術 | バージョン |
|-----|------|----------|
| **言語** | Go | 1.25.5 |
| **フレームワーク** | net/http (標準ライブラリ) | - |
| **データベース** | SQLite | 3 |
| **ORM** | SQLBoiler | latest |
| **マイグレーション** | sql-migrate | latest |
| **LINE SDK** | line-bot-sdk-go | v8 |
| **Webサーバー** | Nginx | 1.24.0 |
| **SSL/TLS** | Let's Encrypt (Certbot) | - |
| **インフラ** | AWS EC2 (t4g.micro, ARM64) | Amazon Linux 2023 |
| **ドメイン** | AWS Route 53 | cupid-linebot.click |

### ディレクトリ構成

```
cupid/
├── cmd/
│   └── server/
│       └── main.go              # エントリーポイント
├── internal/
│   ├── handler/                 # HTTPハンドラー
│   │   ├── webhook.go           # LINE Webhook
│   │   ├── registration_api.go  # ユーザー登録API
│   │   └── crush_registration_api.go # 好きな人登録API
│   ├── service/                 # ビジネスロジック
│   │   ├── user_service.go      # ユーザー管理
│   │   └── matching_service.go  # マッチング処理
│   ├── repository/              # データアクセス
│   │   └── user_repo.go
│   ├── model/                   # ドメインモデル
│   │   └── user.go
│   ├── liff/                    # LIFF認証
│   ├── linebot/                 # LINE Bot Client
│   └── database/                # DB接続
├── entities/                    # SQLBoiler自動生成
├── db/
│   ├── migrations/              # SQLマイグレーション
│   └── dbconfig.yml
├── public/
│   ├── liff/                    # ユーザー登録LIFF
│   └── crush/                   # 好きな人登録LIFF
├── e2e/                         # E2Eテスト
└── docs/
    └── plans/                   # 設計ドキュメント
```

---

## 🗄️ データベーススキーマ

### users テーブル

相思相愛マッチングの全情報を管理。

```sql
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

CREATE INDEX idx_users_name_birthday ON users(name, birthday);
CREATE INDEX idx_users_crush ON users(crush_name, crush_birthday);
```

#### フィールド説明

| フィールド | 型 | 説明 |
|-----------|---|------|
| `line_user_id` | TEXT | LINE ユーザーID（主キー） |
| `name` | TEXT | ユーザーの名前（全角カタカナ） |
| `birthday` | TEXT | 誕生日（YYYY-MM-DD） |
| `registration_step` | INTEGER | 登録ステップ（1: ユーザー登録完了, 2: 好きな人登録完了） |
| `crush_name` | TEXT | 好きな人の名前（NULL可） |
| `crush_birthday` | TEXT | 好きな人の誕生日（NULL可） |
| `matched_with_user_id` | TEXT | マッチング相手のLINE ID（NULL=未マッチ） |
| `registered_at` | TEXT | 登録日時 |
| `updated_at` | TEXT | 更新日時（自動更新） |

#### 設計思想

- **YAGNI原則**: 1人のユーザーは1人の好きな人のみ登録可能
- **シンプル設計**: likesテーブルを廃止し、usersテーブルに統合
- **マッチング判定**: `crush_name`/`crush_birthday` と `name`/`birthday` の相互一致で判定

---

## 🔌 API仕様

### LINE Webhook

**Endpoint**: `POST /webhook`

LINE Platformからのイベントを受信。

#### イベントタイプ

1. **follow**: 友達追加時に挨拶メッセージ送信
2. **message**: ユーザーのメッセージに応じて登録URLを案内

### ユーザー登録API

**Endpoint**: `POST /api/register`

LIFF経由でユーザー情報を登録。

#### リクエスト

```json
{
  "name": "タナカタロウ",
  "birthday": "2000-01-15",
  "confirm_unmatch": false
}
```

#### レスポンス（成功）

```json
{
  "status": "ok"
}
```

#### レスポンス（マッチング中）

```json
{
  "error": "matched_user_exists",
  "message": "現在マッチング中です。変更するとマッチングが解除されます。"
}
```

**HTTPステータスコード**: 409 Conflict

#### 認証

- `Authorization: Bearer {LIFF_ID_TOKEN}`
- LINE LIFF ID Tokenを検証してユーザーIDを取得

### 好きな人登録API

**Endpoint**: `POST /api/register-crush`

LIFF経由で好きな人の情報を登録。

#### リクエスト

```json
{
  "crush_name": "ヤマダハナコ",
  "crush_birthday": "2000-05-20",
  "confirm_unmatch": false
}
```

#### レスポンス（マッチング成立）

```json
{
  "status": "ok",
  "matched": true,
  "message": "ヤマダハナコさんとマッチしました！💘"
}
```

#### レスポンス（未マッチ）

```json
{
  "status": "ok",
  "matched": false,
  "message": "登録しました。相手があなたを登録したらマッチングします。"
}
```

#### レスポンス（マッチング中）

```json
{
  "error": "matched_user_exists",
  "message": "現在マッチング中です。変更するとマッチングが解除されます。"
}
```

**HTTPステータスコード**: 409 Conflict

---

## 🔐 マッチングロジック

### マッチング判定

2人のユーザーA, Bが以下の条件を満たす場合にマッチング成立。

```
A.name == B.crush_name
AND A.birthday == B.crush_birthday
AND B.name == A.crush_name
AND B.birthday == A.crush_birthday
AND B.matched_with_user_id IS NULL
```

### マッチング処理フロー

1. **好きな人登録時**: `MatchingService.CheckAndUpdateMatch()` を実行
2. **相互マッチング検索**: `UserRepository.FindMatchingUser()` で相手を検索
3. **マッチング成立時**:
   - 両者の `matched_with_user_id` を更新
   - 両者にLINE Push通知を送信

### マッチング解除

情報変更時にマッチングが解除される。

#### 解除トリガー

- ユーザー情報変更（名前・誕生日）
- 好きな人変更

#### 解除フロー

1. **確認**: LIFF側で「マッチングが解除されます」と確認
2. **ユーザー承認**: `confirm_unmatch=true` で再送信
3. **解除処理**: 両者の `matched_with_user_id` をNULLに
4. **通知送信**: 両者に解除理由を通知

---

## 🛠️ 開発者向け情報

### ローカル開発環境

#### 必要なツール

- Go 1.25.5+
- SQLite 3
- Make

#### セットアップ

```bash
# 依存関係インストール
go mod download

# マイグレーション実行
make migrate-up

# SQLBoiler entity生成
make generate

# テスト実行
make test

# ローカルサーバー起動
go run ./cmd/server
```

#### 環境変数

`.env` ファイルを作成：

```env
# LINE Bot設定
LINE_CHANNEL_SECRET=your_channel_secret
LINE_CHANNEL_ACCESS_TOKEN=your_access_token

# LIFF設定（ユーザー登録用）
USER_LIFF_CHANNEL_ID=your_user_liff_channel_id
USER_LIFF_URL=your_user_liff_url

# LIFF設定（好きな人登録用）
CRUSH_LIFF_CHANNEL_ID=your_crush_liff_channel_id
CRUSH_LIFF_URL=your_crush_liff_url
```

### テスト

```bash
# 全テスト実行
make test

# カバレッジ確認
go test -cover ./...

# 特定パッケージのテスト
go test ./internal/service/...
```

### マイグレーション

```bash
# マイグレーション適用
make migrate-up

# ロールバック
make migrate-down

# マイグレーション状態確認
make migrate-status

# 新規マイグレーション作成
sql-migrate new -config=db/dbconfig.yml migration_name
```

### ビルド＆デプロイ

```bash
# ローカルビルド
go build -o cupid ./cmd/server

# EC2へデプロイ（自動）
make deploy
```

**`make deploy` の処理内容**:
1. EC2にSSH接続
2. `git pull` で最新コード取得
3. `sql-migrate up` でマイグレーション適用
4. `go build` でビルド
5. `sudo systemctl restart cupid` でサービス再起動

---

## 🚀 本番環境

### インフラ情報

| 項目 | 値 |
|-----|---|
| **ドメイン** | cupid-linebot.click |
| **サーバー** | AWS EC2 t4g.micro (ARM64) |
| **OS** | Amazon Linux 2023 |
| **リージョン** | ap-northeast-1 (東京) |
| **IPアドレス** | 13.115.86.124 (Elastic IP) |
| **SSL証明書** | Let's Encrypt（自動更新設定済み） |

### サービス管理

```bash
# SSH接続
ssh cupid-bot

# サービス状態確認
sudo systemctl status cupid

# サービス再起動
sudo systemctl restart cupid

# ログ確認（リアルタイム）
sudo journalctl -u cupid -f

# ログ確認（最新100行）
sudo journalctl -u cupid -n 100

# データベース確認
sqlite3 ~/cupid/cupid.db "SELECT * FROM users;"
```

### デプロイフロー

1. **ローカルで開発＆テスト**
   ```bash
   make test
   git commit -am "feat: add new feature"
   ```

2. **GitHubにpush**
   ```bash
   git push origin main
   ```

3. **本番デプロイ**
   ```bash
   make deploy
   ```

### 監視

#### ヘルスチェック

```bash
curl https://cupid-linebot.click/
# => "Cupid LINE Bot is running"
```

#### ログモニタリング

```bash
# エラーログのみ表示
sudo journalctl -u cupid -p err -n 50

# 特定文字列で検索
sudo journalctl -u cupid | grep "ERROR"
```

---

## 📊 運用コスト

| 項目 | 月額 |
|-----|------|
| EC2 t4g.micro | $3.67 |
| Elastic IP | $0.00（稼働中は無料） |
| Route 53 ホストゾーン | $0.50 |
| データ転送 | $0.00（少量のため） |
| **合計** | **約$4.17/月（約540円）** |

※ ドメイン取得費用: $3/年（.clickドメイン）

---

## 🎓 設計ドキュメント

詳細な設計ドキュメントは `docs/plans/` に格納。

- **[2026-02-10-likes-migration-design.md](docs/plans/2026-02-10-likes-migration-design.md)**: likesテーブル削除とusersテーブル統合の設計
- **[2026-02-10-likes-migration-implementation.md](docs/plans/2026-02-10-likes-migration-implementation.md)**: 実装計画（21タスク）
- **[2026-02-10-register-from-liff-refactoring-design.md](docs/plans/2026-02-10-register-from-liff-refactoring-design.md)**: RegisterFromLIFFリファクタリング設計

---

## 🔒 セキュリティ

### 認証・認可

- **LIFF ID Token検証**: LINE Platform発行のID Tokenを検証してユーザーIDを取得
- **Webhook署名検証**: LINE Platform署名を検証して正当性を確認

### プライバシー保護

- **匿名性**: 相思相愛でない限り、相手に情報は通知されない
- **最小限のデータ保存**: 名前・誕生日のみ保存
- **データ暗号化**: HTTPS通信で全データ暗号化

---

## 📜 ライセンス

MIT License

---

## 🤝 貢献

Issue・Pull Requestを歓迎します。

---

## 📧 サポート

- **Issue**: [GitHub Issues](https://github.com/morinonusi421/cupid/issues)
- **ドキュメント**: `docs/` ディレクトリ

---

**Cupid** - 相思相愛を見つけるLINE Bot 💘
