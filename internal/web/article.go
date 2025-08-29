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
	server.POST("/withdraw", a.Withdraw)
}

func (a *ArticleHandler) Publish(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "请求参数绑定失败"})
		return
	}
	c := ctx.MustGet("user")
	claim, _ := c.(ijwt.UserClaims)
	id, err := a.svc.Publish(ctx, domain.Article{Id: req.Id, Title: req.Title, Content: req.Content, Author: domain.Author{Id: claim.Uid}})
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		zap.L().Error("帖子发布失败")
		return
	}
	//函数参数是值传递，不可能给你结构体改了的
	//ctx.JSON(http.StatusOK, Result{Msg: "OK", Data: req.id})
	ctx.JSON(http.StatusOK, Result{Msg: "OK", Data: id})
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

func (a *ArticleHandler) Withdraw(ctx *gin.Context) {
	type Req struct {
		Id int64 `json:"id"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "请求参数绑定失败"})
		return
	}
	c := ctx.MustGet("user")
	//这里不可能断言错误，因为login_jwt那最差也是传入空UserClaims
	claim, _ := c.(ijwt.UserClaims)
	err := a.svc.Withdraw(ctx, req.Id, claim.Uid)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		zap.L().Error("帖子撤销失败")
		return
	}
	ctx.JSON(http.StatusOK, Result{Msg: "OK"})
}

type ArticleReq struct {
	Id      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}
