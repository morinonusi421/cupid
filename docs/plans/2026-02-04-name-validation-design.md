# 名前バリデーション設計書

**作成日**: 2026-02-04
**目的**: 名前登録（自分・好きな人）の表記ブレを防ぐため、カタカナフルネームのみを許可する

---

## 概要

### 背景

現在の実装では名前の入力形式に制約がなく、以下の問題が発生している：
- 表記ブレ（漢字/ひらがな/カタカナ混在）
- スペースの有無による不一致
- 全角/半角の混在

これらの問題により、同一人物でもマッチングできないケースが発生する可能性がある。

### 解決策

**カタカナフルネーム（スペースなし）のみを許可する**ことで、表記の統一を図る。

---

## 要件定義

### 決定事項

| 項目 | 内容 |
|------|------|
| **対象** | ユーザー名、好きな人の名前（両方） |
| **形式** | 全角カタカナのみ（ァ-ヴー）、スペース不可 |
| **文字数** | 2〜20文字 |
| **実装箇所** | フロントエンド + バックエンド両方 |
| **既存データ** | 全削除（`make db-reset`） |
| **バリデーションタイミング** | リアルタイム（`blur`イベント）+ Submit時 |
| **ライブラリ** | なし（Plain JavaScript） |

### 正規表現

```regex
^[ァ-ヴー]{2,20}$
```

**説明:**
- `^`: 文字列の開始
- `[ァ-ヴー]`: 全角カタカナ（ァからヴまで、長音記号ー含む）
- `{2,20}`: 2文字以上20文字以下
- `$`: 文字列の終了

### エラーメッセージ

**フロントエンド（ユーザー向け）:**
```
名前はカタカナフルネームで入力してください（例: ヤマダタロウ）
```

**バックエンド（API応答）:**
```
名前は全角カタカナ2〜20文字で入力してください（スペース不可）
```

---

## 設計詳細

### 1. Backend Design - Model層

#### `internal/model/user.go`に追加

```go
import "regexp"

// IsValidName は名前が有効なカタカナフルネームかをチェックする
// 戻り値: (valid bool, errorMessage string)
func IsValidName(name string) (bool, string) {
	// 長さチェック
	runeCount := len([]rune(name))
	if runeCount < 2 || runeCount > 20 {
		return false, "名前は2〜20文字で入力してください"
	}

	// カタカナのみチェック
	matched, _ := regexp.MatchString(`^[ァ-ヴー]+$`, name)
	if !matched {
		return false, "名前は全角カタカナ2〜20文字で入力してください（スペース不可）"
	}

	return true, ""
}
```

**設計ポイント:**
- **Static function**: User構造体のメソッドではなく、パッケージレベルの関数
- **戻り値**: `(bool, string)`で有効性とエラーメッセージを返す
- **文字数カウント**: `len([]rune(name))`で正確にカウント（バイト数ではなく文字数）

#### Service層での使用

`internal/service/user_service.go`の`RegisterFromLIFF`で使用：

```go
func (s *userService) RegisterFromLIFF(ctx context.Context, userID, name, birthday string) error {
	// バリデーション
	valid, errMsg := model.IsValidName(name)
	if !valid {
		return fmt.Errorf("invalid name: %s", errMsg)
	}

	// 既存の処理...
}
```

`RegisterCrush`で使用：

```go
func (s *userService) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string) (matched bool, matchedUserName string, err error) {
	// ...既存の処理...

	// Crush名のバリデーション
	valid, errMsg := model.IsValidName(crushName)
	if !valid {
		return false, "", fmt.Errorf("invalid crush name: %s", errMsg)
	}

	// ...既存の処理...
}
```

---

### 2. Frontend Design - HTML + JavaScript

#### HTML変更

**`static/liff/register.html`:**

```html
<div class="form-group">
    <label for="name">お名前（カタカナフルネーム）</label>
    <input
        type="text"
        id="name"
        placeholder="例: ヤマダタロウ"
        maxlength="20"
        required
    >
    <span id="name-error" class="error-message" style="display: none;"></span>
</div>
```

