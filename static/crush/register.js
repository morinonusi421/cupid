// LIFF ID（本番用）
const LIFF_ID = '2009070891-iIdvFKtI';

// DOM要素
const form = document.getElementById('register-form');
const nameInput = document.getElementById('name');
const submitButton = document.getElementById('submit-button');

// ページ読み込み時にLIFF初期化
window.addEventListener('load', async () => {
    // プレビューモードならLIFF認証をスキップ
    if (isPreviewMode()) {
        console.log('Preview mode: LIFF authentication skipped');
        setupForm();
        return;
    }

    try {
        await liff.init({ liffId: LIFF_ID });

        if (!liff.isLoggedIn()) {
            liff.login(); // 未ログインならLINEログイン画面へ
            return;
        }

        setupForm(); // ログイン済みならフォーム表示
    } catch (error) {
        console.error('LIFF initialization failed', error);
        showMessage(MESSAGES.validation.liffAuthError, 'error');
    }
});

/**
 * フォーム送信イベントを設定
 */
function setupForm() {
    // 誕生日セレクトを初期化
    initBirthdaySelects();

    // 名前入力のblurイベント（リアルタイムバリデーション）
    const nameError = document.getElementById('name-error');
    nameInput.addEventListener('blur', () => {
        const result = validateName(nameInput.value, MESSAGES.validation);
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
        const birthday = getBirthday();

        // バリデーション
        if (!name) {
            showMessage(MESSAGES.crush.nameRequired, 'error');
            return;
        }

        // 名前の詳細バリデーション
        const nameValidation = validateName(name, MESSAGES.validation);
        if (!nameValidation.valid) {
            showMessage(nameValidation.message, 'error');
            return;
        }

        if (!birthday) {
            showMessage(MESSAGES.crush.birthdayRequired, 'error');
            return;
        }

        // 登録処理
        await registerCrush(name, birthday);
    });
}

/**
 * 好きな人登録
 * @param {string} name - 好きな人の名前
 * @param {string} birthday - 好きな人の誕生日
 * @param {boolean} confirmUnmatch - マッチング解除を確認済みかどうか
 */
async function registerCrush(name, birthday, confirmUnmatch = false) {
    try {
        showLoading(true);
        submitButton.disabled = true;

        // 最低ローディング時間（ミリ秒）
        const MIN_LOADING_TIME = 2000;
        const startTime = Date.now();

        // プレビューモードの場合はダミーの成功レスポンス
        if (isPreviewMode()) {
            await ensureMinimumLoadingTime(startTime, MIN_LOADING_TIME);
            showMessage(MESSAGES.crush.registrationSuccess, 'success');
            return;
        }

        // IDトークン取得
        const idToken = liff.getIDToken();

        if (!idToken) {
            throw new Error('認証情報が取得できませんでした');
        }

        // API呼び出し
        const response = await fetch('/api/register-crush', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${idToken}`
            },
            body: JSON.stringify({
                crush_name: name,
                crush_birthday: birthday,
                confirm_unmatch: confirmUnmatch
            })
        });

        if (!response.ok) {
            const errorData = await response.json();

            // user_not_foundの場合は自分の情報登録を促す（crush特有の処理）
            if (errorData.error === 'user_not_found') {
                showLoading(false);
                showMessage(errorData.message || MESSAGES.crush.userNotRegistered, 'error');
                submitButton.disabled = false;

                // ユーザー登録URLがあれば、3秒後に自動的に遷移
                if (errorData.user_liff_url) {
                    setTimeout(() => {
                        window.location.href = errorData.user_liff_url;
                    }, 3000);
                }
                return;
            }

            // その他のエラーハンドリング
            const errorMessage = handleAPIError(errorData, MESSAGES.crush, () => {
                // matched_user_existsの場合の再試行コールバック
                showLoading(false);
                registerCrush(name, birthday, true);
            });

            if (errorMessage) {
                throw new Error(errorMessage);
            } else {
                // matched_user_existsで拒否された場合
                submitButton.disabled = false;
                return;
            }
        }

        // 成功 - 初回/再登録でメッセージを変える
        const data = await response.json();

        // 最低ローディング時間が経過するまで待機
        await ensureMinimumLoadingTime(startTime, MIN_LOADING_TIME);

        if (data.is_first_registration) {
            showMessage(MESSAGES.crush.registrationSuccess, 'success');
        } else {
            showMessage(MESSAGES.crush.updateSuccess, 'success');
        }

    } catch (error) {
        console.error('Registration failed', error);
        showMessage(error.message || MESSAGES.crush.registrationError, 'error');
        submitButton.disabled = false;
    } finally {
        showLoading(false);
    }
}
