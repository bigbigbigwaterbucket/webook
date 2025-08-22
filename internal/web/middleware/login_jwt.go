package middleware

import (
	"encoding/gob"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"learning_go/webook/internal/web/ijwt"
	"net/http"
	"time"
)

type LoginJWTMiddlewareBuilder struct {
	HPath []string
	ijwt.JwtHandler
}

func (this *LoginJWTMiddlewareBuilder) AddHPath(path string) *LoginJWTMiddlewareBuilder {
	this.HPath = append(this.HPath, path)
	return this
}

// 在除了登录和注册页面，验证登陆状态
func (this *LoginJWTMiddlewareBuilder) Build() gin.HandlerFunc {
	gob.Register(time.Now()) //注册编解码
	return func(context *gin.Context) {
		for idx := range this.HPath {
			if context.Request.URL.Path == this.HPath[idx] {
				return
			}
		}
		tokenStr := this.GetTokenFromAuth(context)
		claims := ijwt.UserClaims{}
		//函数规范：传指针就会改指针指向的值，是写；传值就是只读
		tokenReal, err := jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (any, error) {
			return ijwt.AtKey, nil
		})
		if err != nil {
			context.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		//检查是否在黑名单里，是否已经注销
		err = this.CheckSsid(context, claims.Ssid)
		if err != nil {
			// 系统错误或者用户已经主动退出登录了
			// 这里也可以考虑说，如果在 Redis 已经崩溃的时候，
			// 就不要去校验是不是已经主动退出登录了。
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

		////token续约，刷新过期时间，已经被长短token取代了
		//now := time.Now()
		////如果距离过期时间不到50秒
		//if claims.ExpiresAt.Sub(now) < time.Second*50 {
		//	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute))
		//	tokenStr, err = tokenReal.SignedString([]byte("1234567890abcdef1234567890abcdef"))
		//	if err != nil {
		//		log.Println("jwt续约失败")
		//	}
		//	context.Header("x-jwt-token", tokenStr)
		//}
		println(claims.Uid)
		context.Set("userId", claims.Uid)
		context.Set("user", claims) //后续退出登录时要用到短token解析到的ssid，以此把ssid加入到令牌黑名单中
		return
	}
}
