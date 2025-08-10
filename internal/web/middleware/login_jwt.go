package middleware

import (
	"encoding/gob"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"learning_go/webook/internal/web"
	"log"
	"net/http"
	"strings"
	"time"
)

type LoginJWTMiddlewareBuilder struct {
}

// 在除了登录和注册页面，验证登陆状态
func (this *LoginJWTMiddlewareBuilder) Build() gin.HandlerFunc {
	gob.Register(time.Now()) //注册编解码
	return func(context *gin.Context) {
		if context.Request.URL.Path == "/users/login" || context.Request.URL.Path == "/users/signup" {
			return
		}
		token := context.GetHeader("Authorization")
		if token == "" {
			context.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		segs := strings.SplitN(token, " ", 2)
		if len(segs) != 2 {
			context.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		tokenStr := segs[1]
		claims := &web.UserClaims{}
		//函数规范：传指针就会改指针指向的值，是写；传值就是只读
		tokenReal, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
			return []byte("1234567890abcdef1234567890abcdef"), nil
		})
		if err != nil {
			context.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if tokenReal == nil || !tokenReal.Valid || claims.Uid == 0 {
			context.AbortWithStatus(http.StatusUnauthorized) //直接退出，不会执行后续代码
			return
		}
		if claims.UserAgent != context.Request.UserAgent() {
			//严重安全问题,应当监控
			context.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		now := time.Now()
		//如果距离过期时间不到50秒
		if claims.ExpiresAt.Sub(now) < time.Second*50 {
			claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute))
			tokenStr, err = tokenReal.SignedString([]byte("1234567890abcdef1234567890abcdef"))
			if err != nil {
				log.Println("jwt续约失败")
			}
			context.Header("x-jwt-token", tokenStr)
		}
		context.Set("userId", claims.Uid)
		return
	}
}
