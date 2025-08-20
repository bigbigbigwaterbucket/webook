package wechat

import (
	"context"
	"fmt"
	uuid "github.com/lithammer/shortuuid/v4"
	"net/url"
)

type WechatService interface {
	AuthRUL(ctx context.Context) (string, error)
}

type wechatService struct {
	appId string
}

func NewWechatService(appId string) *wechatService {
	return &wechatService{appId: appId}
}

func (w *wechatService) AuthRUL(ctx context.Context) (string, error) {
	const urlPattern = "https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect"
	redirectURI := "https://meoying.com/oauth2/wechat/callback"
	//url字符编码
	redirectURI = url.PathEscape(redirectURI)
	return fmt.Sprintf(urlPattern, w.appId, redirectURI, uuid.New()), nil
}
