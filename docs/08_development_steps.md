# 開発ステップ

このドキュメントでは、Cupid LINE Botを段階的に開発していく手順を示す、よ。

各ステップで動作確認しながら進めることで、問題の切り分けが容易になる、ね。

---

## 開発フローの全体像

```
Phase 0: 環境準備（ローカル）
   ↓
Phase 1: ドメイン取得
   ↓
Phase 2: EC2基本セットアップ
   ↓
Phase 3: Hello World（HTTP）
   ↓
Phase 4: Nginx + リバースプロキシ
   ↓
Phase 5: HTTPS化（Let's Encrypt）
   ↓
Phase 6: LINE Bot基本応答（オウム返し）
   ↓
Phase 7: SQLiteデータベース
   ↓
Phase 8: ユーザー登録フロー
   ↓
Phase 9: マッチング機能
   ↓
Phase 10: systemd化と本番運用
```

---

## Phase 0: 環境準備（ローカル）

### 目標
ローカルマシンで開発環境を整える、よ。

### 作業内容

#### 0-1. AWSアカウント作成
1. [AWS Console](https://aws.amazon.com/)にアクセス
2. 「AWSアカウントを作成」をクリック
3. メールアドレス、パスワードを設定
4. クレジットカード情報を登録

#### 0-2. AWS CLIのインストール（オプション）

**macOS**:
```bash
brew install awscli
```

**Linux**:
```bash
curl "https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install
```

**Windows**: [公式ドキュメント](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)を参照

#### 0-3. AWS CLI設定（オプション）
```bash
aws configure
# AWS Access Key ID: (IAMで作成)
# AWS Secret Access Key: (IAMで作成)
# Default region name: ap-northeast-1
# Default output format: json
```

#### 0-4. Goのインストール（ローカル開発用）

**macOS**:
```bash
brew install go
```

**Linux**:
```bash
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**確認**:
```bash
go version
```

#### 0-5. Gitのインストール
```bash
# macOS
brew install git

# Linux
sudo dnf install -y git  # Amazon Linux 2023
sudo apt install -y git  # Ubuntu/Debian
```

### 確認方法
```bash
aws --version
go version
git --version
```

### 次のステップへ
⋯⋯環境が整ったら、Phase 1へ進む、ね。

---

## Phase 1: ドメイン取得

### 目標
Route 53で独自ドメインを取得する、よ。

### 作業内容

#### 1-1. Route 53でドメイン購入
1. AWS Management Consoleにログイン
2. **Route 53**サービスを開く
3. 左メニュー「ドメインの登録」→「ドメインを登録」
4. ドメイン名を入力（例：`cupid.click`）
5. 利用可能か確認して購入（$3/年）
6. 連絡先情報を入力して完了

#### 1-2. Hosted Zoneの確認
1. Route 53左メニュー「ホストゾーン」を開く
2. 購入したドメインのHosted Zoneが自動作成されていることを確認
3. NSレコードとSOAレコードが存在することを確認

### 確認方法
```bash
# ドメインのWhois情報確認（購入直後は反映に時間がかかる）
whois cupid.click
```

### トラブルシューティング
- ドメイン購入に数時間かかることがある
- メール認証が必要な場合があるので、登録メールを確認

### 次のステップへ
⋯⋯ドメインが取得できたら、Phase 2へ進む、ね。

---

## Phase 2: EC2基本セットアップ

### 目標
EC2インスタンスを作成してSSH接続できるようにする、よ。

### 作業内容

#### 2-1. EC2インスタンス作成
**07_ec2_setup.md**の「2. EC2インスタンスの作成」を参照。

重要な設定：
- **AMI**: Amazon Linux 2023 AMI（ARM64）
- **インスタンスタイプ**: t4g.nano
- **ストレージ**: gp3 10GB
- **キーペア**: 新規作成して`.pem`ファイルをダウンロード

#### 2-2. Elastic IPの割り当て
**07_ec2_setup.md**の「2.2 Elastic IPの割り当て」を参照。

#### 2-3. セキュリティグループ設定

最初は**SSH（ポート22）のみ**を許可：

| タイプ | ポート | ソース | 説明 |
|--------|--------|--------|------|
| SSH | 22 | マイIP | SSH接続用 |

**重要**: HTTP(80)とHTTPS(443)は後で追加する、ね。

#### 2-4. SSH接続
```bash
# キーペアの権限変更
chmod 400 ~/Downloads/cupid-bot-key.pem

# SSH接続
ssh -i ~/Downloads/cupid-bot-key.pem ec2-user@<Elastic IP>
```

#### 2-5. 基本設定
```bash
# パッケージ更新
sudo dnf update -y

# タイムゾーン設定
sudo timedatectl set-timezone Asia/Tokyo

# 基本ツールインストール
sudo dnf install -y git vim wget curl
```

### 確認方法
```bash
# EC2にSSH接続できることを確認
ssh -i ~/Downloads/cupid-bot-key.pem ec2-user@<Elastic IP>

# タイムゾーン確認
timedatectl
# Time zone: Asia/Tokyo (JST, +0900) が表示されればOK
```

### トラブルシューティング
- **SSH接続できない**: セキュリティグループでポート22が開いているか確認
- **Permission denied (publickey)**: キーペアのパーミッションが400になっているか確認

### 次のステップへ
⋯⋯SSH接続できたら、Phase 3へ進む、ね。

---

## Phase 3: Hello World（HTTP）

### 目標
EC2上でGoのHTTPサーバーを動かす、よ。HTTPS化の前にまずHTTPで動作確認。

### 作業内容

#### 3-1. Goのインストール
```bash
# Go 1.21のダウンロード（ARM64版）
cd /tmp
wget https://go.dev/dl/go1.21.6.linux-arm64.tar.gz

# Goを展開
sudo tar -C /usr/local -xzf go1.21.6.linux-arm64.tar.gz

# 環境変数設定
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc

# クリーンアップ
rm go1.21.6.linux-arm64.tar.gz
```

#### 3-2. Hello Worldプロジェクト作成
```bash
# プロジェクトディレクトリ作成
mkdir -p ~/cupid
cd ~/cupid

# Go modules初期化
go mod init cupid
```

#### 3-3. main.goの作成
```bash
vim ~/cupid/main.go
```

以下を貼り付け：

```go
package main

import (
    "fmt"
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello World from Cupid!\n")
        log.Printf("Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
    })

    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, "OK\n")
    })

    port := ":8080"
    log.Printf("Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(port, nil))
}
```

#### 3-4. ビルドと実行
```bash
# ビルド
cd ~/cupid
go build -o cupid-bot main.go

