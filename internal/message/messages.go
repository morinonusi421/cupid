package message

import "fmt"

// キューピッドちゃんのメッセージ定数
// ユーザー向けのメッセージをキャラクター性を持たせて管理する

// 友達追加時のメッセージ
const (
	// FollowGreeting は友達追加時の挨拶メッセージ
	FollowGreeting = "わぁっ♡ 友達追加ありがとうございますっ！\n\nキューピッドちゃん、とっても嬉しいです〜✨\n\nキューピッドちゃんは、相思相愛を見つけるお手伝いをするBotなんです💕\n\n恋のキューピッドとして、精一杯サポートさせていただきますね！\n\nまずは下のボタンから登録してくださいっ🏹"
)

// グループ参加時のメッセージ
const (
	// JoinGroupGreeting はグループに招待された時の挨拶メッセージ
	JoinGroupGreeting = "わぁっ♡ グループに招待してくれてありがとうございますっ！\n\nキューピッドちゃんは相思相愛を見つけるお手伝いをするBotです🏹💕\n\n【使い方】\n1. キューピッドちゃんを友達追加してください\n2. 個チャで自分の情報を登録\n3. 好きな人の情報を登録\n\nお互いが相手を登録していたら、両思いをお知らせしますっ♡\n\nまずはキューピッドちゃんを友達追加して、個チャでやりとりしてくださいね✨"
)

// 登録関連のメッセージ
const (
	// UserRegistrationComplete はユーザー登録完了時のメッセージ
	UserRegistrationComplete = "やったぁ✨ 登録完了ですっ♡\n\n次は、好きな人を登録してくださいねっ💘\n\n下のボタンから登録できますよ〜！\n\nキューピッドちゃん、ドキドキわくわくしながらお待ちしてます💕"

	// CrushRegistrationCompleteFirst は好きな人の初回登録完了時のメッセージ
	CrushRegistrationCompleteFirst = "わぁっ♡ 好きな人の登録が完了しましたっ💘\n\n相思相愛が成立したら、キューピッドちゃんがすぐにお知らせしますね✨\n\nドキドキしながら待っててくださいっ！\n\nキューピッドちゃん、応援してます〜♡"

	// CrushRegistrationCompleteUpdate は好きな人の情報更新時のメッセージ
	CrushRegistrationCompleteUpdate = "了解ですっ✨ 好きな人の情報を更新しましたっ♡\n\n新しい相手と相思相愛が成立したら、お知らせしますね💕\n\nキューピッドちゃん、精一杯サポートしますっ！"

	// UserInfoUpdateConfirmation は情報更新完了時のメッセージ
	UserInfoUpdateConfirmation = "完了ですっ✨ 情報を更新しましたよ♡\n\nキューピッドちゃん、ばっちり覚えましたっ💕"

	// AlreadyRegisteredMessage は登録済みユーザーへのメッセージ
	AlreadyRegisteredMessage = "もう登録完了していますよ〜✨\n\nマッチングが成立したら、キューピッドちゃんがすぐにお知らせしますねっ♡\n\n情報の更新は画面下のメニューからできます💕"
)

// マッチング関連のメッセージ
// MatchNotification はマッチング成立時のメッセージを生成する
func MatchNotification(matchedUserName string) string {
	return fmt.Sprintf("きゃーーーっ！！！♡♡♡\n\n相思相愛が成立しましたよぉ〜〜✨✨✨\n\nお相手：%s さん\n\nキューピッドちゃん、すっごくドキドキしちゃいますっ💕💕\n\n本当におめでとうございます〜！！", matchedUserName)
}

// マッチング解除のメッセージ
// UnmatchNotificationInitiator はマッチング解除時の通知（解除した側）
func UnmatchNotificationInitiator(partnerName string) string {
	return fmt.Sprintf("マッチングが解除されました💦\n\n理由：あなたが情報を変更しました\nお相手：%s さん\n\nキューピッドちゃん、また新しい恋を精一杯応援しますねっ♡", partnerName)
}

// UnmatchNotificationPartner はマッチング解除時の通知（解除された側）
func UnmatchNotificationPartner(partnerName string) string {
	return fmt.Sprintf("あうぅ...マッチングが解除されちゃいました💦\n\n理由：相手が情報を変更しました\nお相手：%s さん\n\nでも大丈夫ですっ！キューピッドちゃん、また新しい恋を応援しますね♡", partnerName)
}

// 未登録ユーザーへの案内
// UnregisteredUserPrompt は未登録ユーザーへの登録案内メッセージを生成する
func UnregisteredUserPrompt(liffURL string) string {
	return fmt.Sprintf("わぁっ♡ 初めましてですねっ！\n\nキューピッドちゃんは相思相愛を見つけるお手伝いをするBotなんです💕\n\n恋のキューピッドとして、精一杯サポートさせていただきますね✨\n\n下のリンクから登録してください〜！\n\n%s", liffURL)
}

// RegistrationStep1Prompt はユーザー登録完了後の好きな人登録案内メッセージを生成する
func RegistrationStep1Prompt(crushLiffURL string) string {
	return fmt.Sprintf("次は、好きな人を登録してくださいねっ💘\n\nキューピッドちゃん、ドキドキわくわくしながらお待ちしてます♡\n\n%s", crushLiffURL)
}