**`static/crush/register.html`:**

```html
<div class="form-group">
    <label for="name">好きな人の名前（カタカナフルネーム）</label>
    <input
        type="text"
        id="name"
        placeholder="例: ヤマダタロウ"
        maxlength="20"
        required
    >
    <span id="name-error" class="error-message" style="display: none;"></span>
</div>
```

**変更点:**
- ラベルに「（カタカナフルネーム）」追加
- `maxlength="50"` → `maxlength="20"`
- `placeholder="例: 山田太郎"` → `placeholder="例: ヤマダタロウ"`
- エラーメッセージ表示用の`<span>`を追加

#### JavaScript変更

**`static/liff/register.js`と`static/crush/register.js`に追加:**

```javascript
/**
 * 名前バリデーション関数
 * @param {string} name - 検証する名前
 * @returns {object} { valid: boolean, message: string }
 */
function validateName(name) {
    const trimmed = name.trim();
    const length = [...trimmed].length; // 正確な文字数カウント

    // 長さチェック
    if (length < 2 || length > 20) {
        return {
            valid: false,
            message: '名前は2〜20文字で入力してください'
        };
    }

    // カタカナのみチェック
    const katakanaRegex = /^[ァ-ヴー]+$/;
    if (!katakanaRegex.test(trimmed)) {
        return {
            valid: false,
            message: '名前はカタカナフルネームで入力してください（例: ヤマダタロウ）'
        };
    }

    return { valid: true, message: '' };
}

/**
 * フォーム設定を拡張（既存のsetupForm関数内に追加）
 */
function setupForm() {
    // リアルタイムバリデーション（blur時）
    nameInput.addEventListener('blur', () => {
        const result = validateName(nameInput.value);
        const errorSpan = document.getElementById('name-error');

        if (!result.valid) {
            errorSpan.textContent = result.message;
            errorSpan.style.display = 'block';
            nameInput.style.borderColor = 'red';
        } else {
            errorSpan.style.display = 'none';
            nameInput.style.borderColor = '';
        }
    });

    // Submit時の処理（既存コードを拡張）
    form.addEventListener('submit', async (e) => {
        e.preventDefault();

        const name = nameInput.value.trim();
        const birthday = birthdayInput.value;

        // 名前バリデーション
        const nameResult = validateName(name);
        if (!nameResult.valid) {
            showMessage(nameResult.message, 'error');
            return;
        }

        // 誕生日バリデーション（既存）
        if (!birthday) {
            showMessage('生年月日を入力してください。', 'error');
            return;
        }

        // 登録処理（既存）
        await registerUser(name, birthday);
    });
}
```

**実装ポイント:**
- `validateName()`: バリデーションロジックを関数化
- `blur`イベント: 入力フィールドから離れた時にリアルタイムチェック
- エラー表示: `<span id="name-error">`にメッセージ表示、inputのborderを赤に
- Submit時: フロントエンドで再度バリデーション（二重チェック）

---

### 3. CSS追加

**`static/liff/register.css`と`static/crush/register.css`に追加:**

```css
/* エラーメッセージスタイル */
.error-message {
    color: #d32f2f;
    font-size: 0.875rem;
    margin-top: 0.25rem;
    display: block;
}

/* バリデーションエラー時のinputスタイル */
input[style*="border-color: red"] {
    border-color: #d32f2f !important;
}
```

---

### 4. Database Reset

#### データクリア方法

開発中デバッグデータのみのため、マイグレーションファイルは作成せず、シンプルに対応：

```bash
# ローカル環境
make db-reset

# EC2環境（デプロイ時）
ssh cupid-bot
cd ~/cupid
make db-reset
```

**実行タイミング:**
バリデーション実装後、テスト前に一度だけ実行：
1. バリデーション機能を実装
2. `make db-reset`でデータクリア
3. 新しいバリデーションルールでテスト開始

---

## テスト戦略

### Backend Tests