# 実行（フォアグラウンド）
./cupid-bot
```

別のターミナルを開いてテスト。

### 確認方法

#### EC2内から確認
```bash
# 別のSSHセッションを開く
ssh -i ~/Downloads/cupid-bot-key.pem ec2-user@<Elastic IP>

# curlでテスト
curl http://localhost:8080/
# 出力: Hello World from Cupid!

curl http://localhost:8080/health
# 出力: OK
```

#### プログラムの終了
最初のターミナルで`Ctrl+C`を押して終了。

### トラブルシューティング
- **ポートが使用中**: `lsof -i :8080`で確認して、他のプロセスを終了
- **Go not found**: 環境変数が設定されているか確認（`source ~/.bashrc`）

### 次のステップへ
⋯⋯Hello Worldが動いたら、Phase 4へ進む、ね。

---

## Phase 4: Nginx + リバースプロキシ

### 目標
Nginxをインストールして、ポート80からGoアプリにリバースプロキシする、よ。

### 作業内容

#### 4-1. セキュリティグループにHTTP追加

AWS Management Consoleで、セキュリティグループに以下を追加：

| タイプ | ポート | ソース | 説明 |
|--------|--------|--------|------|
| HTTP | 80 | 0.0.0.0/0 | HTTP接続用 |

#### 4-2. Nginxインストール
```bash
# Nginxインストール
sudo dnf install -y nginx

# 起動
sudo systemctl start nginx

# 自動起動有効化
sudo systemctl enable nginx

