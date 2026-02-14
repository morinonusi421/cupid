// LIFF ID（本番用）
const LIFF_ID = '2009059076-kBsUXYIC';

// DOM要素
const form = document.getElementById('register-form');
const nameInput = document.getElementById('name');
const birthdayInput = document.getElementById('birthday');
const submitButton = document.getElementById('submit-button');
const loading = document.getElementById('loading');
const message = document.getElementById('message');

/**
 * 名前のバリデーション
 * @param {string} name - 検証する名前
 * @returns {{valid: boolean, message: string}} 検証結果
 */
function validateName(name) {
    const trimmed = name.trim();
    const length = [...trimmed].length;

    // 長さチェック（2〜20文字）
    if (length < 2 || length > 20) {
        return {
            valid: false,
            message: 'あうぅ...名前は2〜20文字で入力してくださいっ💦'
        };
    }

    // カタカナチェック
    const katakanaRegex = /^[ァ-ヴー]+$/;
    if (!katakanaRegex.test(trimmed)) {
        return {
            valid: false,
            message: '名前はカタカナフルネーム(空白なし)で入力してくださいねっ✨（例: ヤマダタロウ）'
        };
    }

    return { valid: true, message: '' };
}

// ページ読み込み時にLIFF初期化
window.addEventListener('load', async () => {
    try {
        await liff.init({ liffId: LIFF_ID });

        if (!liff.isLoggedIn()) {
            liff.login(); // 未ログインならLINEログイン画面へ
            return;
        }

        setupForm(); // ログイン済みならフォーム表示
    } catch (error) {
        console.error('LIFF initialization failed', error);
        showMessage('あうぅ...LINE認証に失敗しちゃいました💦 もう一度試してくださいっ', 'error');
    }
});

/**
 * フォーム送信イベントを設定
 */
function setupForm() {
    // 名前入力のblurイベント（リアルタイムバリデーション）
    const nameError = document.getElementById('name-error');
    nameInput.addEventListener('blur', () => {
        const result = validateName(nameInput.value);
        if (!result.valid) {
            nameError.textContent = result.message;
            nameError.style.display = 'block';
            nameInput.style.borderColor = 'red';
        } else {
            nameError.style.display = 'none';
            nameInput.style.borderColor = '';
        }
    });

    form.addEventListener('submit', async (e) => {
        e.preventDefault();

        const name = nameInput.value.trim();
        const birthday = birthdayInput.value;

        // バリデーション
        if (!name) {
            showMessage('あうぅ...名前を入力してくださいっ💦', 'error');
            return;
        }

        // 名前の詳細バリデーション
        const nameValidation = validateName(name);
        if (!nameValidation.valid) {
            showMessage(nameValidation.message, 'error');
            return;
        }

        if (!birthday) {
            showMessage('あうぅ...生年月日を入力してくださいっ💦', 'error');
            return;
        }

        // 登録処理
        await registerUser(name, birthday);
    });
}

/**
 * ユーザー登録
 * @param {boolean} confirmUnmatch - マッチング解除を確認済みかどうか
 */
async function registerUser(name, birthday, confirmUnmatch = false) {
    try {
        showLoading(true);
        submitButton.disabled = true;

        // IDトークン取得
        const idToken = liff.getIDToken();

        if (!idToken) {
            throw new Error('認証情報が取得できませんでした');
        }

        // API呼び出し
        const response = await fetch('/api/register-user', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${idToken}` // IDトークンをヘッダーで送信
            },
            body: JSON.stringify({
                name,
                birthday,
                confirm_unmatch: confirmUnmatch
            })
        });

        if (!response.ok) {
            const errorData = await response.json();

            // matched_user_existsの場合は確認ダイアログを表示
            if (errorData.error === 'matched_user_exists') {
                showLoading(false);
                const confirmed = confirm(errorData.message + '\n\n本当に変更しますか？');
                if (confirmed) {
                    // 確認済みで再度リクエスト
                    await registerUser(name, birthday, true);
                } else {
                    submitButton.disabled = false;
                }
                return;
            }

            // duplicate_userの場合は特別なエラーメッセージ
            if (errorData.error === 'duplicate_user') {
                throw new Error(errorData.message || '同じ名前・誕生日のユーザーが既に登録されています。');
            }

            throw new Error(errorData.error || '登録に失敗しました。');
        }

        // 成功 - 初回/再登録でメッセージを変える
        const data = await response.json();
        if (data.is_first_registration) {
            showMessage('やったぁ✨ 登録完了ですっ♡ LINEに戻ってくださいねっ！', 'success');
        } else {
            showMessage('完了ですっ✨ 情報を更新しましたよ♡ LINEに戻ってくださいねっ！', 'success');
        }

    } catch (error) {
        console.error('Registration failed', error);
        showMessage(error.message || 'あうぅ...登録に失敗しちゃいました💦 もう一度試してくださいっ', 'error');
        submitButton.disabled = false;
    } finally {
        showLoading(false);
    }
}

/**
 * ローディング表示切り替え
 */
function showLoading(isLoading) {
    if (isLoading) {
        form.style.display = 'none';
        loading.style.display = 'block';
        message.style.display = 'none';
    } else {
        form.style.display = 'block';
        loading.style.display = 'none';
    }
}

/**
 * メッセージ表示
 */
function showMessage(text, type) {
    message.textContent = text;
    message.className = type;
    message.style.display = 'block';
}