**`internal/model/user_test.go`に追加:**

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectValid bool
		expectMsg   string
	}{
		{
			name:        "有効: 標準的なカタカナフルネーム",
			input:       "ヤマダタロウ",
			expectValid: true,
			expectMsg:   "",
		},
		{
			name:        "有効: 最小文字数（2文字）",
			input:       "アベ",
			expectValid: true,
			expectMsg:   "",
		},
		{
			name:        "有効: 最大文字数（20文字）",
			input:       "アイウエオカキクケコサシスセソタチツテト",
			expectValid: true,
			expectMsg:   "",
		},
		{
			name:        "有効: 長音記号含む",
			input:       "カトウユーキ",
			expectValid: true,
			expectMsg:   "",
		},
		{
			name:        "無効: 1文字のみ",
			input:       "ア",
			expectValid: false,
			expectMsg:   "名前は2〜20文字で入力してください",
		},
		{
			name:        "無効: 21文字",
			input:       "アイウエオカキクケコサシスセソタチツテトナ",
			expectValid: false,
			expectMsg:   "名前は2〜20文字で入力してください",
		},
		{
			name:        "無効: 漢字含む",
			input:       "山田太郎",
			expectValid: false,
			expectMsg:   "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name:        "無効: ひらがな含む",
			input:       "やまだたろう",
			expectValid: false,
			expectMsg:   "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name:        "無効: 半角カタカナ",
			input:       "ﾔﾏﾀﾞﾀﾛｳ",
			expectValid: false,
			expectMsg:   "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name:        "無効: スペース含む",
			input:       "ヤマダ タロウ",
			expectValid: false,
			expectMsg:   "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name:        "無効: 英字含む",
			input:       "YamadaTarou",
			expectValid: false,
			expectMsg:   "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name:        "無効: 空文字",
			input:       "",
			expectValid: false,
			expectMsg:   "名前は2〜20文字で入力してください",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, msg := IsValidName(tt.input)
			assert.Equal(t, tt.expectValid, valid)
			if !tt.expectValid {
				assert.Equal(t, tt.expectMsg, msg)
			}
		})
	}
}
```

### Service Layer Tests

**`internal/service/user_service_test.go`に追加:**

```go
func TestUserService_RegisterFromLIFF_InvalidName(t *testing.T) {
	userRepo := new(MockUserRepository)
	likeRepo := new(MockLikeRepository)
	matchingService := new(MockMatchingService)
	service := NewUserService(userRepo, likeRepo, nil, "", matchingService)

	// 無効な名前（漢字）でエラーになることを確認
	err := service.RegisterFromLIFF(context.Background(), "U123", "山田太郎", "2000-01-15")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid name")
}