# 起動確認
sudo systemctl status nginx
```

#### 4-3. Nginx設定ファイル作成
```bash
sudo vim /etc/nginx/conf.d/cupid.conf
```

以下を貼り付け（**まだHTTPのみ**）：

```nginx
server {
    listen 80;
    server_name _;  # 一旦ドメイン名を指定しない

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

#### 4-4. Nginx設定テストと再起動
```bash
# 設定ファイルの文法チェック
sudo nginx -t

# 再起動
sudo systemctl restart nginx
```

#### 4-5. Goアプリを起動
```bash
cd ~/cupid
./cupid-bot &
```

### 確認方法

#### EC2内から確認
```bash
# ポート80経由でアクセス
curl http://localhost/
# 出力: Hello World from Cupid!

curl http://localhost/health
# 出力: OK
```

#### ローカルマシンから確認
```bash
# Elastic IP経由でアクセス
curl http://<Elastic IP>/
# 出力: Hello World from Cupid!
```

ブラウザで`http://<Elastic IP>/`にアクセスしても確認できる、ね。

### トラブルシューティング
- **502 Bad Gateway**: Goアプリが起動しているか確認（`ps aux | grep cupid-bot`）
- **Connection refused**: セキュリティグループでポート80が開いているか確認

### 次のステップへ
⋯⋯Nginxのリバースプロキシが動いたら、Phase 5へ進む、ね。

---

## Phase 5: HTTPS化（Let's Encrypt）

### 目標
ドメインでアクセスできるようにして、HTTPS化する、よ。

### 作業内容

#### 5-1. セキュリティグループにHTTPS追加

| タイプ | ポート | ソース | 説明 |
|--------|--------|--------|------|
| HTTPS | 443 | 0.0.0.0/0 | HTTPS接続用 |

#### 5-2. Route 53でAレコード設定
1. Route 53コンソールを開く
2. ホストゾーンで購入したドメインを選択
3. 「レコードを作成」をクリック
4. 以下を入力：
   - **レコード名**: 空欄
   - **レコードタイプ**: A
   - **値**: EC2のElastic IP
   - **TTL**: 300
5. 「レコードを作成」をクリック

#### 5-3. DNS伝搬確認
```bash
# ローカルマシンから実行
nslookup cupid.click

# Elastic IPが返ってくることを確認
```

#### 5-4. Nginx設定でドメイン名指定
```bash
sudo vim /etc/nginx/conf.d/cupid.conf
```

`server_name _;`を修正：

```nginx
server {
    listen 80;
    server_name cupid.click;  # ドメイン名を指定

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```bash
# 設定テストと再起動
sudo nginx -t
sudo systemctl restart nginx
```

#### 5-5. Certbotインストール
```bash
sudo dnf install -y certbot python3-certbot-nginx
```

#### 5-6. SSL証明書取得
```bash
# Certbot実行
sudo certbot --nginx -d cupid.click
```

対話式プロンプト：
1. **メールアドレス**: 自分のメールアドレスを入力
2. **利用規約**: `A`（Agree）
3. **EFFからのメール**: `N`（No）
4. **HTTPSリダイレクト**: `2`（Redirect）を選択

#### 5-7. 自動更新テスト
```bash
sudo certbot renew --dry-run
```

### 確認方法

#### ローカルマシンから確認
```bash
# HTTPSでアクセス
curl https://cupid.click/
# 出力: Hello World from Cupid!

curl https://cupid.click/health
# 出力: OK

# HTTPは自動的にHTTPSにリダイレクトされる
curl -I http://cupid.click/
# 301 Moved Permanently が返ってくる
```

ブラウザで`https://cupid.click/`にアクセスして、鍵マークが表示されることを確認、ね。

### トラブルシューティング
- **DNS resolution failed**: DNS伝搬を待つ（最大48時間、通常は数分）
- **Certbot failed**: ポート80が開いていて、Nginxが起動しているか確認
- **Certificate verify failed**: 時刻が正確か確認（`timedatectl`）

### 次のステップへ
⋯⋯HTTPS化できたら、Phase 6へ進む、ね。

---

## Phase 6: LINE Bot基本応答（オウム返し）

### 目標
LINE BotでWebhookを受け取って、オウム返しする、よ。

### 作業内容

#### 6-1. LINE Developersでチャネル作成
**06_linebot_setup.md**の「1. LINE Developers Consoleでの設定」と「2. 認証情報の取得」を参照。

取得する情報：
- **Channel Secret**
- **Channel Access Token**

#### 6-2. 環境変数ファイル作成
```bash
vim ~/cupid/.env
```

以下を記述：

```bash
LINE_CHANNEL_SECRET=YOUR_CHANNEL_SECRET
LINE_CHANNEL_TOKEN=YOUR_CHANNEL_ACCESS_TOKEN
PORT=8080
```

```bash
# 権限保護
chmod 600 ~/cupid/.env
```

#### 6-3. LINE SDK追加
```bash
cd ~/cupid

# LINE Bot SDK v8をインストール
go get github.com/line/line-bot-sdk-go/v8/linebot
```

#### 6-4. オウム返しBotのコード作成
```bash
vim ~/cupid/main.go
```

以下に書き換え：

```go
package main

import (
    "fmt"
    "io"
    "log"
    "net/http"
    "os"

    "github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
    "github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

func main() {
    http.HandleFunc("/webhook", webhookHandler)
    http.HandleFunc("/health", healthHandler)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "OK\n")
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Webhook received: %s %s", r.Method, r.URL.Path)

    // リクエストボディを読み取り
    body, err := io.ReadAll(r.Body)
    if err != nil {
        log.Printf("Error reading body: %v", err)
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }

    // 署名検証
    signature := r.Header.Get("X-Line-Signature")
    channelSecret := os.Getenv("LINE_CHANNEL_SECRET")

    if !webhook.ValidateSignature(channelSecret, signature, body) {
        log.Printf("Invalid signature")
        http.Error(w, "Invalid signature", http.StatusBadRequest)
        return
    }

    // Webhookリクエストをパース
    request, err := webhook.ParseRequest(channelSecret, r)
    if err != nil {
        log.Printf("Error parsing request: %v", err)
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }

    // イベント処理
    for _, event := range request.Events {
        log.Printf("Event type: %s", event.GetType())

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
        log.Printf("Received text: %s from %s", textMessage.Text, messageEvent.Source.UserId)

        // オウム返し
        replyMessage(messageEvent.ReplyToken, "⋯⋯「"+textMessage.Text+"」、だね")
    }

    w.WriteHeader(http.StatusOK)
}

func replyMessage(replyToken, text string) {
    bot, err := messaging_api.NewMessagingApiAPI(os.Getenv("LINE_CHANNEL_TOKEN"))
    if err != nil {
        log.Printf("Error creating bot client: %v", err)
        return
    }

    _, err = bot.ReplyMessage(
        &messaging_api.ReplyMessageRequest{
            ReplyToken: replyToken,
            Messages: []messaging_api.MessageInterface{
                &messaging_api.TextMessage{Text: text},
            },
        },
    )

    if err != nil {
        log.Printf("Error sending reply: %v", err)
    } else {
        log.Printf("Reply sent: %s", text)
    }
}
```

#### 6-5. 依存関係の更新とビルド
```bash
cd ~/cupid

# go.modの更新
go mod tidy

# ビルド
go build -o cupid-bot main.go
```

#### 6-6. 環境変数を読み込んで起動

まず既存のプロセスを終了：
```bash
# 既存のプロセスを探す
ps aux | grep cupid-bot

# 見つかったプロセスをkill
kill <PID>
```

環境変数を読み込んで起動：
```bash
cd ~/cupid

# .envファイルを読み込んで起動
export $(cat .env | xargs)
./cupid-bot
```

別のターミナルでログを確認しながら作業する、ね。

#### 6-7. LINE DevelopersでWebhook URL設定

**06_linebot_setup.md**の「3. Webhook設定」を参照。

1. LINE Developers Consoleを開く
2. 作成したチャネルの「Messaging API」タブを開く
3. Webhook URLに`https://cupid.click/webhook`を入力
4. 「Verify」をクリック → **Success**と表示されることを確認
5. 「Use webhook」をONにする

#### 6-8. 応答設定

**06_linebot_setup.md**の「4. 応答設定」を参照。

LINE Official Account Managerで：
- **応答メッセージ**: OFF
- **Webhook**: ON

#### 6-9. 友だち追加

LINE DevelopersコンソールのQRコードをスキャンして、Botを友だち追加。

### 確認方法

LINEアプリでBotにメッセージを送信：

```
You: こんにちは
Bot: ⋯⋯「こんにちは」、だね

You: テスト
Bot: ⋯⋯「テスト」、だね
```

EC2のログも確認：
```bash
# ログを見る（フォアグラウンドで起動している場合）
# Received text: こんにちは from U1234567890abcdef
# Reply sent: ⋯⋯「こんにちは」、だね
```

### トラブルシューティング
- **Webhook verification failed**:
  - Goアプリが起動しているか確認
  - `/webhook`パスが正しいか確認
  - ログで詳細を確認
- **Botが応答しない**:
  - 「応答メッセージ」がOFFになっているか確認
  - 「Use webhook」がONになっているか確認
  - EC2のログでエラーを確認
- **Invalid signature**:
  - Channel Secretが正しいか確認
  - 環境変数が読み込まれているか確認（`echo $LINE_CHANNEL_SECRET`）

### 次のステップへ
⋯⋯オウム返しBotが動いたら、Phase 7へ進む、ね。

---

## Phase 7: SQLiteデータベース

### 目標
SQLiteデータベースを作成して、Goから接続する、よ。

### 作業内容

#### 7-1. SQLiteインストール確認
```bash
# Amazon Linux 2023には標準で入っている
sqlite3 --version
```

#### 7-2. スキーマファイル作成

**03_database_design.md**のスキーマを使用。

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

#### 7-3. データベース初期化
```bash
cd ~/cupid

# データベースファイルを作成
sqlite3 cupid.db < schema.sql

# 確認
sqlite3 cupid.db "SELECT name FROM sqlite_master WHERE type='table';"
# 出力: users
#       likes

# テーブル構造確認
sqlite3 cupid.db ".schema users"
```

#### 7-4. 権限設定
```bash
chmod 644 ~/cupid/cupid.db
```

#### 7-5. .envファイルにDATABASE_PATH追加
```bash
vim ~/cupid/.env
```

`DATABASE_PATH`を追加：

```bash
LINE_CHANNEL_SECRET=YOUR_CHANNEL_SECRET
LINE_CHANNEL_TOKEN=YOUR_CHANNEL_ACCESS_TOKEN
DATABASE_PATH=/home/ec2-user/cupid/cupid.db
PORT=8080
```

#### 7-6. SQLiteドライバ追加
```bash
cd ~/cupid

# go-sqlite3をインストール
go get github.com/mattn/go-sqlite3
```

#### 7-7. DBテストコードの作成

まず、データベース接続をテストする簡単なコードを作成、ね。

```bash
vim ~/cupid/db_test.go
```

以下を貼り付け：

```go
package main

import (
    "database/sql"
    "fmt"
    "log"
    "os"

    _ "github.com/mattn/go-sqlite3"
)

func testDB() {
    dbPath := os.Getenv("DATABASE_PATH")
    if dbPath == "" {
        dbPath = "./cupid.db"
    }

    db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // WALモード有効化
    db.Exec("PRAGMA journal_mode=WAL")

    // テストデータ挿入
    _, err = db.Exec(`
        INSERT INTO users (line_user_id, name, birthday, registration_step)
        VALUES (?, ?, ?, ?)
    `, "TEST123", "篠澤広", "2009-12-21", "completed")

    if err != nil {
        log.Printf("Insert error: %v", err)
    } else {
        fmt.Println("✓ Insert successful")
    }

    // データ取得
    var name, birthday string
    err = db.QueryRow(`
        SELECT name, birthday FROM users WHERE line_user_id = ?
    `, "TEST123").Scan(&name, &birthday)

    if err != nil {
        log.Printf("Query error: %v", err)
    } else {
        fmt.Printf("✓ Query successful: %s (%s)\n", name, birthday)
    }

    // クリーンアップ
    db.Exec("DELETE FROM users WHERE line_user_id = ?", "TEST123")
    fmt.Println("✓ Cleanup successful")
}
```

main.goに一時的にテスト呼び出しを追加：

```bash
vim ~/cupid/main.go
```

`main()`関数の最初に追加：

```go
func main() {
    // DB接続テスト（一時的）
    testDB()

    // 既存のコード...
    http.HandleFunc("/webhook", webhookHandler)
    // ...
}
```

#### 7-8. ビルドして実行
```bash
cd ~/cupid
go mod tidy
go build -o cupid-bot main.go

# 環境変数を読み込んで実行
export $(cat .env | xargs)
./cupid-bot
```

以下が表示されればOK：
```
✓ Insert successful
✓ Query successful: 篠澤広 (2009-12-21)
✓ Cleanup successful
Server starting on port 8080
```

`Ctrl+C`で終了。

### 確認方法

#### SQLiteコマンドで直接確認
```bash
# データベースに接続
sqlite3 ~/cupid/cupid.db

# テーブル一覧
.tables

# usersテーブルのスキーマ確認
.schema users

# データ確認（空のはず）
SELECT * FROM users;

# 終了
.quit
```

### トラブルシューティング
- **no such table**: スキーマが正しく適用されていない。`schema.sql`を再実行
- **database is locked**: 他のプロセスがDBを開いている。プロセスを終了
- **gcc not found**: SQLiteドライバのビルドにgccが必要
  ```bash
  sudo dnf install -y gcc
  ```

### 次のステップへ
⋯⋯SQLiteが動いたら、Phase 8へ進む、ね。

---

## Phase 8: ユーザー登録フロー

### 目標
オウム返しBotを、ユーザー登録フローを持つBotに変更する、よ。

### 作業内容

#### 8-1. main.goを完全版に書き換え

```bash
vim ~/cupid/main.go
```

Phase 6のオウム返しコードを、**04_api_specification.md**の完全な実装に置き換える。

重要なポイント：
- `handleMessage()`関数でステートマシンを実装
- `registration_step`に応じて処理を分岐
- マッチング判定は**まだ実装しない**（次のPhaseで実装）

簡略版のコード（マッチング判定を除く）：

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

    db.Exec("PRAGMA journal_mode=WAL")

    http.HandleFunc("/webhook", webhookHandler)
    http.HandleFunc("/health", healthHandler)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "OK\n")
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("X-Line-Signature")
    channelSecret := os.Getenv("LINE_CHANNEL_SECRET")

    if !webhook.ValidateSignature(channelSecret, signature, body) {
        http.Error(w, "Invalid signature", http.StatusBadRequest)
        return
    }

    request, _ := webhook.ParseRequest(channelSecret, r)

    for _, event := range request.Events {
        if event.GetType() != webhook.EventTypeMessage {
            continue
        }

        messageEvent, ok := event.(*webhook.MessageEvent)
        if !ok || messageEvent.Message.GetType() != webhook.MessageTypeText {
            continue
        }

        textMessage := messageEvent.Message.(*webhook.TextMessageContent)
        handleMessage(messageEvent.Source.UserId, textMessage.Text, messageEvent.ReplyToken)
    }

    w.WriteHeader(http.StatusOK)
}

