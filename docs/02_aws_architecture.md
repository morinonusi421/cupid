# AWS アーキテクチャ

## システム構成図

```
LINE App (ユーザー)
  ↓
LINE Platform
  ↓ HTTPS Webhook
Route 53 (cupid.click)
  ↓
Elastic IP
  ↓
EC2 t4g.nano (Public Subnet)
├── Nginx (ポート443, SSL終端)
│   └── Let's Encrypt証明書
├── Go アプリケーション (localhost:8080)
└── SQLite (cupid.db ファイル)

EC2 → LINE Platform (Push Message API)
```

## 採用したパターン

**EC2 + SQLite + Nginx + Let's Encrypt**

### 選定理由

#### 1. コスト最適化
```
月額: 約$4.78（約620円）
- EC2 t4g.nano: $3.07
- EBS gp3 10GB: $0.96
- Route 53ドメイン: $0.25
- Route 53 Hosted Zone: $0.50

予算2,000円に対して、1/3以下
```

#### 2. シンプルな構成
```
- EC2内で完結（MySQL不要）
- VPC設計が簡単
- トラブルシューティングが容易
- SSH接続で直接操作可能
```

#### 3. 今回の規模に最適
```
想定負荷:
- 同時アクセス: 1-2人
- 1日のリクエスト: 100回程度
- データ量: 数MB〜数十MB

→ t4g.nano + SQLiteで十分すぎる
```

#### 4. 学習効果
```
- EC2管理（起動・停止・監視）
- Nginx設定（リバースプロキシ、SSL）
- Let's Encrypt証明書管理
- SQLite運用
- systemdでのプロセス管理
- Route 53でのDNS管理
```

#### 5. 拡張性の確保
```
将来的にユーザーが増えたら:
1. t4g.micro / t4g.smallにスケールアップ
2. SQLite → MySQLに移行
3. その時点で予算見直し

データ移行も容易（SQLのエクスポート・インポート）
```

---

## 他の候補との比較

| 構成 | 月額 | メリット | デメリット | 判定 |
|-----|------|---------|----------|------|
| **EC2 + SQLite** | **$5** | シンプル、安い、学習効果高 | スケールアップ時に移行必要 | ⭐ 採用 |
| EC2 + RDS | $20 | 拡張性高、可用性高 | 高額、VPC複雑 | ✗ 予算オーバー |
| Lambda + DynamoDB | $1 | 最安、サーバーレス | JOIN不可、拡張性低 | △ 将来性なし |
| Lambda + RDS | $15 | 拡張性高 | VPC複雑、NAT高額 | ✗ 複雑すぎ |

---

## 各コンポーネントの詳細

### EC2 (Elastic Compute Cloud)

#### インスタンス仕様
```
タイプ: t4g.nano
アーキテクチャ: ARM64 (Graviton2)
vCPU: 2
RAM: 0.5GB (512MB)
ネットワーク: 最大5Gbps
```

#### インスタンスタイプ選定理由
```
t4g.nano を選択:
- ARM64で最もコスト効率が良い
- 月$3.07（x86のt3.nanoより安い）
- 軽量アプリには十分な性能
- SQLiteはサーバープロセス不要なので0.5GB RAMでも余裕
```

#### メモリ使用量見積もり
```
OS (Amazon Linux 2023): 100-120MB
Nginx: 5-10MB
Go アプリケーション: 30-50MB
SQLite: 5-10MB（Goプロセス内、サーバープロセス不要）

合計: 140-190MB
空き: 320-370MB (余裕率 63-72%)
```

#### OS選択
```
Amazon Linux 2023 (ARM64)

理由:
- AWS最適化（パフォーマンス向上）
- セキュリティアップデート自動
- 2028年までサポート
- yum (dnf)でパッケージ管理が簡単
- ARM64対応
```

---

### EBS (Elastic Block Store)

#### ボリューム仕様
```
タイプ: gp3 (汎用SSD)
容量: 10GB
IOPS: 3,000（デフォルト）
スループット: 125MB/s（デフォルト）
```

#### 容量内訳
```
OS + システム: 2GB
Go アプリケーション: 0.1GB
SQLiteデータベース: 0.5GB（初期数MB、余裕持たせて）
ログファイル: 0.5GB
余裕: 6.9GB

→ 将来的な拡張にも対応可能
```

#### gp3選定理由
```
- gp2より安い（$0.096/GB vs $0.12/GB）
- 高性能（3,000 IOPS標準）
- 今回の用途には十分すぎる
```

---

### Nginx

