package web

import (
	"github.com/gin-gonic/gin"
	"learning_go/webook/internal/service"
	"learning_go/webook/internal/service/oauth2/wechat"
	"net/http"
)

type OAuth2WechatHandler struct {
	wechatsvc wechat.WechatService
	userSvc   service.UserService
}

func NewOAuth2WechatHandler(svc wechat.WechatService, usersvc service.UserService) *OAuth2WechatHandler {
	return &OAuth2WechatHandler{wechatsvc: svc, userSvc: usersvc}
}

func (oauth *OAuth2WechatHandler) RegisterRouter(engine *gin.Engine) {
	server := engine.Group("/oauth2/wechat")
	server.GET("/authurl", oauth.AuthURL)
	server.Any("/callback", oauth.Callback)
}

func (oauth *OAuth2WechatHandler) AuthURL(ctx *gin.Context) {
	url, err := oauth.wechatsvc.AuthRUL(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "构造扫码登陆url失败"})
		return
	}
	ctx.JSON(http.StatusOK, Result{Data: url})
}

func (oauth *OAuth2WechatHandler) Callback(ctx *gin.Context) {
	//ctx.Param拿到的是url中的参数，Query拿到的才是get请求?后面的参数
	code := ctx.Query("code")
	_ = ctx.Query("state")
	wInfo, err := oauth.wechatsvc.VerifyCode(ctx, code)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "验证失败"})
		return
	}
	//这不应该交给前端做了，获得accesstoken不需要扫码，自己后端解决就可以
	//总没必要让前端帮你发个请求再把结果给你吧？？？
	//ctx.JSON(http.StatusOK, Result{Data: url})
	user, err := oauth.userSvc.FindOrCreateByWechat(ctx, wInfo)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "创建微信用户失败"})
		return
	}

}
