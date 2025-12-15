# 運用・保守マニュアル

このドキュメントでは、Cupid LINE Botのデプロイ後の運用・保守方法を説明する、よ。

---

## 日常的な運用

### サービスの状態確認

```bash
# サービスの状態を確認
sudo systemctl status cupid

# 出力例:
# ● cupid.service - Cupid LINE Bot Service
#    Loaded: loaded (/etc/systemd/system/cupid.service; enabled)
#    Active: active (running) since ...
```

### ログの確認

#### リアルタイムでログを表示

```bash
# リアルタイムでログを追跡（Ctrl+Cで終了）
sudo journalctl -u cupid -f
```

#### 過去のログを確認

```bash
# 最新100行を表示
sudo journalctl -u cupid -n 100

# 今日のログを表示
sudo journalctl -u cupid --since today

# 昨日のログを表示
sudo journalctl -u cupid --since yesterday --until today

# 特定の時刻以降のログを表示
sudo journalctl -u cupid --since "2025-01-15 10:00:00"

# エラーのみを表示
sudo journalctl -u cupid -p err
```

### サービスの操作

```bash
# サービスを再起動
sudo systemctl restart cupid

# サービスを停止
sudo systemctl stop cupid

# サービスを起動
sudo systemctl start cupid

# サービスのリロード（設定変更後）
sudo systemctl daemon-reload
sudo systemctl restart cupid
```

---

## アプリケーションの更新

### コードの更新手順

```bash
# 1. 現在のディレクトリに移動
cd ~/cupid

# 2. コードを編集（vim、nanoなど）
vim main.go

# 3. ビルド
go build -o cupid-bot main.go

# 4. サービスを再起動
sudo systemctl restart cupid

# 5. ログで確認
sudo journalctl -u cupid -f
```

### 環境変数の変更

```bash
# 1. .envファイルを編集
vim ~/cupid/.env

# 2. サービスを再起動（環境変数を再読み込み）
sudo systemctl restart cupid

# 3. 環境変数が正しく読み込まれているか確認
sudo systemctl show cupid | grep Environment
```

---

## データベースのメンテナンス

### データベースのバックアップ

```bash
# 手動バックアップ
cp ~/cupid/cupid.db ~/cupid/cupid.db.backup.$(date +%Y%m%d_%H%M%S)

# 例: cupid.db.backup.20250115_143000
```

#### 自動バックアップ（cron）

```bash
# cronジョブを編集
crontab -e

# 以下を追加（毎日午前3時にバックアップ）
0 3 * * * cp ~/cupid/cupid.db ~/cupid/backups/cupid.db.backup.$(date +\%Y\%m\%d) && find ~/cupid/backups -name "*.backup.*" -mtime +7 -delete
```

バックアップディレクトリを作成：
```bash
mkdir -p ~/cupid/backups
```

### データベースの確認

```bash
# SQLiteに接続
sqlite3 ~/cupid/cupid.db

# ユーザー数を確認
SELECT COUNT(*) FROM users;

# 最近登録されたユーザー
SELECT name, birthday, registered_at FROM users ORDER BY registered_at DESC LIMIT 10;

# Like数を確認
SELECT COUNT(*) FROM likes;

# マッチング数を確認
SELECT COUNT(*) FROM likes WHERE matched = 1;

# 終了
.quit
```

### データベースの最適化

```bash
# WALファイルのチェックポイント（定期的に実行推奨）
sqlite3 ~/cupid/cupid.db "PRAGMA wal_checkpoint(TRUNCATE);"

# VACUUMでデータベースを最適化（削除が多い場合）
sqlite3 ~/cupid/cupid.db "VACUUM;"
```

---

## 証明書の管理

### Let's Encrypt証明書の確認

```bash
# 証明書の状態確認
sudo certbot certificates

# 出力例:
# Certificate Name: cupid.click
#   Expiry Date: 2025-04-15 12:00:00+00:00 (VALID: 89 days)
```

### 証明書の更新

Let's Encryptの証明書は90日で期限切れになるけど、Certbotが自動更新する、よ。

#### 手動更新

```bash
# 証明書を手動で更新
sudo certbot renew

# 更新後、Nginxを再起動
sudo systemctl restart nginx
```

#### 自動更新の確認

```bash
# 自動更新のテスト（実際には更新しない）
sudo certbot renew --dry-run

# エラーが出なければOK
```

---

## システムのメンテナンス

### パッケージの更新

