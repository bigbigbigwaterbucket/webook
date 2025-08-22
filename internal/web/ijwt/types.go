package ijwt

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type JwtHandler interface {
	ClearToken(ctx *gin.Context) error
	SetLoginToken(ctx *gin.Context, uid int64) error
	SetJwtToken(ctx *gin.Context, uid int64, ssid string) error
	SetRefreshToken(ctx *gin.Context, uid int64, ssid string) error
	GetTokenFromAuth(ctx *gin.Context) string
	CheckSsid(ctx *gin.Context, ssid string) error
}

// ssid在两个token里都存，这样无论在logout的时候带的auth字段是哪个token，都可以注销登录
// 存储于短token: access token
type UserClaims struct {
	jwt.RegisteredClaims
	Uid       int64
	Ssid      string
	UserAgent string
}

// 存储于长token: refresh token
type RefreshClaims struct {
	jwt.RegisteredClaims
	Ssid string
	Uid  int64
}
