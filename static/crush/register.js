// DOM要素
const nameInput = document.getElementById('name');
const nameError = document.getElementById('name-error');

// URLパラメータからuser_idを取得
function getUserIdFromURL() {
    const params = new URLSearchParams(window.location.search);
    return params.get('user_id');
}

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

// ページ読み込み時にフォーム設定
window.addEventListener('load', () => {
    setupForm();
});

/**
 * フォーム設定
 */
function setupForm() {
    // 名前入力のblurイベント（リアルタイムバリデーション）
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
}

// フォーム送信処理
document.getElementById('register-form').addEventListener('submit', async (e) => {
    e.preventDefault();

    const name = document.getElementById('name').value.trim();
    const birthday = document.getElementById('birthday').value;
    const userId = getUserIdFromURL();

    if (!userId) {
        showMessage('エラー: ユーザーIDが取得できませんでした', 'error');
        return;
    }

    if (!name || !birthday) {
        showMessage('名前と誕生日を入力してください', 'error');
        return;
    }

    // 名前の詳細バリデーション
    const nameValidation = validateName(name);
    if (!nameValidation.valid) {
        showMessage(nameValidation.message, 'error');
        return;
    }

    // UI更新
    document.getElementById('submit-button').disabled = true;
    document.getElementById('loading').style.display = 'block';
    document.getElementById('message').style.display = 'none';

    try {
        const response = await fetch('/api/register-crush', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                user_id: userId,
                crush_name: name,
                crush_birthday: birthday
            })
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error || '登録に失敗しました');
        }

        // 成功
        showMessage(data.message, data.matched ? 'matched' : 'success');

        // マッチングした場合は3秒後にLINEに戻る
        if (data.matched) {
            setTimeout(() => {
                if (window.liff && window.liff.isInClient()) {
                    window.liff.closeWindow();
                }
            }, 3000);
        }

    } catch (error) {
        console.error('Registration error:', error);
        showMessage(error.message, 'error');
    } finally {
        document.getElementById('submit-button').disabled = false;
        document.getElementById('loading').style.display = 'none';
    }
});

// メッセージ表示
function showMessage(text, type) {
    const messageEl = document.getElementById('message');
    messageEl.textContent = text;
    messageEl.className = type;
    messageEl.style.display = 'block';
}
