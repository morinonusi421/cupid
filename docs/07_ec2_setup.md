# EC2セットアップ手順

このドキュメントでは、EC2インスタンスの作成から、Nginx + Let's Encrypt + Go + SQLiteの完全なセットアップまでを説明する、よ。

---

## 前提条件

- AWSアカウントを持っている
- AWS CLIがインストール済み（またはAWS Management Consoleを使用）
- ドメインを取得済み（Route 53で.clickドメインを推奨）
- LINE Bot の Channel Secret と Channel Access Token を取得済み（`06_linebot_setup.md`参照）

---

## 1. Route 53でドメインを取得

### 1.1 .clickドメインの購入

1. AWS Management Consoleにログイン
2. **Route 53**サービスを開く
3. 左メニューから「ドメインの登録」をクリック
4. ドメイン名を入力（例：`cupid.click`）
5. 「チェック」をクリックして利用可能か確認
6. カートに追加して購入（年間$3）

### 1.2 Hosted Zoneの確認

ドメイン購入後、自動的にHosted Zoneが作成される、よ。
- 左メニュー「ホストゾーン」で確認
- NSレコードとSOAレコードが自動作成されている

---

## 2. EC2インスタンスの作成

### 2.1 インスタンス作成

1. **EC2**サービスを開く
2. 「インスタンスを起動」をクリック
3. 以下の設定を行う：

#### 名前とタグ
- **名前**: `cupid-bot`

#### AMI（Amazon Machine Image）
- **AMI**: `Amazon Linux 2023 AMI`
- **アーキテクチャ**: `64ビット (Arm)`

#### インスタンスタイプ
- **インスタンスタイプ**: `t4g.nano`
  - vCPU: 2
  - メモリ: 0.5 GiB
  - ネットワークパフォーマンス: 最大5 Gbps

#### キーペア
- 「新しいキーペアの作成」をクリック
- **キーペア名**: `cupid-bot-key`
- **キーペアタイプ**: `RSA`
- **プライベートキーファイル形式**: `.pem`（macOS/Linux）または`.ppk`（Windows/PuTTY）
- ダウンロードして安全な場所に保存

#### ネットワーク設定
- **VPC**: デフォルトVPC
- **サブネット**: デフォルトサブネット（パブリックサブネット）
- **パブリックIPの自動割り当て**: 有効化

#### セキュリティグループ
「セキュリティグループを作成」を選択：
- **セキュリティグループ名**: `cupid-bot-sg`
- **説明**: `Security group for Cupid LINE Bot`

#### ストレージ
- **ボリュームタイプ**: `gp3`
- **サイズ**: `10 GiB`
- **IOPS**: `3000`（デフォルト）
- **スループット**: `125 MB/s`（デフォルト）

4. 「インスタンスを起動」をクリック

### 2.2 Elastic IPの割り当て

固定IPアドレスが必要なので、Elastic IPを割り当てる、よ。

1. EC2左メニューから「Elastic IP」を選択
2. 「Elastic IPアドレスを割り当てる」をクリック
3. 「割り当て」をクリック
4. 割り当てられたIPを選択して「アクション」→「Elastic IPアドレスの関連付け」
5. **インスタンス**: `cupid-bot`を選択
6. 「関連付ける」をクリック

⋯⋯このElastic IPをメモしておく、ね。

---

## 3. セキュリティグループの設定

### 3.1 インバウンドルールの追加

作成したセキュリティグループ`cupid-bot-sg`を編集：

1. EC2左メニューから「セキュリティグループ」を選択
2. `cupid-bot-sg`を選択
3. 「インバウンドのルール」タブで「インバウンドのルールを編集」をクリック
4. 以下のルールを追加：