func TestUserService_RegisterCrush_InvalidName(t *testing.T) {
	userRepo := new(MockUserRepository)
	likeRepo := new(MockLikeRepository)
	matchingService := new(MockMatchingService)
	service := NewUserService(userRepo, likeRepo, nil, "", matchingService)

	currentUser := &model.User{
		LineID:           "U111",
		Name:             "ヤマダタロウ",
		Birthday:         "2000-01-15",
		RegistrationStep: 1,
	}

	userRepo.On("FindByLineID", mock.Anything, "U111").Return(currentUser, nil)

	// 無効な名前（ひらがな）でエラーになることを確認
	matched, matchedUserName, err := service.RegisterCrush(context.Background(), "U111", "やまだはなこ", "1995-05-20")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid crush name")
	assert.False(t, matched)
	assert.Empty(t, matchedUserName)
}
```

### Manual Frontend Tests

フロントエンドは手動でテスト（E2Eテストは今回スコープ外）：

#### テストケース

1. **リアルタイムバリデーション（blur時）:**
   - [ ] 「山田太郎」入力 → blur → エラーメッセージ表示、border赤
   - [ ] 「ヤマダタロウ」に修正 → blur → エラー消える、border通常
   - [ ] 「やまだたろう」入力 → blur → エラーメッセージ表示
   - [ ] 「ヤマダ タロウ」（スペース含む）→ blur → エラーメッセージ表示
   - [ ] 「ア」（1文字）→ blur → エラーメッセージ表示

2. **Submit時バリデーション:**
   - [ ] 無効な名前でSubmit → エラーメッセージ表示、登録されない
   - [ ] 有効な名前でSubmit → 登録成功

3. **エッジケース:**
   - [ ] 「カトウユーキ」（長音記号）→ 成功
   - [ ] 21文字以上 → エラー
   - [ ] 半角カタカナ「ﾔﾏﾀﾞﾀﾛｳ」→ エラー

---

## 影響範囲

### 変更が必要なファイル

1. **Backend:**
   - `internal/model/user.go` - `IsValidName()`関数追加
   - `internal/model/user_test.go` - テスト追加
   - `internal/service/user_service.go` - バリデーション呼び出し追加
   - `internal/service/user_service_test.go` - テスト追加

2. **Frontend:**
   - `static/liff/register.html` - label/placeholder/maxlength変更、エラー表示要素追加
   - `static/liff/register.js` - バリデーション関数とイベントリスナー追加
   - `static/liff/register.css` - エラーメッセージスタイル追加
   - `static/crush/register.html` - 同上
   - `static/crush/register.js` - 同上
   - `static/crush/register.css` - 同上

3. **Database:**
   - なし（`make db-reset`で対応）

---

## 実装順序

### Phase 1: Backend実装
1. `model.IsValidName()`関数を実装
2. テスト作成・実行（`TestIsValidName`）
3. `RegisterFromLIFF`と`RegisterCrush`でバリデーション呼び出し
4. テスト追加・実行（Service層）

### Phase 2: Frontend実装
1. HTML変更（label, placeholder, maxlength, error span）
2. CSS追加（error-message スタイル）
3. JavaScript実装（validateName関数、イベントリスナー）
4. 手動テスト実行

### Phase 3: データリセット
1. `make db-reset`実行（ローカル）
2. EC2でも`make db-reset`実行（デプロイ時）

### Phase 4: 統合テスト
1. ローカルで全体動作確認
2. EC2にデプロイ
3. LINE Bot経由で動作確認

---

## 期待される効果

1. **表記ブレの完全排除**
   - カタカナのみに統一
   - スペースなしで統一
   - 全角のみに統一

2. **マッチング精度の向上**
   - 同一人物の名前が常に同じ表記になる
   - マッチング判定が確実になる

3. **データ品質の向上**
   - 不正な文字が入らない
   - DBクエリが高速化（正規化されたデータ）

4. **ユーザー体験の向上**
   - リアルタイムフィードバックで入力ミスを早期発見
   - 具体例付きエラーメッセージで直感的に理解できる

---

## 将来的な拡張

### オプションA: ふりがな自動変換（漢字→カタカナ）
もしユーザーから「漢字で入力させてほしい」という要望があった場合：
- フロントエンドで漢字→カタカナ自動変換APIを使用
- 例: [Yahoo!テキスト解析API](https://developer.yahoo.co.jp/webapi/jlp/furigana/v2/furigana.html)
- 変換後のカタカナでバリデーション実行

### オプションB: 名寄せ機能
もし「タロウ」「タロー」「太郎」などの表記ブレに対応したい場合：
- 類似度判定アルゴリズム（Levenshtein距離など）を導入
- ただし現時点では不要（カタカナ統一で十分）

---

## まとめ

カタカナフルネーム（スペースなし）のバリデーションを実装することで、名前の表記ブレを完全に排除し、マッチング精度を向上させる。

**キーポイント:**
- フロントエンド + バックエンド両方で実装（UX + セキュリティ）
- 全角カタカナのみ（ァ-ヴー）、2〜20文字
- リアルタイムバリデーションでユーザーに即座にフィードバック
- 既存データは`make db-reset`でクリア
- ライブラリ不要（Plain JavaScriptで十分）

**次のステップ:**
実装計画の作成 → git worktreeでの実装 → テスト → デプロイ
