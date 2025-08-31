package web

import (
	"fmt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/service"
	"learning_go/webook/internal/web/ijwt"
	"learning_go/webook/pkg/ginx"
	"net/http"
	"strconv"
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
	server.POST("/edit", ginx.WrapperReqAndToken[ArticleReq, ijwt.UserClaims](a.Edit))
	server.POST("/publish", a.Publish)
	server.POST("/withdraw", a.Withdraw)
	//创作者的分页查询接口 按照restful规范，应该用GET方法
	server.POST("/list", ginx.WrapperReqAndToken[ListReq, ijwt.UserClaims](a.List))
	server.GET("/detail/:id", ginx.WrapperToken[ijwt.UserClaims](a.Detail))
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

func (a *ArticleHandler) Edit(ctx *gin.Context, req ArticleReq, claim ijwt.UserClaims) (Result, error) {
	//这里不可能断言错误，因为login_jwt那最差也是传入空UserClaims
	aid, err := a.svc.Save(ctx, domain.Article{Id: req.Id, Title: req.Title, Content: req.Content, Author: domain.Author{Id: claim.Uid}})
	if err != nil {
		return Result{Msg: "系统错误"}, err
	}
	return Result{Msg: "OK", Data: aid}, err
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

func (a *ArticleHandler) List(ctx *gin.Context, req ListReq, claim ijwt.UserClaims) (ginx.Result, error) {
	res, err := a.svc.List(ctx, claim.Uid, req.Offset, req.Limit)
	if err != nil {
		return Result{Msg: "系统错误"}, err
	}
	return Result{
		//对切片的每个元素应用xxx函数，即map
		Data: slice.Map[domain.Article, ArticleVO](res, func(idx int, src domain.Article) ArticleVO {
			return ArticleVO{
				Id:    src.Id,
				Title: src.Title,
				//在列表页，不需要显示全文，只需要显示摘要，简单的摘要就是几句话
				Abstract: src.Abstract(),
				Status:   src.Status.ToUnt8(),
				Ctime:    src.CTime,
				Utime:    src.UTime,
				//Content: src.Content,
			}
		}),
	}, nil
}

func (a *ArticleHandler) Detail(ctx *gin.Context, claim ijwt.UserClaims) (ginx.Result, error) {
	id := ctx.Param("id")
	aid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		zap.L().Error("前端输入的 ID 不对", zap.Error(err))
		return Result{
			Code: 4,
			Msg:  "参数错误",
		}, fmt.Errorf("查询文章详情的 ID %s 不正确, %w", id, err)
	}
	res, err := a.svc.GetPublishedById(ctx, aid)
	if err != nil {
		return Result{Msg: "系统错误"}, err
	}
	if claim.Uid != res.Author.Id {
		zap.L().Error("作者与文章作者不匹配", zap.Int64("uid", claim.Uid))
		return Result{
			Code: 4,
			Msg:  "系统错误",
		}, fmt.Errorf("查询文章详情的作者ID %s 与文章作者ID不匹配", claim.Uid)
	}
	return Result{
		Data: ArticleVO{
			Id:    res.Id,
			Title: res.Title,
			//Abstract: res.Abstract(),
			Status:  res.Status.ToUnt8(),
			Ctime:   res.CTime,
			Utime:   res.UTime,
			Content: res.Content,
		},
	}, nil
}
