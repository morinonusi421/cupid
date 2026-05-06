// 共通ユーティリティ関数

/**
 * 誕生日セレクトの初期化
 */
function initBirthdaySelects() {
    const birthYearSelect = document.getElementById('birth-year');
    const birthMonthSelect = document.getElementById('birth-month');
    const birthDaySelect = document.getElementById('birth-day');

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

/**
 * 誕生日を取得（YYYY-MM-DD形式）
 */
function getBirthday() {
    const birthYearSelect = document.getElementById('birth-year');
    const birthMonthSelect = document.getElementById('birth-month');
    const birthDaySelect = document.getElementById('birth-day');

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
 * @param {object} messages - エラーメッセージオブジェクト
 * @returns {{valid: boolean, message: string}} 検証結果
 */
function validateName(name, messages) {
    const trimmed = name.trim();
    const length = [...trimmed].length;

    // 長さチェック（2〜20文字）
    if (length < 2 || length > 20) {
        return {
            valid: false,
            message: messages.nameLengthError
        };
    }

    // カタカナチェック
    const katakanaRegex = /^[ァ-ヴー]+$/;
    if (!katakanaRegex.test(trimmed)) {
        return {
            valid: false,
            message: messages.nameFormatError
        };
    }

    return { valid: true, message: '' };
}

/**
 * プレビューモードの判定
 */
function isPreviewMode() {
    const params = new URLSearchParams(window.location.search);
    return params.get('preview') === 'true';
}

/**
 * ローディング表示切り替え
 */
function showLoading(isLoading) {
    const form = document.getElementById('register-form');
    const loading = document.getElementById('loading');
    const message = document.getElementById('message');

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
    const message = document.getElementById('message');
    message.textContent = text;
    message.className = type;
    message.style.display = 'block';
}

/**
 * API エラーハンドリング
 * @param {object} errorData - エラーレスポンスのJSONオブジェクト
 * @param {'user'|'crush'} context - どちらの登録フォームから呼ばれたか（コンテキスト依存メッセージの分岐用）
 * @param {function} onMatchedUserExists - matched_user_existsエラー時のコールバック
 * @returns {string|null} エラーメッセージ、またはnull（matched_user_existsの場合）
 */
function handleAPIError(errorData, context, onMatchedUserExists) {
    // 注: BEはerror codeのみ返す。ユーザー向け文言はFEが MESSAGES から選ぶ。

    // invalid_birthday: 日付が存在しない
    if (errorData.error === 'invalid_birthday') {
        return MESSAGES.apiErrors.invalidBirthday;
    }

    // matched_user_exists: マッチング中のユーザーが情報変更を試みた
    // BE は partner_name のみ返す。確認ダイアログの文言はFEが組み立てる。
    if (errorData.error === 'matched_user_exists') {
        const prompt = MESSAGES.apiErrors.matchedUserExistsConfirm(errorData.partner_name);
        const confirmed = confirm(prompt);
        if (confirmed && onMatchedUserExists) {
            onMatchedUserExists();
        }
        return null; // エラーとして扱わない
    }

    // duplicate_user: 同じ名前・誕生日の他人が既に登録済み
    if (errorData.error === 'duplicate_user') {
        return MESSAGES.apiErrors.duplicateUser;
    }

    // cannot_register_yourself: 自分自身を登録しようとした（フォーム種類で文言が違う）
    if (errorData.error === 'cannot_register_yourself') {
        return MESSAGES[context].cannotRegisterYourself;
    }

    // 名前のバリデーションエラー
    if (errorData.error === 'name_invalid_length') {
        return MESSAGES.validation.nameLengthError;
    }
    if (errorData.error === 'name_invalid_format') {
        return MESSAGES.validation.nameFormatError;
    }

    // internal_error: サーバーの想定外エラー
    if (errorData.error === 'internal_error') {
        return MESSAGES.apiErrors.general;
    }

    // 未知のエラー: error コードはユーザーに見せず、汎用メッセージにフォールバック
    return MESSAGES.apiErrors.registrationFailed;
}

/**
 * 最低ローディング時間を保証する
 * @param {number} startTime - Date.now()で取得した開始時刻
 * @param {number} minLoadingTime - 最低ローディング時間（ミリ秒）
 */
async function ensureMinimumLoadingTime(startTime, minLoadingTime) {
    const elapsed = Date.now() - startTime;
    if (elapsed < minLoadingTime) {
        await new Promise(resolve => setTimeout(resolve, minLoadingTime - elapsed));
    }
}
