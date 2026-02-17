# Cupid - 相思相愛マッチングLINE Bot

[![Go Version](https://img.shields.io/badge/Go-1.25.5-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Production-success)](https://cupid-linebot.click/)

**Cupid**は、相思相愛を見つけるためのLINE Botアプリケーション。自分と好きな人の情報を登録すると、相手も自分を登録している場合のみ、両者にマッチング通知が届く。

🔗 **本番LineBot**: https://line.me/R/ti/p/@763jbwrh

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
- 誕生日
```

### 3. 好きな人を登録

ユーザー登録完了後、好きな人の登録用URLが送られてくる。

```
【入力内容】
- 好きな人の名前（全角カタカナ、空白なし）
- 好きな人の誕生日
```

### 4. マッチング通知

相手も自分を登録している場合、両者にマッチング通知が届く。

### 5. 情報変更

トーク画面下部のリッチメニューから、情報の再登録ができる。

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

| 項目 | 技術 |
|-----|------|
| **言語** | Go |
| **フレームワーク** | net/http (標準ライブラリ) |
| **データベース** | SQLite |
| **ORM** | SQLBoiler |
| **LINE SDK** | line-bot-sdk-go |
| **Mock生成** | Mockery |
| **Webサーバー** | Nginx |
| **SSL/TLS** | Let's Encrypt (Certbot) |
| **インフラ** | AWS EC2 (t4g.micro, ARM64) |
| **ドメイン** | AWS Route 53 |

### ディレクトリ構成

```
cupid/
├── cmd/
│   └── server/
│       └── main.go              # エントリーポイント
├── internal/
│   ├── handler/                 # HTTPハンドラー
│   │   ├── webhook.go           # LINE Webhook
│   │   ├── user_registration_api.go  # ユーザー登録API
│   │   └── crush_registration_api.go # 好きな人登録API
│   ├── service/                 # ビジネスロジック
│   │   ├── user_service.go      # ユーザー管理
│   │   ├── matching_service.go  # マッチング処理
│   │   ├── notification_service.go # 通知処理
│   │   └── mocks/               # Mockery自動生成
│   ├── repository/              # データアクセス層
│   │   ├── user_repo.go
│   │   └── mocks/               # Mockery自動生成
│   ├── model/                   # ドメインモデル
│   │   └── user.go
│   ├── message/                 # メッセージ定数
│   ├── middleware/              # HTTPミドルウェア
│   ├── liff/                    # LIFF認証
│   │   └── mocks/               # Mockery自動生成
│   └── linebot/                 # LINE Bot Client
├── pkg/                         # 共通パッケージ
│   ├── database/                # DB接続
│   ├── httputil/                # HTTP応答ヘルパー
│   └── testutil/                # テストユーティリティ
├── entities/                    # SQLBoiler自動生成
├── db/
│   └── schema.sql               # データベーススキーマ
├── public/
│   ├── liff/                    # ユーザー登録LIFF
│   └── crush/                   # 好きな人登録LIFF
└── e2e/                         # E2Eテスト
```

---

## 🗄️ データベーススキーマ

### スキーマ管理

- **ファイル**: `db/schema.sql`
- **初期化**: アプリケーション起動時に自動作成

### users テーブル

相思相愛マッチングの全情報を管理。

#### フィールド説明

| フィールド | 型 | 説明 |
|--------|---|------|
| `line_user_id` | TEXT | LINE ユーザーID（主キー） |
| `name` | TEXT | ユーザーの名前（全角カタカナ） |
| `birthday` | TEXT | 誕生日（YYYY-MM-DD） |
| `crush_name` | TEXT | 好きな人の名前（NULL可） |
| `crush_birthday` | TEXT | 好きな人の誕生日（NULL可） |
| `matched_with_user_id` | TEXT | マッチング相手のLINE ID（NULL=未マッチ） |
| `registered_at` | TEXT | 登録日時 |
| `updated_at` | TEXT | 更新日時（自動更新） |

---

## 🔌 エンドポイント

### LINE Webhook

**Endpoint**: `POST /webhook`

LINE Platformからのイベントを受信。

- **follow**: 友達追加時に挨拶メッセージ送信
- **join**: グループ招待時に挨拶メッセージ送信
- **message**: ユーザーのメッセージに応じて登録URLを案内

### 内部API

以下のAPIはLIFF経由でのみ使用される内部APIです。

- `POST /api/register-user` - ユーザー情報登録
- `POST /api/register-crush` - 好きな人情報登録

詳細な仕様はコードを参照してください。

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

## ✅ バリデーションと制約

### 登録時のバリデーションルール

#### 1. 自己登録の防止

ユーザーは自分自身を好きな人として登録できません。

**防止ケース:**
- 自分の名前・誕生日と同じ情報を好きな人として登録 → エラー
- 好きな人を登録後、自分の情報を好きな人と同じに変更 → エラー

**エラーメッセージ**: 「自分自身は登録できません」

#### 2. 重複ユーザーの防止

既に登録されているユーザーと同じ名前・誕生日での登録はできません。

**防止ケース:**
- ユーザーA: タカハシヒカル（2000-04-21）登録済み
- ユーザーB: 同じ名前・誕生日で登録しようとする → エラー

**エラーメッセージ**: 「同じ名前・誕生日のユーザーが既に登録されています。」

**例外**: 自分自身の情報更新は可能（同じ名前・誕生日のまま更新）

#### 3. 名前のバリデーション

**ルール:**
- 全角カタカナのみ（ひらがな・漢字・英数字不可）
- 2〜20文字
- スペース不可（姓名を続けて入力）

### マッチング中の情報変更

#### 変更時の挙動

マッチング中でも自分や好きな人の情報変更は可能ですが、以下の処理が行われます。

1. **確認**: LIFF画面で確認ダイアログを表示
2. **解除**: 両者の `matched_with_user_id` を NULL にリセット
3. **通知**: 両者にマッチング解除を通知

---

## 🛠️ 開発者向け情報

### システムアーキテクチャ

#### レイヤー構成

このアプリケーションは、標準的な3層アーキテクチャを採用しています。

```
[Handler Layer (HTTP)]
         ↓
[Service Layer (Business Logic)]
         ↓
[Repository Layer (Data Access)]
         ↓
[Database (SQLite)]
```

#### 各レイヤーの責務

| レイヤー | パッケージ | 責務 |
|---------|----------|------|
| **Handler** | `internal/handler/` | HTTPリクエスト/レスポンス処理、バリデーション |
| **Service** | `internal/service/` | ビジネスロジック、トランザクション制御 |
| **Repository** | `internal/repository/` | データベースCRUD操作 |
| **Model** | `internal/model/` | ドメインモデル定義 |

#### 依存関係の方向

```
Handler → Service → Repository → Database
   ↓         ↓           ↓
 Model    Model       Model
```

- 各レイヤーは下位レイヤーのみに依存
- 上位レイヤーは下位レイヤーのインターフェースを通じて依存（疎結合）
- テスト時はモックを注入して単体テスト可能

#### コード生成ツール

| ツール | 生成対象 | コマンド | 設定ファイル |
|--------|---------|---------|------------|
| **SQLBoiler** | `entities/` | `make generate` | `sqlboiler.toml` |
| **Mockery** | `internal/*/mocks/` | `make mocks` | `.mockery.yaml` |

**SQLBoiler**:
- `db/schema.sql`からGo構造体とCRUD操作を自動生成

**Mockery**:
- インターフェースから自動でモック生成

### ローカル開発環境

#### セットアップ

```bash
# リポジトリクローン
git clone https://github.com/morinonusi421/cupid.git
cd cupid

# 依存関係インストール
go mod download

# 環境変数設定
cp .env.example .env
# .envファイルを編集して実際の値を設定

# SQLBoiler entity生成
make generate

# Mockery mock生成
make mocks

# ローカルサーバー起動
go run ./cmd/server
```

**注意**: データベースはアプリケーション起動時に`db/schema.sql`から自動作成されます。

### テスト

```bash
# 全テスト実行
make test
```

**注意**: `entities/`配下のSQLBoiler自動生成テストは実行しません。

---

## 🏗️ インフラセットアップ

本番環境を構築するために実施した設定です。

### LINE Platform

#### Messaging API
- **Channel作成**: LINE Developers ConsoleでMessaging APIチャンネル作成
- **認証情報取得**: Channel Secret、Channel Access Token
- **Webhook設定**: `https://cupid-linebot.click/webhook` を登録

#### LIFF（LINE Front-end Framework）
- **ユーザー登録用**: Endpoint `https://cupid-linebot.click/liff/register.html`
- **好きな人登録用**: Endpoint `https://cupid-linebot.click/crush/register.html`
- 各LIFFアプリでLIFF IDを取得

### AWS

#### ドメイン
- **Route 53**: `cupid-linebot.click` を取得
- **Aレコード**: ドメイン → Elastic IPを設定

#### EC2
- **インスタンス**: t4g.micro（ARM64、無料枠対象）
- **OS**: Amazon Linux 2023
- **セキュリティグループ**: SSH/HTTP/HTTPSを許可
- **Elastic IP**: 固定IPを割り当て

### サーバー

#### Webサーバー
- **Nginx**: リバースプロキシとして設定
- **設定ファイル**: リポジトリの`nginx/cupid.conf`をシンボリックリンク
- **SSL証明書**: Let's Encryptで取得、自動更新設定

#### サービス化
- **systemd**: リポジトリの`systemd/cupid.service`をシンボリックリンク
- **自動起動**: サーバー再起動時に自動でGoアプリ起動

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

**注意**: 以下のコマンドは、管理者（開発者）がローカル環境で`~/.ssh/config`にSSH設定を行っている前提です。このリポジトリをクローンしただけでは、本番EC2にはアクセスできません（セキュリティのため意図的）。

```bash
# SSH接続（管理者のみ）
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

   **`make deploy`の処理内容**:
   - EC2にSSH接続（`ssh cupid-bot`コマンドをセットアップしている前提）
   - `git pull`で最新コード取得
   - `go build`でビルド
   - `sudo systemctl restart cupid`でサービス再起動

   **注意**: データベーススキーマ変更時は、EC2で手動でDBファイルを削除してから再起動してください。

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

## 🔒 セキュリティ

### 認証・認可

- **LIFF ID Token検証**: LINE Platform発行のID Tokenを検証してユーザーIDを取得
- **Webhook署名検証**: LINE Platform署名を検証して正当性を確認