```bash
# パッケージリストを更新
sudo dnf check-update

# 全パッケージを更新
sudo dnf update -y

# 再起動が必要かチェック
sudo needs-restarting -r

# 再起動が必要な場合
sudo reboot
```

再起動後、サービスが自動起動することを確認：

```bash
# SSH再接続後
sudo systemctl status cupid
sudo systemctl status nginx
```

### ディスク使用量の確認

```bash
# ディスク使用量を確認
df -h

# 出力例:
# Filesystem      Size  Used Avail Use% Mounted on
# /dev/xvda1      10G   2.5G  7.5G  25% /
```

ディスク使用量が80%を超えたら、以下を実行：

```bash
# ログのクリーンアップ（1週間より古いものを削除）
sudo journalctl --vacuum-time=7d

# データベースの古いバックアップを削除
find ~/cupid/backups -name "*.backup.*" -mtime +30 -delete
```

### メモリ使用量の確認

```bash
# メモリ使用量を確認
free -h

# 出力例:
#               total        used        free      shared  buff/cache   available
# Mem:          470Mi       180Mi       150Mi       1.0Mi       140Mi       290Mi
```

メモリ使用量が80%を超えたら、サービスを再起動：

```bash
sudo systemctl restart cupid
sudo systemctl restart nginx
```

---

## モニタリング

### 基本的なモニタリング項目

定期的に以下を確認することを推奨、よ。

| 項目 | コマンド | 確認頻度 |
|------|---------|---------|
| サービス状態 | `sudo systemctl status cupid` | 毎日 |
| ログエラー | `sudo journalctl -u cupid -p err --since today` | 毎日 |
| ディスク使用量 | `df -h` | 週1回 |
| メモリ使用量 | `free -h` | 週1回 |
| 証明書有効期限 | `sudo certbot certificates` | 月1回 |
| ユーザー数 | `sqlite3 ~/cupid/cupid.db "SELECT COUNT(*) FROM users;"` | 任意 |

### アラート設定（オプション）

CloudWatch Alarmsを使ってアラートを設定できる、ね（追加コストが発生する）。

---

## トラブルシューティング

### サービスが起動しない

```bash
# 詳細なエラーログを確認
sudo journalctl -u cupid -n 100 --no-pager

# よくある原因:
# 1. 環境変数が読み込めていない → .envファイルの存在と権限を確認
# 2. ポートが使用中 → lsof -i :8080 で確認
# 3. データベースファイルが見つからない → ls -l ~/cupid/cupid.db
```

### Botが応答しない

```bash
# 1. サービスが起動しているか確認
sudo systemctl status cupid

# 2. Nginxが起動しているか確認
sudo systemctl status nginx

# 3. ログでエラーを確認
sudo journalctl -u cupid -f

# 4. Nginxのエラーログを確認
sudo tail -f /var/log/nginx/error.log

# 5. LINE Webhook設定を確認
# LINE Developers Consoleで「Verify」ボタンをクリック
```

### データベースが壊れた

```bash
# 1. バックアップから復元
cp ~/cupid/backups/cupid.db.backup.20250115 ~/cupid/cupid.db

# 2. サービスを再起動
sudo systemctl restart cupid

# 3. データベースの整合性チェック
sqlite3 ~/cupid/cupid.db "PRAGMA integrity_check;"
# 出力: ok
```

### SSL証明書エラー

```bash
# 1. 証明書の状態確認
sudo certbot certificates

# 2. 証明書の更新
sudo certbot renew --force-renewal

# 3. Nginxを再起動
sudo systemctl restart nginx

# 4. ブラウザでHTTPSアクセスを確認
curl -I https://cupid.click/health
```

### メモリ不足

```bash
# 1. メモリ使用量を確認
free -h

# 2. プロセスのメモリ使用量を確認
ps aux --sort=-%mem | head -10

# 3. サービスを再起動
sudo systemctl restart cupid

# 4. 必要に応じてNginxも再起動
sudo systemctl restart nginx
```

---

## セキュリティのベストプラクティス

### 定期的なセキュリティアップデート

```bash
# 毎月1回、セキュリティアップデートを実行
sudo dnf update --security -y

# カーネルアップデート後は再起動
sudo reboot
```

### SSH鍵の管理

```bash
# SSH鍵のパーミッション確認（ローカルマシン）
ls -l ~/Downloads/cupid-bot-key.pem
# 出力: -r-------- 1 user user ... cupid-bot-key.pem

# パーミッションが間違っている場合
chmod 400 ~/Downloads/cupid-bot-key.pem
```

### セキュリティグループの定期確認