#### 役割
```
1. リバースプロキシ
   - 外部(443) → 内部(8080)にプロキシ

2. SSL終端
   - HTTPSリクエストを受けてSSL復号
   - バックエンド(Go)にはHTTPで転送

3. 静的ファイル配信（将来的に）
   - 今回は不要だが、拡張時に利用可能
```

#### 設定例
```nginx
server {
    listen 80;
    server_name cupid.click;

    # HTTPからHTTPSへリダイレクト
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name cupid.click;

    # Let's Encrypt証明書（Certbotが自動設定）
    ssl_certificate /etc/letsencrypt/live/cupid.click/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cupid.click/privkey.pem;

    # セキュリティヘッダー
    add_header Strict-Transport-Security "max-age=31536000" always;

    # リバースプロキシ設定
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

### Let's Encrypt + Certbot

#### Let's Encryptとは
```
無料のSSL証明書発行サービス
- 証明書: 完全無料
- 有効期限: 90日
- 自動更新: Certbotが対応
```

#### Certbotの役割
```
1. 証明書の自動取得
   - ドメイン所有確認
   - Let's Encryptから証明書取得

2. Nginx設定の自動書き換え
   - SSL設定を自動追加

3. 証明書の自動更新
   - 期限30日前に自動更新
   - cronで定期実行
```

#### セットアップ手順（概要）
```bash
# Certbotインストール
sudo dnf install certbot python3-certbot-nginx -y

# 証明書取得 + Nginx自動設定
sudo certbot --nginx -d cupid.click

# 自動更新タイマー有効化
sudo systemctl enable certbot-renew.timer
```

---

### SQLite

#### 選定理由
```
1. サーバープロセス不要
   - メモリ使用量が最小
   - 管理が不要

2. ファイルベース
   - データはcupid.dbファイル1個
   - バックアップ: ファイルコピーだけ

3. シンプル
   - インストール不要（Goドライバに組み込み）
   - ポート管理不要
   - ユーザー・パスワード不要

4. 性能十分
   - 同時アクセス1-2人には十分すぎる
   - 秒間数百クエリ処理可能

5. 後でMySQLに移行可能
   - データエクスポート・インポート簡単
   - Go の database/sql は同じインターフェース
```

#### データベースファイル配置
```
/home/ec2-user/cupid/cupid.db

権限:
- 所有者: ec2-user
- パーミッション: 644
```

---

### Route 53

#### 役割
```
1. ドメイン登録
   - cupid.click ドメイン取得

2. DNS管理
   - Aレコード: cupid.click → Elastic IP
```

#### 設定
```
Hosted Zone: cupid.click
レコード:
  - Type: A
  - Name: cupid.click
  - Value: <Elastic IP>
  - TTL: 300
```

#### コスト
```
ドメイン登録(.click): 年$3 (月$0.25)
Hosted Zone: 月$0.50
合計: 月$0.75
```

---

### Elastic IP

#### 役割
```
EC2インスタンスに固定IPアドレスを割り当て
- インスタンス再起動してもIP変わらない
- Route 53のAレコードで参照
```

#### コスト
```
使用中: 無料
未使用: 月$3.6

→ 必ずEC2にアタッチすること
```

---

## ネットワーク構成

### VPC設定
```
VPC: デフォルトVPC使用（新規作成不要）
Subnet: Public Subnet（自動割り当て）
Internet Gateway: デフォルトVPCに既存
```

### セキュリティグループ
```
名前: cupid-sg

インバウンドルール:
1. SSH (22)
   - プロトコル: TCP
   - ソース: 自宅IP（例: 123.456.789.0/32）
   - 用途: サーバー管理

2. HTTP (80)
   - プロトコル: TCP
   - ソース: 0.0.0.0/0（全て許可）
   - 用途: Let's Encryptドメイン認証

3. HTTPS (443)
   - プロトコル: TCP
   - ソース: 0.0.0.0/0（全て許可）
   - 用途: LINE Webhook

アウトバウンドルール:
- すべて許可（デフォルト）
```

---

## デプロイ戦略

### 初回セットアップフロー
```
1. Route 53でドメイン取得
2. EC2インスタンス起動
3. Elastic IP割り当て
4. Route 53でAレコード設定
5. セキュリティグループ設定
6. SSH接続
7. 環境構築（Go、Nginx、Certbot）
8. Let's Encrypt証明書取得
9. アプリケーションデプロイ
10. systemdでサービス化
```

### デプロイ方法
```
手動デプロイ（初回）:
- SSH接続してgit pull
- go buildで再ビルド
- systemctl restartで再起動