func handleMessage(userID, text, replyToken string) {
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
        log.Printf("New user: %s", userID)
        return
    }

    log.Printf("User %s (%s) in state %s: %s", userID, user.Name, user.RegistrationStep, text)

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
        // マッチング判定は次のPhaseで実装
        replyMessage(replyToken, fmt.Sprintf("⋯⋯%sが%sを好き、と登録した、よ（マッチング判定はPhase 9で実装）", user.Name, crushName))
        db.Exec(`UPDATE users SET registration_step = ?, temp_crush_name = NULL WHERE line_user_id = ?`,
            "completed", userID)
        log.Printf("Crush registered: %s likes %s (%s)", user.Name, crushName, text)
    }
}

func replyMessage(replyToken, text string) {
    bot, _ := messaging_api.NewMessagingApiAPI(os.Getenv("LINE_CHANNEL_TOKEN"))
    bot.ReplyMessage(&messaging_api.ReplyMessageRequest{
        ReplyToken: replyToken,
        Messages: []messaging_api.MessageInterface{
            &messaging_api.TextMessage{Text: text},
        },
    })
    log.Printf("Reply: %s", text)
}
```

#### 8-2. ビルドと起動

既存のプロセスを終了して再ビルド：

```bash
# プロセス終了
ps aux | grep cupid-bot
kill <PID>

