package linebot

import (
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// Client はLINE Messaging APIクライアントのインターフェース
type Client interface {
	ReplyMessage(request *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error)
	PushMessage(request *messaging_api.PushMessageRequest) (*messaging_api.PushMessageResponse, error)
}

// client はLINE SDKをラップする実装
type client struct {
	api *messaging_api.MessagingApiAPI
}

// NewClient はLINE Bot Clientの新しいインスタンスを作成する
func NewClient(api *messaging_api.MessagingApiAPI) Client {
	return &client{api: api}
}

// ReplyMessage はメッセージを返信する
func (c *client) ReplyMessage(request *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error) {
	return c.api.ReplyMessage(request)
}

// PushMessage はメッセージをプッシュ送信する
// retry keyは空文字列を指定（リトライ機能は使用しない）
//
// 【重要】LINE Messaging APIの料金制限
// - 無料プランでは月200通まで送信可能（有償メッセージ）
// - Reply APIは無料だが、Push APIは有償カウント対象
// - 制限超過時は 429 Too Many Requests エラーが返される
func (c *client) PushMessage(request *messaging_api.PushMessageRequest) (*messaging_api.PushMessageResponse, error) {
	return c.api.PushMessage(request, "")
}
