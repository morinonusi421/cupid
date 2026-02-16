// LIFF ID（本番用）
const LIFF_ID = '2009059076-kBsUXYIC';

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
            showMessage(MESSAGES.user.nameRequired, 'error');
            return;
        }

        // 名前の詳細バリデーション
        const nameValidation = validateName(name);
        if (!nameValidation.valid) {
            showMessage(nameValidation.message, 'error');
            return;
        }

        if (!birthday) {
            showMessage(MESSAGES.user.birthdayRequired, 'error');
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

        // プレビューモードの場合はダミーの成功レスポンス
        if (isPreviewMode()) {
            await new Promise(resolve => setTimeout(resolve, 2000)); // 2秒待機
            showMessage(MESSAGES.user.registrationSuccess, 'success');
            return;
        }

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

            // 自己登録エラーの場合は特別なエラーメッセージ
            if (errorData.error === 'cannot_register_yourself') {
                throw new Error(MESSAGES.user.cannotRegisterYourself);
            }

            throw new Error(errorData.error || '登録に失敗しました。');
        }

        // 成功 - 初回/再登録でメッセージを変える
        const data = await response.json();
        if (data.is_first_registration) {
            showMessage(MESSAGES.user.registrationSuccess, 'success');
        } else {
            showMessage(MESSAGES.user.updateSuccess, 'success');
        }

    } catch (error) {
        console.error('Registration failed', error);
        showMessage(error.message || MESSAGES.user.registrationError, 'error');
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