| タイプ | プロトコル | ポート範囲 | ソース | 説明 |
|--------|-----------|----------|--------|------|
| SSH | TCP | 22 | マイIP | SSH接続用（自宅IPのみ） |
| HTTP | TCP | 80 | 0.0.0.0/0 | Let's Encryptドメイン認証用 |
| HTTPS | TCP | 443 | 0.0.0.0/0 | LINE Webhook + ユーザーアクセス |

5. 「ルールを保存」をクリック

**重要**: SSH（ポート22）は必ず「マイIP」に制限すること。「0.0.0.0/0」にすると全世界から攻撃を受ける、よ。

### 3.2 アウトバウンドルール

デフォルトのまま（全て許可）でOK、ね。

---

## 4. Route 53でのDNS設定

### 4.1 Aレコードの追加

ドメインをEC2のElastic IPに向ける、よ。

1. **Route 53**サービスを開く
2. 左メニュー「ホストゾーン」をクリック
3. 購入したドメイン（例：`cupid.click`）を選択
4. 「レコードを作成」をクリック
5. 以下を入力：
   - **レコード名**: 空欄（ルートドメイン）
   - **レコードタイプ**: `A`
   - **値**: EC2のElastic IP（例：`3.112.23.0`）
   - **TTL**: `300`秒
   - **ルーティングポリシー**: `シンプルルーティング`
6. 「レコードを作成」をクリック

### 4.2 DNS伝搬の確認

```bash
# ローカルマシンから実行
nslookup cupid.click
# または
dig cupid.click
```

Elastic IPが返ってくればOK、ね。DNS伝搬には最大48時間かかることがあるけど、通常は数分で完了する、よ。

---

## 5. EC2への接続

### 5.1 キーペアの権限設定

```bash
# ダウンロードしたキーペアの権限を変更
chmod 400 ~/Downloads/cupid-bot-key.pem
```

### 5.2 SSH接続

```bash
ssh -i ~/Downloads/cupid-bot-key.pem ec2-user@<Elastic IP>
# 例：ssh -i ~/Downloads/cupid-bot-key.pem ec2-user@3.112.23.0

# またはドメイン名で接続（DNS伝搬後）
ssh -i ~/Downloads/cupid-bot-key.pem ec2-user@cupid.click
```

初回接続時は「yes」を入力して接続を続ける、ね。

---

## 6. Amazon Linux 2023の初期設定

### 6.1 パッケージの更新

```bash
# システムパッケージを最新化
sudo dnf update -y
```

### 6.2 タイムゾーンの設定

```bash
# タイムゾーンを日本時間に設定
sudo timedatectl set-timezone Asia/Tokyo

# 確認
timedatectl
```

### 6.3 必要なツールのインストール

```bash
# 基本ツールのインストール
sudo dnf install -y git vim wget curl
```

---

## 7. Goのインストール

### 7.1 Goのインストール（ARM64版）

```bash
# Go 1.21の最新版をダウンロード
cd /tmp
wget https://go.dev/dl/go1.21.6.linux-arm64.tar.gz

# 既存のGoを削除（初回はスキップ）
sudo rm -rf /usr/local/go

# Goを展開
sudo tar -C /usr/local -xzf go1.21.6.linux-arm64.tar.gz

# クリーンアップ
rm go1.21.6.linux-arm64.tar.gz
```

### 7.2 環境変数の設定

```bash
# .bashrcに追加
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc

# 設定を反映
source ~/.bashrc
```

### 7.3 Goのバージョン確認

```bash
go version
# 出力例: go version go1.21.6 linux/arm64
```

---

## 8. Nginxのインストールと設定

### 8.1 Nginxのインストール

```bash
# Nginxをインストール
sudo dnf install -y nginx

# Nginxを起動
sudo systemctl start nginx

# 自動起動を有効化
sudo systemctl enable nginx

# 起動確認
sudo systemctl status nginx
```

### 8.2 初期動作確認

ブラウザで`http://<Elastic IP>`にアクセスして、Nginxのデフォルトページが表示されることを確認、ね。

