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
- **Aレコード**: `cupid-linebot.click` → `13.115.86.124`
- **SSL証明書**: Let's Encrypt（2026-04-12まで有効、自動更新設定済み）

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
  - HTTP (TCP/80) from 0.0.0.0/0
  - HTTPS (TCP/443) from 0.0.0.0/0
- **メモリ**: 916 MiB
- **スワップ**: 1 GB (`/swapfile`、`/etc/fstab`で永続化設定済み)
  - 理由: modernc.org/sqliteのビルド時にメモリ不足を回避
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

## Bot キャラクター設定

### キューピッドちゃん

- **名前**: キューピッドちゃん
- **一人称**: キューピッドちゃん
- **喋り方**:
  - ですます調
  - 「ふえぇ」「はわわ」「あうぅ」などのあざとい昔のアニメキャラみたいな喋り方
  - 絵文字を多用（♡、💕、✨など）
- **性格**:
  - キューピッドの使命（恋人作り）に一生懸命
  - 恋愛ごとにはドキドキキャーキャーみたいな感じ
  - マッチング成立時は特にテンション高め
- **メッセージ例**:
  - 友達追加時: 「わぁぁ！友達追加ありがとうございますっ♡」
  - マッチング成立: 「はわわわわっ！！相思相愛が成立しましたよぉ〜〜♡♡」
  - 登録完了: 「あうぅ...登録完了ですっ♡」

### メッセージ実装の注意点

1. **ユーザー向けメッセージのみキャラ付け**
   - LINE送信メッセージ: キューピッドちゃんキャラで
   - エラーメッセージ（技術的なもの）: そのまま（キャラ付けしない）
   - ログメッセージ: そのまま

2. **メッセージは定数化**
   - `internal/message/` ディレクトリを作成
   - `messages.go` でメッセージを const として管理
   - URL埋め込みが必要な場合は関数として実装

3. **一貫性の維持**
   - 全てのユーザー向けメッセージでキャラクターを統一
   - テンションの高低はシーンに応じて調整（マッチング > 登録 > 通常）

## LINEミニアプリ情報

**【重要】現在は本番検証段階のため、本番用LIFF IDを使用すること**
- `.env`の`LINE_LIFF_USER_CHANNEL_ID`と`LINE_LIFF_CRUSH_CHANNEL_ID`は本番用を設定
- `static/user/register.js`と`static/crush/register.js`のLIFF_IDも本番用を使用
- **リッチメニューのURLも本番用を設定すること**（LINE公式アカウント管理画面から）

### チャネルID
- **開発用**: `2009059074`
- **審査用**: `2009059075`
- **本番用**: `2009059076`

### LIFF URL（ユーザー登録用）
- **開発用**: `https://miniapp.line.me/2009059074-aX6pc41R`
- **審査用**: `https://miniapp.line.me/2009059075-2bCpQry4`
- **本番用**: `https://miniapp.line.me/2009059076-kBsUXYIC`

### LIFF ID
- **開発用**: `2009059074-aX6pc41R`
- **審査用**: `2009059075-2bCpQry4`
- **本番用**: `2009059076-kBsUXYIC`

### エンドポイントURL（ユーザー登録用）
- **開発用**: `https://cupid-linebot.click/user/register.html`
- **審査用**: （自動反映）
- **本番用**: （自動反映）

### チャネルID（好きな人登録用）
- **開発用**: `2009070889`
- **審査用**: `2009070890`
- **本番用**: `2009070891`

### LIFF URL（好きな人登録用）
- **開発用**: `https://miniapp.line.me/2009070889-qZo1cdq6`
- **審査用**: `https://miniapp.line.me/2009070890-jtxmk3U1`
- **本番用**: `https://miniapp.line.me/2009070891-iIdvFKtI`

### LIFF ID（好きな人登録用）
- **開発用**: `2009070889-qZo1cdq6`
- **審査用**: `2009070890-jtxmk3U1`
- **本番用**: `2009070891-iIdvFKtI`

### エンドポイントURL（好きな人登録用）
- **開発用**: `https://cupid-linebot.click/crush/register.html`
- **審査用**: （自動反映）
- **本番用**: （自動反映）

### リッチメニュー設定

LINE公式アカウントの管理画面から設定。**必ず本番用URLを使用すること。**

**本番用URL（現在の設定）:**
- 自分の情報を登録/編集: `https://miniapp.line.me/2009059076-kBsUXYIC`
- 想い人の情報を登録/編集: `https://miniapp.line.me/2009070891-iIdvFKtI`

**開発用URL（参考）:**
- 自分の情報を登録/編集: `https://miniapp.line.me/2009059074-aX6pc41R`
- 想い人の情報を登録/編集: `https://miniapp.line.me/2009070889-qZo1cdq6`

**注意:**
- リッチメニューのURLが開発用になっていると、サーバー側の認証が失敗する
- 「LINE認証に失敗しました」エラーが出る場合は、リッチメニューのURLを確認

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

