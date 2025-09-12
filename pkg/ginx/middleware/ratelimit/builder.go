package webratelimit

import (
	_ "embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"learning_go/webook/pkg/ratelimit"
	"log"
	"net/http"
)

type Builder struct {
	prefix  string //Redis key 的“命名空间”，方便不同的业务进行不同的限流
	limiter ratelimit.Limiter
}

func NewBuilder(limit ratelimit.Limiter) *Builder {
	return &Builder{
		prefix:  "ip-limiter", //这里执行的是ip限流，是最广层面的对web业务的访问限流
		limiter: limit,
	}
}

func (b *Builder) Prefix(prefix string) *Builder {
	b.prefix = prefix
	return b
}

func (b *Builder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limited, err := b.limit(ctx)
		if err != nil {
			log.Println(err)
			//限流出错该怎么办？系统错误
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if limited {
			log.Println(err)
			//限流，返回tooManyRequests错误
			ctx.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		ctx.Next() //如果没被限流，把控制权交给下一个"中间件"或路由处理函数
		//gin独有
	}
}

func (b *Builder) limit(ctx *gin.Context) (bool, error) {
	key := fmt.Sprintf("%s:%s", b.prefix, ctx.ClientIP())
	return b.limiter.Limit(ctx, key)
}
