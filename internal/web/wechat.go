package web

import (
	"errors"
	"fmt"
	"learning_go/webook/api/proto/gen/user/userv1"
	"learning_go/webook/internal/service/oauth2/wechat"
	"learning_go/webook/internal/web/ijwt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	uuid "github.com/lithammer/shortuuid/v4"
)

type OAuth2WechatHandler struct {
	wechatsvc wechat.WechatService
	userSvc   userv1.UserServiceClient
	stateName string
	ijwt.JwtHandler
}

func NewOAuth2WechatHandler(svc wechat.WechatService, usersvc userv1.UserServiceClient, jwt ijwt.JwtHandler) *OAuth2WechatHandler {
	return &OAuth2WechatHandler{wechatsvc: svc, userSvc: usersvc, stateName: "jwt-state", JwtHandler: jwt}
}

func (oauth *OAuth2WechatHandler) RegisterRouter(engine *gin.Engine) {
	server := engine.Group("/oauth2/wechat")
	server.GET("/authurl", oauth.AuthURL)
	server.Any("/callback", oauth.Callback)
}

func (oauth *OAuth2WechatHandler) AuthURL(ctx *gin.Context) {
	state := uuid.New()
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, StateClaim{UUID: state})
	tokenStr, err := token.SignedString([]byte("1234567890abcdef1234567890abcdef"))
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		return
	}
	//path:限制在哪个路径及其子路径下才生效  domain:限制cookie在哪个域名下生效，“”表示当前域名  httpOnly：是否只允许服务器访问，JS不能访问
	ctx.SetCookie(oauth.stateName, tokenStr, 600, "/oauth2/wechat/callback", "", false, true)
	url, err := oauth.wechatsvc.AuthRUL(ctx, state)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "构造扫码登陆url失败"})
		return
	}
	ctx.JSON(http.StatusOK, Result{Data: url})
}

func (oauth *OAuth2WechatHandler) Callback(ctx *gin.Context) {
	//ctx.Param拿到的是url中的参数，Query拿到的才是get请求?后面的参数
	code := ctx.Query("code")
	err := oauth.VerifyState(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "state验证失败"})
		return
	}
	fmt.Println(code)
	wInfo, err := oauth.wechatsvc.VerifyCode(ctx, code)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "code验证失败"})
		return
	}
	//这不应该交给前端做了，获得accesstoken不需要扫码，自己后端解决就可以
	//总没必要让前端帮你发个请求再把结果给你吧？？？
	//ctx.JSON(http.StatusOK, Result{Data: url})
	resp, err := oauth.userSvc.FindOrCreateByWechat(ctx, &userv1.FindOrCreateByWechatReq{Info: &userv1.WechatInfo{UnionId: wInfo.UnionId, OpenId: wInfo.OpenId}})
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "创建微信用户失败"})
		return
	}
	err = oauth.SetLoginToken(ctx, resp.User.Id)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "设置token失败"})
		return
	}
	ctx.String(200, "欢迎，"+strconv.FormatInt(resp.User.Id, 10))
}

func (oauth *OAuth2WechatHandler) VerifyState(ctx *gin.Context) error {
	state := ctx.Query("state")
	tokenStr, err := ctx.Cookie(oauth.stateName)
	if err != nil {
		//没找到state，有人攻击，要做好监控
		return fmt.Errorf("请求参数没有state, %w", err)
	}
	var stateClaim StateClaim
	token, err := jwt.ParseWithClaims(tokenStr, &stateClaim, func(token *jwt.Token) (any, error) {
		return []byte("1234567890abcdef1234567890abcdef"), nil
	})
	//其实valid不为true也会报err，但保险起见两边都验证一下
	if err != nil || !token.Valid {
		return fmt.Errorf("cookie中的state过期, %w", err)
	}
	if stateClaim.UUID != state {
		return errors.New("cookie中的state遭到篡改")
	}
	return nil
}

type StateClaim struct {
	jwt.RegisteredClaims
	//Go 里小写字段是私有的，encoding/json（jwt 底层用的也是 JSON 序列化）根本不会处理。
	//所以 uuid 根本不会被写入 token，解析出来就是空字符串。
	//这里也必须是大写的
	UUID string
}