### 8.3 Nginx設定ファイルの作成（一旦HTTPのみ）

Let's Encryptの証明書取得前に、HTTP用の設定を作る、よ。

```bash
# 設定ファイルを作成
sudo vim /etc/nginx/conf.d/cupid.conf
```

以下を記述：

```nginx
server {
    listen 80;
    server_name cupid.click;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

設定をテスト：

```bash
# 設定ファイルの文法チェック
sudo nginx -t

# Nginxを再起動
sudo systemctl restart nginx
```

---

## 9. Let's Encryptの証明書取得

### 9.1 Certbotのインストール

```bash
# EPELリポジトリを有効化（Amazon Linux 2023では不要な場合もある）
sudo dnf install -y epel-release

# Certbotとプラグインをインストール
sudo dnf install -y certbot python3-certbot-nginx
```

### 9.2 SSL証明書の取得

```bash
# Certbotを実行（Nginxプラグイン使用）
sudo certbot --nginx -d cupid.click
```

対話式プロンプトが表示される：
1. **メールアドレス**: 証明書の期限通知用（自分のメールアドレスを入力）
2. **利用規約**: `A`（Agree）を入力
3. **EFFからのメール**: `N`（No）でOK
4. **HTTPSリダイレクト**: `2`（Redirect - make all requests redirect to secure HTTPS access）を選択

証明書が正常に取得されると、Nginx設定ファイルが自動的に更新される、よ。

### 9.3 証明書の自動更新設定

Certbotは自動更新のためのcronジョブを自動的に設定するけど、念のため確認：

```bash
# 自動更新のテスト（実際には更新しない）
sudo certbot renew --dry-run
```

エラーが出なければOK、ね。

### 9.4 更新後のNginx設定確認

```bash
# 更新された設定を確認
sudo cat /etc/nginx/conf.d/cupid.conf
```

以下のような内容に自動更新されている：

```nginx
server {
    server_name cupid.click;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    listen 443 ssl; # managed by Certbot
    ssl_certificate /etc/letsencrypt/live/cupid.click/fullchain.pem; # managed by Certbot
    ssl_certificate_key /etc/letsencrypt/live/cupid.click/privkey.pem; # managed by Certbot
    include /etc/letsencrypt/options-ssl-nginx.conf; # managed by Certbot
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem; # managed by Certbot
}