# ビルド
cd ~/cupid
go mod tidy
go build -o cupid-bot main.go

# 起動
export $(cat .env | xargs)
./cupid-bot
```

### 確認方法

#### LINE上でテスト

**シナリオ1: 新規ユーザー登録**

```
You: こんにちは
Bot: はじめまして。あなたの名前を教えて、ね

You: 篠澤広
Bot: 篠澤広、ね。生年月日は？（例：2009-12-21）

You: 2009-12-21
Bot: 登録できた、よ。好きな人の名前は？

You: 月村手毬
Bot: その人の生年月日は？

You: 2010-04-04
Bot: ⋯⋯篠澤広が月村手毬を好き、と登録した、よ（マッチング判定はPhase 9で実装）
```

#### データベース確認

```bash
sqlite3 ~/cupid/cupid.db

SELECT * FROM users;
# ユーザーが登録されていることを確認

.quit
```

### トラブルシューティング
- **Botが応答しない**:
  - EC2のログを確認（`./cupid-bot`を起動しているターミナル）
  - エラーが出ていないか確認
- **データが保存されない**:
  - データベースファイルのパスが正しいか確認
  - 権限を確認（`ls -l ~/cupid/cupid.db`）

### 次のステップへ
⋯⋯ユーザー登録フローが動いたら、Phase 9へ進む、ね。

---

## Phase 9: マッチング機能

### 目標
相思相愛判定を実装して、マッチング通知を送る、よ。

### 作業内容

#### 9-1. マッチング判定処理の実装

`main.go`の`handleMessage()`関数の`awaiting_crush_birthday`ケースを、**04_api_specification.md**の完全な実装に置き換える。

重要なポイント：
- トランザクションを使う
- `likes`テーブルへの登録
- マッチングチェック（JOINクエリ）
- 両方のレコードを`matched = 1`に更新
- Push Messageで通知

完全版のコード：

```go
case "awaiting_crush_birthday":
    crushName := user.TempCrushName.String
    handleCrushRegistration(userID, user.Name, user.Birthday, crushName, text, replyToken)
