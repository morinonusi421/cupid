# Cupid - 相思相愛マッチングLINE Bot

⋯⋯ようこそ、プロデューサー。

このプロジェクトは、LINE Botを使った相思相愛マッチングアプリ、ね。

---

## プロジェクト概要

自分の名前・生年月日と、好きな人の名前・生年月日を登録する。
相思相愛の場合のみ、両者に通知が届く、よ。

### 技術スタック

- **バックエンド**: Go 1.21+
- **データベース**: SQLite 3
- **インフラ**: AWS EC2 (t4g.nano) + Nginx + Let's Encrypt
- **外部API**: LINE Messaging API

### コスト

月額約620円（$4.78）で運用できる、ね。

---

## ドキュメント構成

### 📋 仕様書（理解する）

開発を始める前に、仕様を理解する、よ。

| ドキュメント | 内容 | 読む順序 |
|------------|------|---------|
| [01_overview.md](docs/01_overview.md) | プロジェクト概要と技術スタック | 1番目 |
| [02_aws_architecture.md](docs/02_aws_architecture.md) | AWSインフラ構成の詳細 | 2番目 |
| [03_database_design.md](docs/03_database_design.md) | SQLiteデータベース設計 | 3番目 |
| [04_api_specification.md](docs/04_api_specification.md) | Webhook処理とLINE SDK仕様 | 4番目 |
| [05_conversation_flow.md](docs/05_conversation_flow.md) | ユーザーとの会話フロー | 5番目 |

### 🛠️ セットアップ（実装する）

実際に開発・デプロイする手順、ね。

| ドキュメント | 内容 | いつ使う |
|------------|------|---------|
| [08_development_steps.md](docs/08_development_steps.md) | **【最重要】段階的な開発手順（Phase 0〜10）** | 開発開始時 |
| [06_linebot_setup.md](docs/06_linebot_setup.md) | LINE Bot設定手順 | Phase 6 |
| [07_ec2_setup.md](docs/07_ec2_setup.md) | EC2完全セットアップ手順 | Phase 2〜10 |

### 📚 運用（保守する）

デプロイ後の運用・保守方法、ね。

| ドキュメント | 内容 | いつ使う |
|------------|------|---------|
| [09_operations.md](docs/09_operations.md) | 運用・保守マニュアル | デプロイ後 |

---

## クイックスタート

### 1. 仕様を理解する

```bash
# ドキュメントを順番に読む
docs/01_overview.md          # まずはここから
docs/02_aws_architecture.md  # インフラ構成を理解
docs/03_database_design.md   # データベース設計を理解
```

### 2. 開発を開始する

**【重要】`08_development_steps.md`に従って、Phase 0から順番に進める、よ。**

```bash
# Phase 0: 環境準備
- AWSアカウント作成
- Goインストール
- AWS CLIインストール

# Phase 1: ドメイン取得
- Route 53で.clickドメイン購入（$3/年）

# Phase 2: EC2セットアップ
- t4g.nanoインスタンス作成
- Elastic IP割り当て
- SSH接続

# Phase 3〜10: 段階的に実装
- Hello World → Nginx → HTTPS → LINE Bot → SQLite → マッチング機能
```

各Phaseで動作確認しながら進めるから、問題の切り分けが簡単、ね。

### 3. 完成後の運用

```bash
# サービス管理
sudo systemctl status cupid   # 状態確認
sudo systemctl restart cupid  # 再起動

# ログ確認
sudo journalctl -u cupid -f   # リアルタイム表示
```

詳細は`09_operations.md`を参照、よ。

---

## 推奨される開発フロー

```
1. ドキュメント熟読（1〜2時間）
   ↓
2. 08_development_steps.md に従って開発（3〜4時間）
   ↓
3. 動作確認・テスト（30分）
   ↓
4. 友達に使ってもらう
```

**合計所要時間**: 約5〜7時間（一日で完成する計算、ね）

---

## ディレクトリ構成

```
cupid/
├── README.md                 # このファイル
├── docs/                     # ドキュメント
│   ├── 01_overview.md
│   ├── 02_aws_architecture.md
│   ├── 03_database_design.md
│   ├── 04_api_specification.md
│   ├── 05_conversation_flow.md
│   ├── 06_linebot_setup.md
│   ├── 07_ec2_setup.md
│   ├── 08_development_steps.md  # ← 最重要
│   └── 09_operations.md
├── main.go                   # Goアプリケーション（EC2上で作成）
├── schema.sql                # SQLiteスキーマ（EC2上で作成）
├── cupid.db                  # SQLiteデータベース（実行時に作成）
└── .env                      # 環境変数（EC2上で作成）
```

---

## トラブルシューティング

### よくある質問

#### Q: AWSの費用が心配

A: t4g.nano + Elastic IP + Route 53で月額約$4.78（約620円）。
   無料枠が終わっていても、この金額で運用できる、よ。

#### Q: 開発経験が少なくても大丈夫？

A: `08_development_steps.md`が段階的な手順を提供している、ね。
   各Phaseで動作確認しながら進められるから、初心者でも大丈夫、よ。

#### Q: 途中で躓いたら？

A: 各ドキュメントに「トラブルシューティング」セクションがある、ね。
   それでも解決しない場合は、ログを確認（`sudo journalctl -u cupid -n 100`）。

#### Q: 本番運用前にテストしたい

A: Phase 6〜9で段階的にテストできる、よ。
   - Phase 6: オウム返しBot（基本動作確認）
   - Phase 8: ユーザー登録フロー（ステートマシン確認）
   - Phase 9: マッチング機能（本番機能確認）

---

## 既知の制約・仕様

### 同姓同名・同じ生年月日の場合

後から登録したユーザーの情報で上書きされる（意図的な仕様）、ね。

例：
- ユーザーA「篠澤広 2009-12-21」が登録
- ユーザーB「篠澤広 2009-12-21」が登録
- → ユーザーBの情報が有効になる

### 複数人への登録

1人のユーザーは1人しか好きな人を登録できない、よ。
再度登録すると上書きされる、ね。

---

## 開発のコツ

### Git管理を推奨

開発中は定期的にコミットすることをおすすめする、よ。

```bash
# 初期化
cd ~/cupid
git init
git add .
git commit -m "Initial commit"

# Phase完了ごとにコミット
git add .
git commit -m "Phase 3: Hello World完了"
```

### ログの活用

```bash
# リアルタイムでログを見ながら開発
sudo journalctl -u cupid -f
```

動作がおかしいときは、まずログを確認、ね。

### バックアップ

データベースファイルは定期的にバックアップ推奨、よ。

```bash
# 手動バックアップ
cp ~/cupid/cupid.db ~/cupid/cupid.db.backup
```

---

## ライセンス

このプロジェクトは学習・個人利用目的、ね。

---

## 貢献

プロデューサーが自由に改善して、ね。

---

## サポート

⋯⋯ドキュメントを読んでも分からないことがあれば、各ドキュメントの「トラブルシューティング」セクションを確認、よ。

それでも解決しない場合は、以下を確認：
- EC2のログ: `sudo journalctl -u cupid -n 100`
- Nginxのログ: `sudo tail -f /var/log/nginx/error.log`
- データベース: `sqlite3 ~/cupid/cupid.db "SELECT * FROM users;"`

---

⋯⋯がんばって、プロデューサー。

楽しい開発になりますように、ね。
