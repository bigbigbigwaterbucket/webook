package ginx

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

var vector *prometheus.CounterVec

func InitOpt(opt prometheus.CounterOpts) {
	vector = prometheus.NewCounterVec(opt, []string{"code"})
	prometheus.MustRegister(vector)
	//这里还可以用method、命中路由、状态码等等，code是你自定义的业务错误码，方便你自己定位错误
}

func WrapperReqAndToken[Req any, C jwt.Claims](handler func(*gin.Context, Req, C) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		err := ctx.Bind(&req)
		if err != nil {
			ctx.JSON(http.StatusOK, Result{Msg: "请求参数绑定失败"})
			return
		}
		c := ctx.MustGet("user")
		cReal, ok := c.(C)
		if !ok {
			ctx.JSON(http.StatusUnauthorized, Result{Msg: "未登录"})
			return
		}
		res, err := handler(ctx, req, cReal)
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		if err != nil {
			zap.L().Error("err", zap.String("path", ctx.Request.URL.Path), zap.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}

func WrapperToken[C jwt.Claims](handler func(*gin.Context, C) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		c := ctx.MustGet("user")
		cReal, ok := c.(C)
		if !ok {
			ctx.JSON(http.StatusUnauthorized, Result{Msg: "未登录"})
			return
		}
		res, err := handler(ctx, cReal)
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		if err != nil {
			zap.L().Error("err", zap.String("path", ctx.Request.URL.Path), zap.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}

func Wrapper(handler func(*gin.Context) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		res, err := handler(ctx)
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		if err != nil {
			zap.L().Error("err", zap.String("path", ctx.Request.URL.Path), zap.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}

func WrapperReq[Req any](handler func(*gin.Context, Req) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		err := ctx.Bind(&req)
		if err != nil {
			ctx.JSON(http.StatusOK, Result{Msg: "请求参数绑定失败"})
			return
		}
		res, err := handler(ctx, req)
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		if err != nil {
			zap.L().Error("err", zap.String("path", ctx.Request.URL.Path), zap.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}
