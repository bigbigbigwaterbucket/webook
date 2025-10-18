package wechat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"learning_go/webook/internal/domain"

	"net/http"
	"net/url"
)

type WechatService interface {
	AuthRUL(ctx context.Context, state string) (string, error)
	VerifyCode(ctx context.Context, code string) (domain.WechatInfo, error)
}

type WechatServiceI struct {
	appId     string
	appSecret string
	webClient *http.Client //传地址
}

func NewWechatService(appId string, appSecret string) *WechatServiceI {
	return &WechatServiceI{appId: appId, appSecret: appSecret, webClient: http.DefaultClient} //DefaultClient就是一个空的http.Client类型
}

func (w *WechatServiceI) AuthRUL(ctx context.Context, state string) (string, error) {
	const urlPattern = "https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect"
	redirectURI := "https://meoying.com/oauth2/wechat/callback"
	//url字符编码
	redirectURI = url.PathEscape(redirectURI)
	return fmt.Sprintf(urlPattern, w.appId, redirectURI, state), nil
}

func (w *WechatServiceI) VerifyCode(ctx context.Context, code string) (domain.WechatInfo, error) {
	const urlPattern = "https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code"
	url := fmt.Sprintf(urlPattern, w.appId, w.appSecret, code)
	//内部发送一个请求的全流程:newreques构造一个新请求，然后构造一个httpClient，让客户端去do请求，即可拿到响应resp
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return domain.WechatInfo{}, err
	}
	resp, err := w.webClient.Do(req)
	if err != nil {
		return domain.WechatInfo{}, err
	}
	var res WechatResult
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return domain.WechatInfo{}, err
	}
	//成功就保留0值
	if res.ErrCode != 0 {
		return domain.WechatInfo{}, errors.New("微信登录信息验证失败:" + res.ErrMsg)
	}
	//微信登录成功，这里就要考虑findorcreate问题，由于涉及到数据库操作，需要定义一个domain对象，用于服务层、存储层、web层的数据交换
	return domain.WechatInfo{OpenId: res.Openid, UnionId: res.Unionid}, nil
}

type WechatResult struct {
	//这里接收json格式数据的反序列化结构体应当既包含错误信息的字段，也要包含正确字段
	//如果 JSON 里没有某个字段，那么 struct 中对应的字段会保留 “零值”。
	ErrCode int64  `json:"errcode"`
	ErrMsg  string `json:"errmsg"`

	//解析token的字段首字母必须大写，否则无法写入
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Openid       string `json:"openid"`
	Scope        string `json:"scope"`
	Unionid      string `json:"unionid"`
}
