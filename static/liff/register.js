// LIFF ID（環境変数から設定する想定）
const LIFF_ID = '2008809168-1A2B3C4D'; // TODO: 実際のLIFF IDに置き換える

// DOM要素
const form = document.getElementById('register-form');
const nameInput = document.getElementById('name');
const birthdayInput = document.getElementById('birthday');
const submitButton = document.getElementById('submit-button');
const loading = document.getElementById('loading');
const message = document.getElementById('message');

// ページ読み込み時にLIFFを初期化
window.addEventListener('load', () => {
    initializeLiff();
});

/**
 * LIFFを初期化
 */
function initializeLiff() {
    liff.init({ liffId: LIFF_ID })
        .then(() => {
            // ログインチェック
            if (!liff.isLoggedIn()) {
                liff.login();
            } else {
                // フォーム送信イベントを設定
                setupForm();
            }
        })
        .catch((err) => {
            console.error('LIFF initialization failed', err);
            showMessage('LIFFの初期化に失敗しました。', 'error');
        });
}

/**
 * フォーム送信イベントを設定
 */
function setupForm() {
    form.addEventListener('submit', async (e) => {
        e.preventDefault();

        const name = nameInput.value.trim();
        const birthday = birthdayInput.value;

        // バリデーション
        if (!name) {
            showMessage('名前を入力してください。', 'error');
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
        // ローディング表示
        showLoading(true);
        submitButton.disabled = true;

        // アクセストークン取得
        const accessToken = liff.getAccessToken();
        if (!accessToken) {
            throw new Error('アクセストークンの取得に失敗しました。');
        }

        // API呼び出し
        const response = await fetch('/api/register', {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${accessToken}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ name, birthday })
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || '登録に失敗しました。');
        }

        // 成功
        showMessage('登録が完了しました！', 'success');

        // 1秒後にLIFFウィンドウを閉じる
        setTimeout(() => {
            liff.closeWindow();
        }, 1000);

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
