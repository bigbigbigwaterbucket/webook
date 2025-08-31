package ginx

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"net/http"
)

func WrapperBodyAndToken[Req any, C jwt.Claims](handler func(*gin.Context, Req, C) (Result, error)) gin.HandlerFunc {
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
		if err != nil {
			zap.L().Error("err", zap.String("path", ctx.Request.URL.Path), zap.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}
