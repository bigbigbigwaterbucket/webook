package middleware

import (
	"encoding/gob"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type LoginMiddlewareBuilder struct {
}

// 在除了登录和注册页面，验证登陆状态
func (this *LoginMiddlewareBuilder) Build() gin.HandlerFunc {
	gob.Register(time.Now()) //注册编解码
	return func(context *gin.Context) {
		if context.Request.URL.Path == "/users/login" || context.Request.URL.Path == "/users/signup" {
			return
		}
		sess := sessions.Default(context)
		//前面会设置session，因此sess不可能为nil，除非有人手动删除了sessions_id
		id := sess.Get("userId")
		if id == nil {
			context.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		updateTime := sess.Get("updateTime")
		sess.Set("userId", id)
		sess.Options(sessions.Options{MaxAge: 30 * 60}) //无论有没有登陆过，都先设置下，反正影响性能的是save()，只要没save就相当于没刷新max-age
		now := time.Now().UnixMilli()
		//没有说明刚登录，没刷新过
		if updateTime == nil {
			sess.Set("updateTime", now)
			sess.Save()
			return
		}
		updateTimeVal, ok := updateTime.(int64)
		if !ok {
			context.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if now-updateTimeVal > 60*1000 {
			sess.Set("updateTime", now)
			sess.Save()
		}
		return
	}
}
