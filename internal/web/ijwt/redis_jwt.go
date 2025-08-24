package ijwt

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"net/http"
	"strings"
	"time"
)

// accesstoken 加密key
var AtKey []byte = []byte(("1234567890abcdef1234567890abcdef"))

// refreshtoken 加密key
var RtKey []byte = []byte(("1234567890abcdef1234567890abcdex"))

type RedisJwtHandler struct {
	cmd redis.Cmdable
}

func NewRedisJwtHandler(cmd redis.Cmdable) *RedisJwtHandler {
	return &RedisJwtHandler{cmd: cmd}
}

func (h *RedisJwtHandler) key(ssid string) string { return fmt.Sprintf("users:ssid:%s", ssid) }

func (h *RedisJwtHandler) ClearToken(ctx *gin.Context) error {
	//清空token，后续访问让前端跳转401，返回登陆页面
	ctx.Header("x-jwt-token", "")
	ctx.Header("x-refresh-token", "")
	//断言一定能拿到，否则就panic
	//从login_jwt中间件那里拿到的
	userClaim, ok := ctx.MustGet("user").(UserClaims)
	if !ok {
		return errors.New("无法获取ssid")
	}
	//这里的ssid的生命周期与长token一致，长token失效时，必须重新登录，ssid也就会重新生成
	return h.cmd.Set(ctx, h.key(userClaim.Ssid), "", time.Hour).Err()
}

func (h *RedisJwtHandler) SetLoginToken(ctx *gin.Context, uid int64) error {
	ssid := uuid.New().String()
	err := h.SetJwtToken(ctx, uid, ssid)
	if err != nil {
		return err
	}
	err = h.SetRefreshToken(ctx, uid, ssid)
	return err
}

func (h *RedisJwtHandler) SetJwtToken(ctx *gin.Context, uid int64, ssid string) error {
	//60秒后过期，jwt的valid将变为false
	claims := UserClaims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30))},
		Uid:       uid,
		Ssid:      ssid,
		UserAgent: ctx.Request.UserAgent()}
	//生成带自定义信息的token
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString(AtKey)
	if err != nil {
		return err
	}
	ctx.Header("x-jwt-token", tokenStr)
	fmt.Printf("%s\n", tokenStr)
	fmt.Printf("%v\n", uid)
	return nil
}

func (h *RedisJwtHandler) SetRefreshToken(ctx *gin.Context, uid int64, ssid string) error {
	claims := RefreshClaims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		Uid:  uid,
		Ssid: ssid}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString(RtKey)
	if err != nil {
		return err
	}
	ctx.Header("x-refresh-token", tokenStr)
	fmt.Printf("%s\n", tokenStr)
	fmt.Printf("%v\n", uid)
	return nil
}

// 前端从一般路径发请求带的auth字段是短token,即aceessToken，从登录和refresh_token路径带的是长token，即refreshToken
func (h *RedisJwtHandler) GetTokenFromAuth(ctx *gin.Context) string {
	token := ctx.GetHeader("Authorization")
	if token == "" {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return ""
	}
	// SplitN 的意思是切割字符串，但是最多 N 段
	// 如果要是 N 为 0 或者负数，则是另外的含义，可以看它的文档
	segs := strings.SplitN(token, " ", 2)
	if len(segs) != 2 {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return ""
	}
	tokenStr := segs[1]
	return tokenStr
}

func (h *RedisJwtHandler) CheckSsid(ctx *gin.Context, ssid string) error {
	cnt, err := h.cmd.Exists(ctx, h.key(ssid)).Result()
	switch err {
	case redis.Nil:
		return nil
	case nil:
		if cnt > 0 {
			return errors.New("token已过期")
		}
		return nil
	default:
		return err
	}
}
