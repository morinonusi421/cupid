// フロントエンド用メッセージ定数
// キューピッドちゃんのメッセージを一元管理

const MESSAGES = {
    // 共通バリデーション（クライアントサイドで弾かれる時に使う）
    validation: {
        nameLengthError: 'あうぅ...名前は2〜20文字で入力してくださいっ💦',
        nameFormatError: '名前はカタカナフルネーム(空白なし)で入力してくださいねっ✨（例: ヤマダタロウ）',
        liffAuthError: 'あうぅ...LINE認証に失敗しちゃいました💦 もう一度試してくださいっ',
    },

    // サーバーから error コードで返ってくるエラーに対応するユーザー向け文言
    apiErrors: {
        duplicateUser: 'あうぅ...その名前と誕生日の組み合わせは既に登録されていますっ💦\n\n別の情報で登録してくださいね✨',
        invalidBirthday: 'あうぅ...その日付は存在しませんっ💦\n\n正しい誕生日を入力してくださいね✨',
        general: 'ふえぇ...エラーが発生しちゃいましたっ💦\n\nもう一度試してみてくださいね✨',
        registrationFailed: 'あうぅ...登録に失敗しちゃいました💦 もう一度試してくださいっ',
    },

    // ユーザー登録（自分の情報）
    user: {
        nameRequired: 'あうぅ...名前を入力してくださいっ💦',
        birthdayRequired: 'あうぅ...生年月日を入力してくださいっ💦',
        registrationSuccess: 'やったぁ✨ 登録完了ですっ♡ LINEに戻ってくださいねっ！',
        updateSuccess: '完了ですっ✨ 情報を更新しましたよ♡ LINEに戻ってくださいねっ！',
        cannotRegisterYourself: 'あうぅ...自分自身は登録できませんっ💦',
    },

    // 好きな人登録
    crush: {
        nameRequired: 'あうぅ...好きな人の名前を入力してくださいっ💦',
        birthdayRequired: 'あうぅ...好きな人の誕生日を入力してくださいっ💦',
        registrationSuccess: 'わぁっ♡ 登録完了ですっ💘 結果はLINEでお知らせしますねっ✨ ドキドキ〜！',
        updateSuccess: '了解ですっ✨ 情報を更新しましたよ♡ 結果はLINEでお知らせしますねっ💕',
        userNotRegistered: 'あうぅ...先に自分の情報を登録してくださいっ💦',
        cannotRegisterYourself: 'あうぅ...自分自身を好きな人として登録することはできませんっ💦',
    },
};