### Nginx
- **設定ファイル**: `nginx/cupid.conf`（Git管理）
- **EC2配置**: `/etc/nginx/conf.d/cupid.conf`へシンボリックリンク（`sudo ln -s ~/cupid/nginx/cupid.conf /etc/nginx/conf.d/cupid.conf`）
- **設定変更後**: `git pull` → `sudo nginx -t` → `sudo systemctl reload nginx`

### systemd（Goサーバー）
- **サービス名**: `cupid.service`
- **サービスファイル**: `systemd/cupid.service`（Git管理）
- **EC2配置**: `/etc/systemd/system/cupid.service`へシンボリックリンク
- **実行ファイル**: `~/cupid/cupid`（`go build -o cupid`でビルド）

## 参考資料

### LINE Bot開発
- **LINE Bot SDK Go v8 (ローカルにcloneしてきた)**: `/Users/takahashi.hikaru/line-bot-sdk-go/`
- **LINE Messaging API Reference**: https://developers.line.biz/en/docs/messaging-api/
- **Webhook Events**: https://developers.line.biz/en/docs/messaging-api/receiving-messages/

### SQLBoiler
- **設定ファイル**: `sqlboiler.toml`
- **自動生成ディレクトリ**: `entities/`
- **コマンド**: `make generate` でentitiesを再生成（`--no-auto-timestamps`フラグ付き）
- **理由**: SQLite で created_at/updated_at を TEXT 型として保存しているため、time.Time として自動処理されると型エラーが発生する
- **参考**: https://zenn.dev/da1chi/articles/806fa57b4eff3c
- **テストについて**:
  - sqlboilerが自動生成する `entities/` 配下のテストは実行しない
  - 理由: sqlboiler-sqlite3のfKeyDestroyer正規表現にバグがあり、FOREIGN KEY削除時に構文エラーが発生する
  - テスト実行は `make test` を使用（entitiesディレクトリを自動的に除外）

### Mockery (Mockライブラリ)
- **ツール**: github.com/vektra/mockery/v2
- **インストール**: `go install github.com/vektra/mockery/v2@latest`
- **設定ファイル**: `.mockery.yaml`
- **自動生成ディレクトリ**: `internal/service/mocks/`, `internal/repository/mocks/`
- **コマンド**: `make mocks` でmockを再生成
- **特徴**:
  - testify/mockと完全に統合
  - interfaceから自動でmock生成
  - 既存の手動mockと互換性あり
  - sqlboilerと同じパターンで管理
- **生成されるmock**:
  - `UserService` → `internal/service/mocks/MockUserService.go`
  - `MatchingService` → `internal/service/mocks/MockMatchingService.go`
  - `UserRepository` → `internal/repository/mocks/MockUserRepository.go`
- **使い方**:
  - テストで `servicemocks.NewMockUserService(t)` を使用
  - On/Return でモック動作を設定
  - AssertExpectations で検証
- **参考**: https://vektra.github.io/mockery/

### Database Schema
- **スキーマファイル**: `db/schema.sql`
- **自動作成**: アプリケーション起動時に、テーブルが存在しない場合は自動的にスキーマを作成
- **テスト**: `pkg/testutil/SetupTestDB` を使用してテスト用DBをセットアップ
  - e2e/integration_test.go: `testutil.SetupTestDB(t, testDBFile, "../db/schema.sql")`
  - internal/repository/user_repo_test.go: `testutil.SetupTestDB(t, "test_repo_cupid.db", "../../db/schema.sql")`
- **スキーマ変更**:
  - 開発初期段階のため、`db/schema.sql`を直接編集してOK
  - 本番運用後は必要に応じてマイグレーションツールの導入を検討

## 使用するgitアカウント

- このリポジトリでは**morinonusi421**アカウントを使用
- 他のリポジトリ（仕事用など）とは別のGitアカウント
- SSH設定で自動的に鍵を切り替え

## 開発方針

### 破壊的変更について
- **現在は開発中のため、破壊的変更を気にせず進める**
- スキーマ変更やAPI変更など、後方互換性を気にせずリファクタリング可能
- 本番リリース時に改めて見直し・調整を行う

### 環境変数（.env）の更新について
- **ローカルで`.env`を更新した場合、EC2側も必ず更新すること**
- 更新手順:
  1. ローカルで`.env`を編集
  2. EC2にSSHして、同じ内容をEC2の`~/cupid/.env`にも反映
  3. `sudo systemctl restart cupid`でサーバー再起動

### テストの作成
・mockeryを活用
・各serviceやレイヤーでの責任分離を考える。(一つの実装ファイルだけをみてテストを作らず、周辺テストも考えるべき)
・必要に応じてテーブルドリブンを意識

## 次のステップ

- [x] Phase 0: 環境準備（ローカル）
- [x] Phase 1: ドメイン取得（cupid-linebot.click）
- [x] Phase 2: EC2基本セットアップ
- [x] Phase 3: Hello World (HTTP)
- [x] Phase 4: Nginx + リバースプロキシ
- [x] Phase 5: HTTPS化 + systemdサービス化
- [x] Phase 6: LINE Bot基本応答（オウム返し）
- [ ] Phase 7: ユーザー登録フローとDB実装