AWS Management Consoleで定期的に確認：
- SSH（ポート22）が「マイIP」に制限されているか
- 不要なポートが開いていないか

### 環境変数ファイルの保護

```bash
# .envファイルの権限確認
ls -l ~/cupid/.env
# 出力: -rw------- 1 ec2-user ec2-user ... .env

# パーミッションが間違っている場合
chmod 600 ~/cupid/.env
```

---

## パフォーマンスチューニング

### SQLiteのチューニング

```bash
# WALモードの確認
sqlite3 ~/cupid/cupid.db "PRAGMA journal_mode;"
# 出力: wal

# キャッシュサイズの調整（メモリに余裕がある場合）
sqlite3 ~/cupid/cupid.db "PRAGMA cache_size = 10000;"
```

### Nginxのチューニング

```bash
# Nginx設定ファイルを編集
sudo vim /etc/nginx/nginx.conf
```

以下を調整（t4g.nanoの場合）：

```nginx
worker_processes 2;  # vCPU数に合わせる

events {
    worker_connections 512;  # 小さめに設定
}
```

設定後、Nginxを再起動：

```bash
sudo nginx -t
sudo systemctl restart nginx
```

---

## コスト管理

### 月次コストの確認

AWS Management Consoleの「Billing」で確認：
- EC2: 約$3.07/月
- EBS: 約$0.96/月
- Route 53 Hosted Zone: 約$0.50/月
- Route 53 Domain: 約$0.25/月（年間$3を月割り）
- **Elastic IP（稼働中）**: $0/月
- **Elastic IP（停止中）**: $3.65/月 ⚠️

### Elastic IPの注意

**重要**: EC2を停止すると、Elastic IPに課金される、よ。

長期間使わない場合：
1. EC2を停止
2. Elastic IPを解放（再起動時に新しいIPを取得）
3. Route 53のAレコードを更新

---

## 障害対応

### 障害発生時のチェックリスト

1. **サービスの状態確認**
   ```bash
   sudo systemctl status cupid
   sudo systemctl status nginx
   ```

2. **ログの確認**
   ```bash
   sudo journalctl -u cupid -n 100
   sudo tail -f /var/log/nginx/error.log
   ```

3. **ネットワークの確認**
   ```bash
   curl http://localhost:8080/health  # Goアプリ直接
   curl http://localhost/health       # Nginx経由
   curl https://cupid.click/health    # 外部から
   ```

4. **データベースの確認**
   ```bash
   sqlite3 ~/cupid/cupid.db "PRAGMA integrity_check;"
   ```

5. **ディスク・メモリの確認**
   ```bash
   df -h
   free -h
   ```

### エスカレーション

上記で解決しない場合：
1. EC2を再起動（`sudo reboot`）
2. それでも解決しない場合は、AMIから新しいインスタンスを作成

---

## バックアップとリストア

### 完全バックアップ

```bash
# アプリケーション全体をバックアップ
tar -czf ~/cupid-backup-$(date +%Y%m%d).tar.gz ~/cupid/

# ローカルマシンにダウンロード
scp -i ~/Downloads/cupid-bot-key.pem ec2-user@cupid.click:~/cupid-backup-*.tar.gz ~/Downloads/
```

### リストア

```bash
# バックアップをEC2にアップロード
scp -i ~/Downloads/cupid-bot-key.pem ~/Downloads/cupid-backup-20250115.tar.gz ec2-user@cupid.click:~/

# EC2上で展開
cd ~
tar -xzf cupid-backup-20250115.tar.gz

# サービスを再起動
sudo systemctl restart cupid
```

---

## よくある運用パターン

### 開発時（頻繁にコード変更）

```bash
# コード編集 → ビルド → 再起動 → ログ確認
vim ~/cupid/main.go
go build -o cupid-bot main.go
sudo systemctl restart cupid
sudo journalctl -u cupid -f
```

### 本番運用時（安定稼働）

```bash
# 毎日: ログのエラーチェック
sudo journalctl -u cupid -p err --since today

# 毎週: ディスク・メモリの確認
df -h
free -h

# 毎月: システムアップデート
sudo dnf update -y
sudo reboot
```

---

## ドキュメントのリンク

運用中に参照する主なドキュメント、ね：

- [06_linebot_setup.md](06_linebot_setup.md): LINE Bot設定とトラブルシューティング
- [07_ec2_setup.md](07_ec2_setup.md): EC2の詳細設定
- [03_database_design.md](03_database_design.md): データベース構造の理解

---

⋯⋯安定した運用を、プロデューサー。
