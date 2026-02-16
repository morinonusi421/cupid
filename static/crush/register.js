// LIFF ID（本番用）
const LIFF_ID = '2009070891-iIdvFKtI';

// DOM要素
const form = document.getElementById('register-form');
const nameInput = document.getElementById('name');
const birthYearSelect = document.getElementById('birth-year');
const birthMonthSelect = document.getElementById('birth-month');
const birthDaySelect = document.getElementById('birth-day');
const submitButton = document.getElementById('submit-button');
const loading = document.getElementById('loading');
const message = document.getElementById('message');

// 誕生日セレクトの初期化
function initBirthdaySelects() {
    // 年を生成（1950年〜現在の年まで、降順）
    const currentYear = new Date().getFullYear();
    for (let year = currentYear; year >= 1950; year--) {
        const option = document.createElement('option');
        option.value = year;
        option.textContent = year;
        birthYearSelect.appendChild(option);
    }

    // 月を生成（1〜12月）
    for (let month = 1; month <= 12; month++) {
        const option = document.createElement('option');
        option.value = month;
        option.textContent = month;
        birthMonthSelect.appendChild(option);
    }

    // 日を生成（1〜31日）
    for (let day = 1; day <= 31; day++) {
        const option = document.createElement('option');
        option.value = day;
        option.textContent = day;
        birthDaySelect.appendChild(option);
    }
}

// 誕生日を取得（YYYY-MM-DD形式）
function getBirthday() {
    const year = birthYearSelect.value;
    const month = birthMonthSelect.value.padStart(2, '0');
    const day = birthDaySelect.value.padStart(2, '0');

    if (!year || !month || !day) {
        return null;
    }

    return `${year}-${month}-${day}`;
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
            message: MESSAGES.validation.nameLengthError
        };
    }

    // カタカナチェック
    const katakanaRegex = /^[ァ-ヴー]+$/;
    if (!katakanaRegex.test(trimmed)) {
        return {
            valid: false,
            message: MESSAGES.validation.nameFormatError
        };
    }

    return { valid: true, message: '' };
}

// プレビューモードの判定
function isPreviewMode() {
    const params = new URLSearchParams(window.location.search);
    return params.get('preview') === 'true';
}

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
        const birthday = getBirthday();

        // バリデーション
        if (!name) {
            showMessage(MESSAGES.crush.nameRequired, 'error');
            return;
        }

        // 名前の詳細バリデーション
        const nameValidation = validateName(name);
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
            await new Promise(resolve => setTimeout(resolve, MIN_LOADING_TIME));
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
                'Authorization': `Bearer ${idToken}` // IDトークンをヘッダーで送信
            },
            body: JSON.stringify({
                crush_name: name,
                crush_birthday: birthday,
                confirm_unmatch: confirmUnmatch
            })
        });

        if (!response.ok) {
            console.log('[DEBUG] Response not OK, status:', response.status);
            const errorData = await response.json();
            console.log('[DEBUG] Error data:', errorData);

            // user_not_foundの場合は自分の情報登録を促す
            if (errorData.error === 'user_not_found') {
                console.log('[DEBUG] Matched user_not_found error');
                showLoading(false);
                showMessage(errorData.message || MESSAGES.crush.userNotRegistered, 'error');
                submitButton.disabled = false;

                // ユーザー登録URLがあれば、3秒後に自動的に遷移
                if (errorData.user_liff_url) {
                    console.log('[DEBUG] Will redirect to:', errorData.user_liff_url);
                    setTimeout(() => {
                        window.location.href = errorData.user_liff_url;
                    }, 3000);
                }
                return;
            }

            // matched_user_existsの場合は確認ダイアログを表示
            if (errorData.error === 'matched_user_exists') {
                showLoading(false);
                const confirmed = confirm(errorData.message + '\n\n本当に変更しますか？');
                if (confirmed) {
                    // 確認済みで再度リクエスト
                    await registerCrush(name, birthday, true);
                } else {
                    submitButton.disabled = false;
                }
                return;
            }

            // invalid_birthdayの場合は特別なエラーメッセージ
            if (errorData.error === 'invalid_birthday') {
                throw new Error(errorData.message || 'その日付は存在しません。');
            }

            // 自己登録エラーの場合は特別なエラーメッセージ
            if (errorData.error === 'cannot_register_yourself') {
                throw new Error(MESSAGES.crush.cannotRegisterYourself);
            }

            throw new Error(errorData.error || '登録に失敗しました。');
        }

        // 成功 - 初回/再登録でメッセージを変える
        const data = await response.json();

        // 最低ローディング時間が経過するまで待機
        const elapsed = Date.now() - startTime;
        if (elapsed < MIN_LOADING_TIME) {
            await new Promise(resolve => setTimeout(resolve, MIN_LOADING_TIME - elapsed));
        }

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