将来的（オプション）:
- GitHub Actionsで自動デプロイ
- Terraformでインフラコード化
```

---

## コスト詳細

### 月額コスト（東京リージョン）

| サービス | スペック | 月額 |
|---------|---------|------|
| EC2 | t4g.nano（常時稼働） | $3.07 |
| EBS | gp3 10GB | $0.96 |
| Elastic IP | EC2にアタッチ | $0 |
| Route 53 | .click ドメイン | $0.25 |
| Route 53 | Hosted Zone | $0.50 |
| データ転送 | 無料枠内 | $0 |
| **合計** | | **$4.78 (約620円)** |

### 無料枠について
```
AWS無料枠（12ヶ月）は既に終了済み
→ 初月から課金

ただし、今回の構成は無料枠なしでも月$5以下
```

---

## 監視・運用

### CloudWatch基本モニタリング（無料）
```
自動取得される指標（5分間隔）:
- CPU使用率
- ネットワークIN/OUT
- ディスクRead/Write

→ 追加料金なし
```

### ログ管理
```
Goアプリケーションログ:
/var/log/cupid/app.log

Nginxログ:
/var/log/nginx/access.log
/var/log/nginx/error.log

ログローテーション:
- logrotateで自動設定
- 7日間保持
```

### アラート設定（オプション）
```
CloudWatch Alarm:
- CPU使用率 > 80% でSNS通知
- 無料枠内で設定可能
```

---

## バックアップ戦略

### バックアップなし（コスト削減）
```
理由:
- 友達しか使わない
- 最悪データ消失しても再登録可能
- コスト削減優先

リスク:
- EC2インスタンス終了 → データ消失
- EBS障害 → データ消失
```

### 最小限の対策
```
重要操作前に手動バックアップ:
cp cupid.db cupid.db.backup
```

### 将来的なバックアップ案（オプション）
```
方法1: EBSスナップショット
- 週1回手動実行
- コスト: 月$0.5程度

方法2: SQLiteダンプ → S3
- cronで毎日実行
- コスト: 月$0.01程度
```

---

## スケーリング戦略

### 短期的（現状維持）
```
t4g.nano + SQLite
→ 同時アクセス10人程度まで対応可能
```

### 中期的（ユーザー増加時）
```
1. インスタンスタイプ変更
   t4g.nano → t4g.micro (月+$3)

2. EBS容量拡張（必要に応じて）
   10GB → 20GB (月+$1)
```

### 長期的（大規模化）
```
1. データベース移行
   SQLite → MySQL (EC2内)
   または
   SQLite → RDS (月+$15)

2. ロードバランサー追加
   ALB + 複数EC2インスタンス

3. 予算再検討
```

---

## セキュリティ

### SSL/TLS
```
Let's Encrypt (TLS 1.2以上)
- 証明書: RSA 2048bit
- 自動更新で常に最新
```

### SSH接続
```
- 公開鍵認証のみ（パスワード認証無効）
- 接続元IP制限（セキュリティグループ）
- ポート変更（オプション、22 → 他のポート）
```

### アプリケーション
```
- LINE署名検証（必須）
- SQLインジェクション対策（プレースホルダー使用）
- 環境変数で秘密情報管理
```

### OS
```
- 自動セキュリティアップデート有効化
- 不要なサービス停止
- ファイアウォール設定（firewalld）
```

---

## トラブルシューティング

### よくある問題と対処

#### 1. Webhookが届かない
```
確認項目:
- セキュリティグループでHTTPS(443)が開いているか
- Nginxが起動しているか（systemctl status nginx）
- SSL証明書が有効か（certbot certificates）
- Goアプリが起動しているか（systemctl status cupid）
```

#### 2. メモリ不足
```
対処:
- free -h でメモリ確認
- プロセス確認（top, htop）
- ログ確認（/var/log/messages）
- ダメなら t4g.micro に変更
```

#### 3. ディスク容量不足
```
対処:
- df -h で確認
- ログローテーション確認
- 古いログ削除
- EBS拡張（必要なら）
```

#### 4. SSL証明書エラー
```
対処:
- certbot certificates で確認
- 手動更新: sudo certbot renew
- Nginx再起動: sudo systemctl restart nginx
```

---

## まとめ

### 最終構成
```
EC2 t4g.nano + Amazon Linux 2023
+ Nginx + Let's Encrypt
+ Go アプリケーション
+ SQLite

月額: $4.78（約620円）
```

### メリット
- コスト最適化（予算の1/3）
- シンプルな構成（EC2内で完結）
- 学習効果が高い
- 今回の規模に最適
- 将来の拡張も可能

### 次のステップ
詳細なセットアップ手順は `06_linebot_setup.md` および別途作成予定の `07_ec2_setup.md` を参照。