server {
    if ($host = cupid.click) {
        return 301 https://$host$request_uri;
    } # managed by Certbot

    listen 80;
    server_name cupid.click;
    return 404; # managed by Certbot
}
```

---

## 10. SQLiteデータベースの初期化

### 10.1 プロジェクトディレクトリの作成

```bash
# ホームディレクトリにプロジェクトフォルダを作成
mkdir -p ~/cupid
cd ~/cupid
```

### 10.2 SQLiteのインストール確認

Amazon Linux 2023には標準でSQLite3が入っている：

```bash
sqlite3 --version
# 出力例: 3.40.1 2022-12-28 14:03:47
```

### 10.3 スキーマファイルの作成

`03_database_design.md`のスキーマを使って、初期化スクリプトを作成：

```bash
vim ~/cupid/schema.sql
```

以下を貼り付け：

```sql
-- ユーザーテーブル
CREATE TABLE users (
  line_user_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  birthday TEXT NOT NULL,
  registration_step TEXT NOT NULL DEFAULT 'awaiting_name',
  temp_crush_name TEXT,
  registered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_name_birthday ON users(name, birthday);

CREATE TRIGGER update_users_updated_at
AFTER UPDATE ON users
FOR EACH ROW
BEGIN
  UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE line_user_id = NEW.line_user_id;
END;

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

CREATE INDEX idx_likes_to_name_birthday ON likes(to_name, to_birthday);
CREATE INDEX idx_likes_matched ON likes(matched);

-- 外部キー制約を有効化
PRAGMA foreign_keys = ON;
```

### 10.4 データベースの初期化

```bash
# SQLiteデータベースを作成してスキーマを適用
sqlite3 ~/cupid/cupid.db < ~/cupid/schema.sql

# 確認
sqlite3 ~/cupid/cupid.db "SELECT name FROM sqlite_master WHERE type='table';"
# 出力: users, likes
```

### 10.5 ファイル権限の設定

```bash
# データベースファイルの権限を設定
chmod 644 ~/cupid/cupid.db
```

---

## 11. Goアプリケーションのデプロイ

### 11.1 Goプロジェクトの初期化

```bash
cd ~/cupid

# Go modulesを初期化
go mod init cupid
```

### 11.2 必要な依存関係のインストール

```bash
# LINE Bot SDK v8
go get github.com/line/line-bot-sdk-go/v8/linebot

# SQLiteドライバ
go get github.com/mattn/go-sqlite3
```

### 11.3 main.goの作成

```bash
vim ~/cupid/main.go
```

以下のコードを貼り付け（`04_api_specification.md`の完全な実装を参照）：

```go
package main

import (
    "database/sql"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"

    "github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
    "github.com/line/line-bot-sdk-go/v8/linebot/webhook"
    _ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {
    // データベース接続
    var err error
    dbPath := os.Getenv("DATABASE_PATH")
    if dbPath == "" {
        dbPath = "./cupid.db"
    }

    db, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // WALモードを有効化
    db.Exec("PRAGMA journal_mode=WAL")

    // HTTPハンドラー設定
    http.HandleFunc("/webhook", webhookHandler)
    http.HandleFunc("/health", healthHandler)

    // サーバー起動
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    // リクエストボディを読み取り
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }

    // 署名検証
    signature := r.Header.Get("X-Line-Signature")
    channelSecret := os.Getenv("LINE_CHANNEL_SECRET")

    if !webhook.ValidateSignature(channelSecret, signature, body) {
        http.Error(w, "Invalid signature", http.StatusBadRequest)
        return
    }

    // Webhookリクエストをパース
    request, err := webhook.ParseRequest(channelSecret, r)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }

    // イベント処理
    for _, event := range request.Events {
        if event.GetType() != webhook.EventTypeMessage {
            continue
        }

        messageEvent, ok := event.(*webhook.MessageEvent)
        if !ok {
            continue
        }

        if messageEvent.Message.GetType() != webhook.MessageTypeText {
            continue
        }

        textMessage := messageEvent.Message.(*webhook.TextMessageContent)
        handleMessage(messageEvent.Source.UserId, textMessage.Text, messageEvent.ReplyToken)
    }

    w.WriteHeader(http.StatusOK)
}

func handleMessage(userID, text, replyToken string) {
    // ユーザー取得
    var user struct {
        LineUserID       string
        Name             string
        Birthday         string
        RegistrationStep string
        TempCrushName    sql.NullString
    }

    err := db.QueryRow(`
        SELECT line_user_id, name, birthday, registration_step, temp_crush_name
        FROM users WHERE line_user_id = ?
    `, userID).Scan(&user.LineUserID, &user.Name, &user.Birthday, &user.RegistrationStep, &user.TempCrushName)

    if err == sql.ErrNoRows {
        // 新規ユーザー
        db.Exec(`INSERT INTO users (line_user_id) VALUES (?)`, userID)
        replyMessage(replyToken, "はじめまして。あなたの名前を教えて、ね")
        return
    }

    // ステートマシン
    switch user.RegistrationStep {
    case "awaiting_name":
        db.Exec(`UPDATE users SET name = ?, registration_step = ? WHERE line_user_id = ?`,
            text, "awaiting_birthday", userID)
        replyMessage(replyToken, fmt.Sprintf("%s、ね。生年月日は？（例：2009-12-21）", text))

    case "awaiting_birthday":
        db.Exec(`UPDATE users SET birthday = ?, registration_step = ? WHERE line_user_id = ?`,
            text, "completed", userID)
        replyMessage(replyToken, "登録できた、よ。好きな人の名前は？")

    case "completed":
        db.Exec(`UPDATE users SET temp_crush_name = ?, registration_step = ? WHERE line_user_id = ?`,
            text, "awaiting_crush_birthday", userID)
        replyMessage(replyToken, "その人の生年月日は？")

    case "awaiting_crush_birthday":
        crushName := user.TempCrushName.String
        handleCrushRegistration(userID, user.Name, user.Birthday, crushName, text, replyToken)
    }
}

