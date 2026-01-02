# Cupid LINE Bot - プロジェクト情報

このファイルは、Cupid LINE Botプロジェクト専用のClaude Code設定ファイル。

## プロジェクト基本情報

- **プロジェクト名**: Cupid LINE Bot
- **説明**: 相思相愛を見つけるLINE Bot
- **ドメイン**: `cupid-linebot.click` (AWS Route 53で取得済み)
- **リポジトリ**: https://github.com/morinonusi421/cupid

## インフラ情報

### ドメイン
- **ドメイン名**: `cupid-linebot.click`
- **取得場所**: AWS Route 53

### AWS設定
- **アカウントID**: 838890403187
- **リージョン**: ap-northeast-1 (東京)
- **IAMユーザー**: HikaruTakahashi (AdministratorAccess)
- **AWS CLIプロファイル**: `personal` (direnvで自動切り替え設定済み)

### EC2インスタンス
- **名前**: cupid-bot
- **インスタンスID**: i-00f240fa944612ee5
- **タイプ**: t4g.micro (ARM64, 無料枠対象)
- **AMI**: Amazon Linux 2023.10.20260105 (アップグレード済み)
- **状態**: running
- **タイムゾーン**: Asia/Tokyo (JST)
- **パブリックIP**: 13.115.86.124 (Elastic IP)
- **プライベートIP**: 172.31.10.53
- **パブリックDNS**: ec2-13-115-86-124.ap-northeast-1.compute.amazonaws.com
- **キーペア**: cupid-bot-key (`~/.ssh/cupid-bot-key.pem`)
- **SSH接続**: `ssh cupid-bot` (`~/.ssh/config`設定済み)
- **ネットワークインターフェイスID**: eni-06c20b8172f868ca4
- **セキュリティグループID**: sg-0c8acd95cc0b039a8
  - SSH (TCP/22) from 0.0.0.0/0
- **ストレージ**: 10 GB gp3 (3000 IOPS)
- **起動日**: 2026-01-02

### Elastic IP
- **IPアドレス**: 13.115.86.124
- **割り当てID**: eipalloc-0715a5692ab6fe774
- **関連付け済み**: i-00f240fa944612ee5 (cupid-bot)

## LINE Bot情報

- **Channel ID**: `2008809168`
- **Channel Secret**: `.env`
- **Channel Access Token**: `.env`

## 開発環境

### Go
- **バージョン管理**: goenv
- **指定バージョン**: 1.25.5 (`.go-version`で管理)
- **EC2インストール済み**: Go 1.25.5 linux/arm64 (`/usr/local/go/bin/go`)

### Git

#### ローカル（Mac）
- **アカウント**: morinonusi421
- **SSH Host**: `github.com-morinonusi`
- **SSH鍵**: `~/.ssh/id_ed25519`
- **リモートURL**: `git@github.com-morinonusi:morinonusi421/cupid.git`

#### EC2
- **SSH鍵**: EC2専用の鍵を作成済み（`~/.ssh/id_ed25519`）
- **GitHub登録**: `cupid-bot-ec2`として登録済み
- **SSH Config**: `~/.ssh/config`に`github.com-morinonusi`設定済み
- **リポジトリ**: `~/cupid`にclone済み

### AWS CLI
- **プロファイル**: personal
- **設定場所**: `~/.aws/credentials`, `~/.aws/config`
- **自動切り替え**: `.envrc`でdirenv使用（cupidディレクトリ内で自動的にpersonalプロファイルを使用）

## 重要な注意事項

- このリポジトリでは**morinonusi421**アカウントを使用
- 他のリポジトリ（仕事用など）とは別のGitアカウント
- SSH設定で自動的に鍵を切り替え

## 次のステップ

- [x] Phase 0: 環境準備（ローカル）
- [x] Phase 1: ドメイン取得（cupid-linebot.click）
- [x] Phase 2: EC2基本セットアップ完了
  - [x] EC2インスタンス作成
  - [x] Elastic IP割り当て
  - [x] SSH接続確認
  - [x] OSアップグレード（2023.10）
  - [x] タイムゾーン設定（JST）
  - [x] 基本ツール確認
- [ ] Phase 3: Hello World (HTTP)（進行中）
  - [x] Goインストール（ARM64）- 1.25.5
  - [x] EC2専用SSH鍵作成とGitHub登録
  - [x] EC2でリポジトリclone
  - [ ] HTTPサーバー作成（ローカルで開発→EC2にpull）
  - [ ] 動作確認
- [ ] Phase 4: Nginx + リバースプロキシ
- [ ] Phase 5: HTTPS化
- [ ] Phase 6: LINE Bot基本応答
