// LIFF ID（環境変数から取得する想定、開発用）
const LIFF_ID = '2009059074-aX6pc41R';

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
            message: '名前は2〜20文字で入力してください'
        };
    }

    // カタカナチェック
    const katakanaRegex = /^[ァ-ヴー]+$/;
    if (!katakanaRegex.test(trimmed)) {
        return {
            valid: false,
            message: '名前はカタカナフルネームで入力してください（例: ヤマダタロウ）'
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
        showMessage('LINE認証に失敗しました。再度お試しください。', 'error');
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
            showMessage('名前を入力してください。', 'error');
            return;
        }

        // 名前の詳細バリデーション
        const nameValidation = validateName(name);
        if (!nameValidation.valid) {
            showMessage(nameValidation.message, 'error');
            return;
        }

        if (!birthday) {
            showMessage('生年月日を入力してください。', 'error');
            return;
        }

        // 登録処理
        await registerUser(name, birthday);
    });
}

/**
 * ユーザー登録
 */
async function registerUser(name, birthday) {
    try {
        showLoading(true);
        submitButton.disabled = true;

        // アクセストークン取得
        const accessToken = liff.getAccessToken();

        if (!accessToken) {
            throw new Error('認証情報が取得できませんでした');
        }

        // API呼び出し（user_idは送らない）
        const response = await fetch('/api/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${accessToken}` // トークンをヘッダーで送信
            },
            body: JSON.stringify({ name, birthday }) // user_id削除
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || '登録に失敗しました。');
        }

        // 成功
        showMessage('登録が完了しました！LINEに戻って話しかけてね。', 'success');

    } catch (error) {
        console.error('Registration failed', error);
        showMessage(error.message || '登録に失敗しました。', 'error');
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