func handleCrushRegistration(userID, userName, userBirthday, crushName, crushBirthday, replyToken string) {
    tx, _ := db.Begin()
    defer tx.Rollback()

    // Like登録
    tx.Exec(`INSERT INTO likes (from_user_id, to_name, to_birthday) VALUES (?, ?, ?)`,
        userID, crushName, crushBirthday)

    // マッチングチェック
    var matchedUserID string
    err := tx.QueryRow(`
        SELECT from_user_id FROM likes
        WHERE to_name = ? AND to_birthday = ? AND matched = 0
    `, userName, userBirthday).Scan(&matchedUserID)

    if err == sql.ErrNoRows {
        // マッチングなし
        tx.Exec(`UPDATE users SET registration_step = ?, temp_crush_name = NULL WHERE line_user_id = ?`,
            "completed", userID)
        tx.Commit()
        replyMessage(replyToken, "登録した、よ。相思相愛なら通知する、ね")
        return
    }

    // マッチング成立
    tx.Exec(`UPDATE likes SET matched = 1 WHERE from_user_id = ?`, userID)
    tx.Exec(`UPDATE likes SET matched = 1 WHERE from_user_id = ?`, matchedUserID)
    tx.Exec(`UPDATE users SET registration_step = ?, temp_crush_name = NULL WHERE line_user_id = ?`,
        "completed", userID)
    tx.Commit()

    // 両方に通知
    replyMessage(replyToken, "相思相愛、みたい。おめでとう。")
    pushMessage(matchedUserID, "相思相愛、みたい。おめでとう。")
}

func replyMessage(replyToken, text string) {
    bot, _ := messaging_api.NewMessagingApiAPI(os.Getenv("LINE_CHANNEL_TOKEN"))
    bot.ReplyMessage(&messaging_api.ReplyMessageRequest{
        ReplyToken: replyToken,
        Messages: []messaging_api.MessageInterface{
            &messaging_api.TextMessage{Text: text},
        },
    })
}

func pushMessage(userID, text string) {
    bot, _ := messaging_api.NewMessagingApiAPI(os.Getenv("LINE_CHANNEL_TOKEN"))
    bot.PushMessage(&messaging_api.PushMessageRequest{
        To: userID,
        Messages: []messaging_api.MessageInterface{
            &messaging_api.TextMessage{Text: text},
        },
    }, "")
}
```

**注意**: 上記はシンプルな実装例。本番環境ではエラーハンドリングやログ出力を追加すること。

### 11.4 ビルド

```bash
cd ~/cupid
go build -o cupid-bot main.go

# 実行権限を付与
chmod +x cupid-bot
```

---

## 12. 環境変数の設定

### 12.1 .envファイルの作成

```bash
vim ~/cupid/.env
```

以下を記述（`06_linebot_setup.md`で取得した値を使用）：

```bash
LINE_CHANNEL_SECRET=YOUR_CHANNEL_SECRET
LINE_CHANNEL_TOKEN=YOUR_CHANNEL_ACCESS_TOKEN
DATABASE_PATH=/home/ec2-user/cupid/cupid.db
PORT=8080
```

### 12.2 ファイルの権限保護

```bash
chmod 600 ~/cupid/.env
```

---

## 13. systemdサービスの設定

### 13.1 サービスファイルの作成

```bash
sudo vim /etc/systemd/system/cupid.service
```

以下を記述：

```ini
[Unit]
Description=Cupid LINE Bot Service
After=network.target

