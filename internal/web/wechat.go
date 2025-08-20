package web

import (
	"github.com/gin-gonic/gin"
	"learning_go/webook/internal/service/oauth2/wechat"
	"net/http"
)

type OAuth2WechatHandler struct {
	svc wechat.WechatService
}

func NewOAuth2WechatHandler(svc wechat.WechatService) *OAuth2WechatHandler {
	return &OAuth2WechatHandler{svc: svc}
}

func (oauth *OAuth2WechatHandler) RegisterRouter(engine *gin.Engine) {
	server := engine.Group("/oauth2/wechat")
	server.GET("/authurl", oauth.AuthURL)
	server.Any("/callback", oauth.Callback)
}

func (oauth *OAuth2WechatHandler) AuthURL(ctx *gin.Context) {
	url, err := oauth.svc.AuthRUL(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "构造扫码登陆url失败"})
		return
	}
	ctx.JSON(http.StatusOK, Result{Data: url})
}

func (oauth *OAuth2WechatHandler) Callback(ctx *gin.Context) {

}
