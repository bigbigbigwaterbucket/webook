package web

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/service"
	"net/http"
)

var _ handler = (*ArticleHandler)(nil)

type ArticleHandler struct {
	svc service.ArticleService
}

func NewArticleHandler(svc service.ArticleService) *ArticleHandler {
	return &ArticleHandler{svc: svc}
}

func (a *ArticleHandler) RegisterRouter(engine *gin.Engine) {
	server := engine.Group("/articles")
	//非restful路由风格
	server.POST("/edit", a.Edit)
}

func (a *ArticleHandler) Edit(ctx *gin.Context) {
	type Req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "请求参数绑定失败"})
		return
	}
	aid, err := a.svc.Save(ctx, domain.Article{Title: req.Title, Content: req.Content})
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		zap.L().Error("帖子保存失败")
		return
	}
	ctx.JSON(http.StatusOK, Result{Msg: "OK", Data: aid})

}
