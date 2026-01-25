// TODO: セキュリティ改善 - ワンタイムトークン方式に変更する
// 現在はURLパラメータに直接user_idを含めているが、なりすまし可能
// 将来的にはサーバー生成のワンタイムトークンを使用すべき

// DOM要素
const form = document.getElementById('register-form');
const nameInput = document.getElementById('name');
const birthdayInput = document.getElementById('birthday');
const submitButton = document.getElementById('submit-button');
const loading = document.getElementById('loading');
const message = document.getElementById('message');

// ページ読み込み時にフォーム設定
window.addEventListener('load', () => {
    setupForm();
});

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
 * URLパラメータからuser_idを取得
 */
function getUserIdFromURL() {
    const params = new URLSearchParams(window.location.search);
    return params.get('user_id');
}

/**
 * ユーザー登録
 */
async function registerUser(name, birthday) {
    try {
        // ローディング表示
        showLoading(true);
        submitButton.disabled = true;

        // URLパラメータからuser_idを取得
        const userId = getUserIdFromURL();
        if (!userId) {
            throw new Error('ユーザーIDが見つかりません。URLが正しいか確認してください。');
        }

        // API呼び出し
        const response = await fetch('/api/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ user_id: userId, name, birthday })
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
