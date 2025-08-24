package logger

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"io"
	"time"
)

type MiddlewareBuilder struct {
	allowReqBody  bool
	allowRespBody bool
	logFunc       func(*gin.Context, *AccessLog)
}

func NewLoggerBuilder(logfunc func(ctx *gin.Context, log *AccessLog)) *MiddlewareBuilder {
	return &MiddlewareBuilder{logFunc: logfunc}
}

func (b *MiddlewareBuilder) AllowReqBody() *MiddlewareBuilder {
	b.allowReqBody = true
	return b
}

func (b *MiddlewareBuilder) AllowRespBody() *MiddlewareBuilder {
	b.allowRespBody = true
	return b
}

func (b *MiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		now := time.Now()
		url := ctx.Request.URL.String()
		if len(url) > 1024 {
			url = url[:1024]
		}
		al := &AccessLog{Method: ctx.Request.Method,
			Url: url}
		if b.allowReqBody && ctx.Request.Body != nil {
			// 直接忽略 error，不影响程序运行
			reqBodyBytes, _ := ctx.GetRawData()
			// Request.Body 是一个 Stream（流）对象，所以是只能读取一次的
			// 因此读完之后要放回去，不然后续步骤是读不到的
			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
			al.ReqBody = string(reqBodyBytes)
		}
		//为了获取gin的响应体返回值，需要装饰gin的Writer结构体
		if b.allowRespBody {
			ctx.Writer = responseWriter{ResponseWriter: ctx.Writer, al: al} //传入一个ctx自带的writer结构体，组合接口中 没有被覆盖的方法就会调用writer的方法
		}

		//在后续业务执行完后调用日志输出函数，具体咋输出由外部传入
		//函数指针的用法：让外部自定义操作内部参数
		defer func() {
			duration := time.Since(now)
			al.Duration = duration.String()
			b.logFunc(ctx, al)
		}()

		ctx.Next()
	}
}

type AccessLog struct {
	//http请求的方法
	Method string
	//url整个请求记录
	Url        string
	ReqBody    string
	RespBody   string
	StatusCode int
	Duration   string
}

type responseWriter struct {
	gin.ResponseWriter //组合接口，只需要实现gin在写回响应时用到的方法即可 组合相比接口字段的区别：方法提升+可覆写
	al                 *AccessLog
}

// 覆写接口的方法
func (r responseWriter) Write(data []byte) (int, error) {
	r.al.RespBody = string(data)
	return r.ResponseWriter.Write(data)
}

// 覆写接口的方法
func (r responseWriter) WriteHeader(statusCode int) {
	r.al.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// 覆写接口的方法
func (r responseWriter) WriteString(data string) (int, error) {
	r.al.RespBody = data
	return r.ResponseWriter.WriteString(data)
}