[Service]
Type=simple
User=ec2-user
WorkingDirectory=/home/ec2-user/cupid
EnvironmentFile=/home/ec2-user/cupid/.env
ExecStart=/home/ec2-user/cupid/cupid-bot
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

### 13.2 サービスの有効化と起動

```bash
# systemdに変更を認識させる
sudo systemctl daemon-reload

# サービスを起動
sudo systemctl start cupid

# 自動起動を有効化
sudo systemctl enable cupid

# 起動確認
sudo systemctl status cupid
```

### 13.3 ログの確認

```bash
# リアルタイムでログを表示
sudo journalctl -u cupid -f

# 最新100行を表示
sudo journalctl -u cupid -n 100
```

---

## 14. 動作確認

### 14.1 ヘルスチェック

```bash
# EC2内から確認
curl http://localhost:8080/health
# 出力: OK

# ローカルマシンから確認（HTTPS）
curl https://cupid.click/health
# 出力: OK
```

### 14.2 LINE Botの動作確認

1. LINE Developers Consoleで Webhook URLを設定（`06_linebot_setup.md`参照）
   - Webhook URL: `https://cupid.click/webhook`
   - Verifyボタンをクリックして「Success」を確認
2. LINE BotをQRコードで友だち追加
3. メッセージを送信して応答を確認

```
You: こんにちは
Bot: はじめまして。あなたの名前を教えて、ね
```

---

## 15. トラブルシューティング

### サービスが起動しない

```bash
# 詳細なログを確認
sudo journalctl -u cupid -n 50 --no-pager

# 環境変数が読み込まれているか確認
sudo systemctl show cupid | grep Environment
```

### Nginxエラー

```bash
# Nginxのエラーログ確認
sudo tail -f /var/log/nginx/error.log

# Nginx設定の文法チェック
sudo nginx -t
```

### データベースエラー

```bash
# ファイルの存在確認
ls -l ~/cupid/cupid.db

# SQLiteに直接接続してテスト
sqlite3 ~/cupid/cupid.db "SELECT * FROM users;"
```

### Let's Encrypt証明書エラー

```bash
# 証明書の状態確認
sudo certbot certificates

# 手動で更新
sudo certbot renew
```

---

## 16. デプロイ後のメンテナンス

### 16.1 アプリケーションの更新

```bash
# コードを更新した後
cd ~/cupid
go build -o cupid-bot main.go

# サービスを再起動
sudo systemctl restart cupid

# ログで確認
sudo journalctl -u cupid -f
```

### 16.2 ログのローテーション

journaldは自動でログローテーションを行うけど、設定を確認：

```bash
# ログの容量制限を確認
sudo journalctl --disk-usage

# 古いログを削除（1週間より古いもの）
sudo journalctl --vacuum-time=7d
```

### 16.3 システムのアップデート

```bash
# 定期的にパッケージを更新
sudo dnf update -y

# 再起動が必要な場合
sudo reboot
```

---

## 17. セキュリティのベストプラクティス

### 17.1 SSHキーの管理

- `.pem`ファイルは絶対に公開しない
- パーミッションは`400`を維持
- 定期的にキーペアを更新

### 17.2 環境変数の保護

- `.env`ファイルの権限は`600`を維持
- GitHubなどにpushしない（`.gitignore`に追加）

### 17.3 セキュリティグループの見直し

- SSH（ポート22）は必ず「マイIP」に制限
- 不要なポートは開放しない

### 17.4 定期的なアップデート

- 毎月1回は`sudo dnf update -y`を実行
- Go、Nginx、Certbotの更新を確認

---

## 完了、よ

⋯⋯これで、EC2上でCupid LINE Botが完全に動作する、ね。

お疲れ様、プロデューサー。
