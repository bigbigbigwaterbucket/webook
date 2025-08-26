package web

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/service"
	"learning_go/webook/internal/web/ijwt"
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
	server.POST("/publish", a.Publish)
}

func (a *ArticleHandler) Publish(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "请求参数绑定失败"})
		return
	}
	c := ctx.MustGet("user")
	claim, _ := c.(ijwt.UserClaims)
	err := a.svc.Publish(ctx, domain.Article{Id: req.Id, Title: req.Title, Content: req.Content, Author: domain.Author{Id: claim.Uid}})
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		zap.L().Error("帖子发布失败")
		return
	}
	ctx.JSON(http.StatusOK, Result{Msg: "OK", Data: req.Id})
}

func (a *ArticleHandler) Edit(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "请求参数绑定失败"})
		return
	}
	c := ctx.MustGet("user")
	//这里不可能断言错误，因为login_jwt那最差也是传入空UserClaims
	claim, _ := c.(ijwt.UserClaims)
	aid, err := a.svc.Save(ctx, domain.Article{Id: req.Id, Title: req.Title, Content: req.Content, Author: domain.Author{Id: claim.Uid}})
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		zap.L().Error("帖子保存失败")
		return
	}
	ctx.JSON(http.StatusOK, Result{Msg: "OK", Data: aid})

}

type ArticleReq struct {
	Id      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}