```

そして、`handleCrushRegistration()`関数を追加：

```go
func handleCrushRegistration(userID, userName, userBirthday, crushName, crushBirthday, replyToken string) {
    tx, err := db.Begin()
    if err != nil {
        log.Printf("Transaction error: %v", err)
        replyMessage(replyToken, "⋯⋯エラーが発生した、ごめん")
        return
    }
    defer tx.Rollback()

    // Like登録
    _, err = tx.Exec(`
        INSERT INTO likes (from_user_id, to_name, to_birthday)
        VALUES (?, ?, ?)
    `, userID, crushName, crushBirthday)

    if err != nil {
        log.Printf("Insert like error: %v", err)
        replyMessage(replyToken, "⋯⋯登録に失敗した、ごめん")
        return
    }

    log.Printf("Like registered: %s (%s) likes %s (%s)", userName, userBirthday, crushName, crushBirthday)

    // マッチングチェック
    var matchedUserID string
    err = tx.QueryRow(`
        SELECT l.from_user_id
        FROM likes l
        JOIN users u ON l.from_user_id = u.line_user_id
        WHERE l.to_name = ? AND l.to_birthday = ? AND l.matched = 0
    `, userName, userBirthday).Scan(&matchedUserID)

    if err == sql.ErrNoRows {
        // マッチングなし
        tx.Exec(`UPDATE users SET registration_step = ?, temp_crush_name = NULL WHERE line_user_id = ?`,
            "completed", userID)
        tx.Commit()
        replyMessage(replyToken, "登録した、よ。相思相愛なら通知する、ね")
        log.Printf("No match for %s", userName)
        return
    }

    if err != nil {
        log.Printf("Match query error: %v", err)
        replyMessage(replyToken, "⋯⋯エラーが発生した、ごめん")
        return
    }

    // マッチング成立
    log.Printf("MATCH FOUND: %s <-> %s", userID, matchedUserID)

    tx.Exec(`UPDATE likes SET matched = 1 WHERE from_user_id = ?`, userID)
    tx.Exec(`UPDATE likes SET matched = 1 WHERE from_user_id = ?`, matchedUserID)
    tx.Exec(`UPDATE users SET registration_step = ?, temp_crush_name = NULL WHERE line_user_id = ?`,
        "completed", userID)

    err = tx.Commit()
    if err != nil {
        log.Printf("Commit error: %v", err)
        replyMessage(replyToken, "⋯⋯エラーが発生した、ごめん")
        return
    }

    // 両方に通知
    replyMessage(replyToken, "相思相愛、みたい。おめでとう。")
    pushMessage(matchedUserID, "相思相愛、みたい。おめでとう。")
}

