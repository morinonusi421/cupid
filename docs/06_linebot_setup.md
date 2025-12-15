# LINE Bot セットアップ手順

## 前提条件
- LINEアカウントを持っている
- LINE Developersアカウントを作成済み

---

## 1. LINE Developers Consoleでの設定

### 1.1 プロバイダーの作成（初回のみ）
1. [LINE Developers Console](https://developers.line.biz/console/) にログイン
2. 「プロバイダー」タブをクリック
3. 「作成」ボタンをクリック
4. プロバイダー名を入力（例：`Cupid`）
5. 「作成」をクリック

### 1.2 Messaging APIチャネルの作成
1. 作成したプロバイダーを選択
2. 「チャネルを作成」をクリック
3. 「Messaging API」を選択
4. 以下を入力：
   - **チャネル名**: `Cupid Bot`（任意）
   - **チャネル説明**: `相思相愛マッチングBot`（任意）
   - **大業種**: `個人`
   - **小業種**: `個人（その他）`
   - **メールアドレス**: 自分のメールアドレス
5. 利用規約に同意して「作成」

---

## 2. 認証情報の取得

### 2.1 Channel Secretの取得
1. 作成したチャネルを開く
2. 「Basic settings」タブをクリック
3. 「Channel secret」をコピー
4. メモしておく（後でEC2の環境変数に設定）

### 2.2 Channel Access Tokenの発行
1. 「Messaging API」タブをクリック
2. 「Channel access token (long-lived)」セクションで「Issue」ボタンをクリック
3. 発行されたトークンをコピー
4. メモしておく（後でEC2の環境変数に設定）

---

## 3. Webhook設定（EC2セットアップ後に実施）

### 3.1 Webhook URLの設定
1. 「Messaging API」タブを開く
2. 「Webhook settings」セクションで「Edit」をクリック
3. Webhook URL に以下を入力：
   ```
   https://cupid.click/webhook
   ```
   ※この URLは EC2上でNginx+Let's Encrypt設定後に利用可能
4. 「Update」をクリック

### 3.2 Webhook URLの検証
1. 「Verify」ボタンをクリック
2. 「Success」と表示されることを確認

### 3.3 Webhookの有効化
1. 「Use webhook」をONにする

---

## 4. 応答設定

### 4.1 自動応答メッセージの無効化
1. 「Messaging API」タブを開く
2. 「LINE Official Account features」セクションで「Edit」リンクをクリック
   （LINE Official Account Managerが開く）
3. 「設定」→「応答設定」を開く
4. 以下を設定：
   - **応答メッセージ**: OFF
   - **Webhook**: ON
5. 保存

---

## 5. EC2への環境変数設定

### 5.1 環境変数ファイルの作成

EC2にSSH接続後、環境変数ファイルを作成：

```bash
# /home/ec2-user/cupid/.env ファイルを作成
sudo nano /home/ec2-user/cupid/.env
```

以下の内容を記述：
```bash
LINE_CHANNEL_SECRET=YOUR_CHANNEL_SECRET
LINE_CHANNEL_TOKEN=YOUR_CHANNEL_ACCESS_TOKEN
DATABASE_PATH=/home/ec2-user/cupid/cupid.db
PORT=8080
```

### 5.2 ファイルの権限設定
```bash
# .envファイルを保護
chmod 600 /home/ec2-user/cupid/.env
```

### 5.3 systemdサービスでの読み込み

systemdサービスファイル（`/etc/systemd/system/cupid.service`）で環境変数を読み込む設定を追加：

```ini
[Service]
EnvironmentFile=/home/ec2-user/cupid/.env
```

詳細は「07_ec2_setup.md」を参照、ね。

---

## 6. 友だち追加

### 6.1 QRコードの取得
1. LINE Developers Consoleの「Messaging API」タブを開く
2. 「Bot information」セクションで QRコードを表示
3. LINEアプリでスキャンして友だち追加

### 6.2 友だち追加URL
以下のURLでも追加可能：
```
https://line.me/R/ti/p/@{Basic ID}
```
※Basic IDは「Messaging API」タブで確認可能（例：`@123abcde`）

---

## 7. テスト

### 7.1 初回メッセージ送信
1. 友だち追加したBotにメッセージを送信
2. 「はじめまして。あなたの名前を教えて、ね」と返信されることを確認

### 7.2 登録フローのテスト
会話フローに従ってテスト：
```
You: こんにちは
Bot: はじめまして。あなたの名前を教えて、ね
You: テストユーザー
Bot: テストユーザー、ね。生年月日は？（例：2009-12-21）
You: 2000-01-01
Bot: 登録できた、よ。好きな人の名前は？
```

---

## トラブルシューティング

### Webhookの検証が失敗する
- Nginxが正しく起動しているか確認：`sudo systemctl status nginx`
- ドメインのDNS設定が正しいか確認（Route 53）
- SSL証明書が正しく設定されているか確認：`sudo certbot certificates`
- Goアプリケーションが起動しているか確認：`sudo systemctl status cupid`
- ログを確認：`sudo journalctl -u cupid -f`

### Botが応答しない
- Webhookが有効になっているか確認（LINE Developers Console）
- 応答メッセージがOFFになっているか確認（LINE Official Account Manager）
- Goアプリケーションのログを確認：`sudo journalctl -u cupid -n 100`
- Nginxのエラーログを確認：`sudo tail -f /var/log/nginx/error.log`

### 署名検証エラー
- Channel Secretが`.env`ファイルに正しく設定されているか確認
- `.env`ファイルの権限が正しいか確認：`ls -l /home/ec2-user/cupid/.env`
- systemdサービスで環境変数が読み込まれているか確認：`sudo systemctl show cupid | grep Environment`

### メッセージ送信エラー
- Channel Access Tokenが正しいか確認（`.env`ファイル）
- トークンの有効期限が切れていないか確認（Long-lived tokenは無期限）
- インターネット接続を確認：`curl -I https://api.line.me`

### データベースエラー
- SQLiteファイルが存在するか確認：`ls -l /home/ec2-user/cupid/cupid.db`
- ファイルの権限を確認：`chmod 644 /home/ec2-user/cupid/cupid.db`
- ディスク容量を確認：`df -h`

---

## 参考リンク

- [LINE Messaging API ドキュメント](https://developers.line.biz/ja/docs/messaging-api/)
- [line-bot-sdk-go GitHub](https://github.com/line/line-bot-sdk-go)
- [Webhook イベントオブジェクト](https://developers.line.biz/ja/reference/messaging-api/#webhook-event-objects)
- [メッセージ送信API](https://developers.line.biz/ja/reference/messaging-api/#send-reply-message)
