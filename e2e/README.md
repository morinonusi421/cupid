# Backend Integration Tests

このディレクトリには、Cupid LINE Botのバックエンド統合テストが含まれています。

## テストアーキテクチャ

```
Test Code
  ↓ (Send Webhook Event)
WebhookHandler (Real)
  ↓
UserService (Real)
  ↓
Repository (Real)
  ↓
SQLite DB (Real - cupid_test.db)
  ↓
LINE Messaging API (Real - with skip option)
```

**方針:**
- **フロントエンド**: モック（Playwrightテスト不要）
- **バックエンド**: 全層で本物のコンポーネントを使用
- **LINE Server**: 本物のLINE Messaging APIを使用（CI/CDではスキップ可能）

## セットアップ

### 1. 環境変数の設定

プロジェクトルートの`.env`ファイルを使用します。テスト用に別のチャンネルを用意する必要はありません：

- テストDBは`cupid_test.db`（本番DBとは別ファイル）
- LINE APIは本番チャンネルを使用するが、DBが分離されているため問題なし
- まだ開発中で本番ユーザーがいないため、同じチャンネルで安全

既存の`.env`に以下が設定されていることを確認：

```env
LINE_CHANNEL_SECRET=your_channel_secret
LINE_CHANNEL_ACCESS_TOKEN=your_channel_access_token
REGISTER_URL=https://cupid-linebot.click/liff/register.html
```

### 2. テストデータベース

テストは自動的に`cupid_test.db`を使用します。このファイルは：
- `.gitignore`で除外されている（`*.db`パターン）
- 各テスト実行前後に自動クリーンアップされる
- 本番DB（`cupid.db`）には一切影響しない

## テストの実行

### ローカル環境

```bash
make test-integration
```

または：

```bash
go test ./e2e -v
```

### CI/CD環境（LINE API呼び出しをスキップ）

```bash
SKIP_LINE_API=true make test-integration
```

この場合、LINE Messaging APIへの実際の呼び出しはスキップされ、モックが使用されます。

## テストシナリオ

### 1. ユーザー登録フロー
- Webhook経由でフォローイベントを送信
- LIFF登録APIでユーザー情報を送信
- DBに正しく保存されることを確認
- Crush登録URLが正しく返信されることを確認

### 2. 好きな人登録（マッチなし）
- 既存ユーザーがCrush登録APIを呼び出し
- DBに好きな人情報が保存される
- マッチが発生しない（相手がまだ登録していない）

### 3. 相思相愛マッチング
- ユーザーAがユーザーBを好きな人として登録
- ユーザーBがユーザーAを好きな人として登録
- 相思相愛が検出される
- 両方のユーザーにマッチ通知が送信される

### 4. バリデーションエラー
- 漢字・ひらがなを含む名前での登録
- 適切なエラーメッセージが返される
- DBには保存されない

## トラブルシューティング

### テストが失敗する場合

1. **LINE API認証エラー**:
   - `.env`の認証情報が正しいか確認
   - LINEチャンネルが有効か確認

2. **DB関連エラー**:
   - `cupid_test.db`を手動で削除してから再実行
   - マイグレーションが正しく適用されているか確認

3. **署名検証エラー**:
   - `.env`の`LINE_CHANNEL_SECRET`が正しいか確認
   - Webhook署名の生成ロジックが正しいか確認

### CI/CD環境での注意点

CI/CD環境では`SKIP_LINE_API=true`を設定することを推奨します。これにより：
- LINE APIへの実際の呼び出しがスキップされる
- テストが高速化される
- LINE APIのレート制限を気にする必要がなくなる

## ベストプラクティス

1. **テストデータのクリーンアップ**: 各テストは独立して実行可能にする
2. **実際のLINE APIを使う**: ローカル開発では実際のLINE APIを使ってエンドツーエンドの動作を確認
3. **モックとの使い分け**: CI/CDでは`SKIP_LINE_API`を使ってモックに切り替え
4. **テストDBの分離**: `cupid_test.db`を使い、本番DBに影響を与えない

## 参考資料

- [LINE Messaging API Reference](https://developers.line.biz/en/docs/messaging-api/)
- [Go httptest package](https://pkg.go.dev/net/http/httptest)
- [LINE Bot SDK Go v8](https://github.com/line/line-bot-sdk-go)