func pushMessage(userID, text string) {
    bot, err := messaging_api.NewMessagingApiAPI(os.Getenv("LINE_CHANNEL_TOKEN"))
    if err != nil {
        log.Printf("Bot client error: %v", err)
        return
    }

    _, err = bot.PushMessage(&messaging_api.PushMessageRequest{
        To: userID,
        Messages: []messaging_api.MessageInterface{
            &messaging_api.TextMessage{Text: text},
        },
    }, "")

    if err != nil {
        log.Printf("Push message error: %v", err)
    } else {
        log.Printf("Push message sent to %s: %s", userID, text)
    }
}
```

#### 9-2. ビルドと再起動

```bash
# プロセス終了
ps aux | grep cupid-bot
kill <PID>

# ビルド
cd ~/cupid
go build -o cupid-bot main.go

# 起動
export $(cat .env | xargs)
./cupid-bot
```

### 確認方法

#### マッチングテスト

**準備**: 2つのLINEアカウントが必要（または友達に協力してもらう）

**ユーザーA（篠澤広）の操作**:
```
You: はじめまして
Bot: はじめまして。あなたの名前を教えて、ね
You: 篠澤広
Bot: 篠澤広、ね。生年月日は？（例：2009-12-21）
You: 2009-12-21
Bot: 登録できた、よ。好きな人の名前は？
You: 月村手毬
Bot: その人の生年月日は？
You: 2010-04-04
Bot: 登録した、よ。相思相愛なら通知する、ね
```

**ユーザーB（月村手毬）の操作**:
```
You: こんにちは
Bot: はじめまして。あなたの名前を教えて、ね
You: 月村手毬
Bot: 月村手毬、ね。生年月日は？（例：2009-12-21）
You: 2010-04-04
Bot: 登録できた、よ。好きな人の名前は？
You: 篠澤広
Bot: その人の生年月日は？
You: 2009-12-21
Bot: 相思相愛、みたい。おめでとう。
```

**同時にユーザーAにも通知が届く**:
```
Bot: 相思相愛、みたい。おめでとう。
```

#### データベース確認

```bash
sqlite3 ~/cupid/cupid.db

