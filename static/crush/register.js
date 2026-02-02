// URLパラメータからuser_idを取得
function getUserIdFromURL() {
    const params = new URLSearchParams(window.location.search);
    return params.get('user_id');
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
