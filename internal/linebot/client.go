package linebot

import (
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// Client はLINE Messaging APIクライアントのインターフェース
type Client interface {
	ReplyMessage(request *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error)
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