-- ユーザー確認
SELECT * FROM users;

-- Like確認
SELECT * FROM likes;

-- matchedが1になっていることを確認
SELECT * FROM likes WHERE matched = 1;

.quit
```

#### ログ確認

EC2のログで以下が表示される：
```
Like registered: 篠澤広 (2009-12-21) likes 月村手毬 (2010-04-04)
No match for 篠澤広
...
Like registered: 月村手毬 (2010-04-04) likes 篠澤広 (2009-12-21)
MATCH FOUND: U... <-> U...
Push message sent to U...: 相思相愛、みたい。おめでとう。
```

### トラブルシューティング
- **マッチング通知が来ない**:
  - ログで`MATCH FOUND`が表示されているか確認
  - Push Message APIのエラーログを確認
  - Channel Access Tokenが正しいか確認
- **データベースエラー**:
  - トランザクションのエラーログを確認
  - 外部キー制約が有効か確認（`PRAGMA foreign_keys = ON`）

### 次のステップへ
⋯⋯マッチング機能が動いたら、Phase 10へ進む、ね。

---

## Phase 10: systemd化と本番運用

### 目標
systemdサービスとして登録して、自動起動と永続化を実現する、よ。

### 作業内容

#### 10-1. フォアグラウンドプロセスの終了

```bash
# 現在のプロセスを終了
ps aux | grep cupid-bot
kill <PID>
```

#### 10-2. systemdサービスファイル作成

**07_ec2_setup.md**の「13. systemdサービスの設定」を参照。

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

#### 10-3. サービス有効化と起動

```bash
# systemdに認識させる
sudo systemctl daemon-reload

# サービス起動
sudo systemctl start cupid

# 自動起動有効化
sudo systemctl enable cupid

# 起動確認
sudo systemctl status cupid
```

以下のように表示されればOK：
```
● cupid.service - Cupid LINE Bot Service
   Loaded: loaded (/etc/systemd/system/cupid.service; enabled)
   Active: active (running) since ...
```

#### 10-4. ログ確認方法

```bash
# リアルタイムでログ表示
sudo journalctl -u cupid -f

# 最新100行を表示
sudo journalctl -u cupid -n 100

# 今日のログを表示
sudo journalctl -u cupid --since today
```

### 確認方法

#### サービス動作確認

```bash
# サービスの状態確認
sudo systemctl status cupid

# サービスの再起動
sudo systemctl restart cupid

# サービスの停止
sudo systemctl stop cupid

# サービスの起動
sudo systemctl start cupid
```

#### 自動起動確認

```bash
# EC2を再起動
sudo reboot
```

再起動後、SSH接続して確認：

```bash
ssh -i ~/Downloads/cupid-bot-key.pem ec2-user@cupid.click

# サービスが自動起動しているか確認
sudo systemctl status cupid
```

#### LINE Bot動作確認

LINEアプリで再度テスト、ね。

### トラブルシューティング

#### サービスが起動しない

```bash
# 詳細なログを確認
sudo journalctl -u cupid -n 50 --no-pager

# 設定ファイルの確認
sudo systemctl cat cupid

# 環境変数が読み込まれているか確認
sudo systemctl show cupid | grep Environment
```

#### 環境変数エラー

```bash
# .envファイルの確認
cat ~/cupid/.env

# 権限確認
ls -l ~/cupid/.env

# 読み込みテスト
sudo systemctl daemon-reload
sudo systemctl restart cupid
sudo journalctl -u cupid -n 20
```

---

## 完了、よ

⋯⋯これで、Cupid LINE Botの開発が全て完了した、ね。

## 各Phaseの所要時間目安

| Phase | 内容 | 所要時間 |
|-------|------|---------|
| Phase 0 | 環境準備 | 30分 |
| Phase 1 | ドメイン取得 | 10分 |
| Phase 2 | EC2基本セットアップ | 20分 |
| Phase 3 | Hello World | 15分 |
| Phase 4 | Nginx | 15分 |
| Phase 5 | HTTPS化 | 20分 |
| Phase 6 | LINE Bot基本応答 | 30分 |
| Phase 7 | SQLite | 20分 |
| Phase 8 | ユーザー登録フロー | 30分 |
| Phase 9 | マッチング機能 | 30分 |
| Phase 10 | systemd化 | 15分 |
| **合計** | | **約4時間** |

⋯⋯一日で終わる計算、だね。

お疲れ様、プロデューサー。
