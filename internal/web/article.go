package web

import (
	"fmt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"learning_go/webook/api/proto/gen/interactive/intrv1"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/service"
	"learning_go/webook/internal/web/ijwt"
	"learning_go/webook/pkg/ginx"
	"net/http"
	"strconv"
)

var _ handler = (*ArticleHandler)(nil)

type ArticleHandler struct {
	svc      service.ArticleService
	interSvc intrv1.InteractiveServiceClient
	biz      string
}

func NewArticleHandler(svc service.ArticleService, interSvc intrv1.InteractiveServiceClient) *ArticleHandler {
	return &ArticleHandler{svc: svc, interSvc: interSvc, biz: "article"}
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
	pub := server.Group("/pub")
	pub.GET("/:id", ginx.WrapperToken[ijwt.UserClaims](a.PubDetail))
	pub.POST("/like", ginx.WrapperReqAndToken[LikeReq, ijwt.UserClaims](a.Like))
	pub.POST("/collect", ginx.WrapperReqAndToken[CollectReq, ijwt.UserClaims](a.Collect))
	pub.POST("/:top", a.GetTop)
}

func (a *ArticleHandler) GetTop(ctx *gin.Context) {
	//没必要定义请求结构体，get请求，想统计什么top直接从路由参数那拿就可以
	var err error
	var req TopReq
	var topData []*intrv1.Interactive
	var articles []domain.Article
	top := ctx.Param("top")
	err = ctx.Bind(&req)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		return
	}
	switch top {
	case "liketop":
		//这里拿到的数据已经被减少过了
		getlikeResp, err := a.interSvc.GetLikeTop(ctx, &intrv1.GetLikeTopReq{Biz: a.biz, TopNum: req.TopNum})
		if err != nil {
			ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
			return
		}
		topData = getlikeResp.Inters
		ids := slice.Map[*intrv1.Interactive, int64](topData, func(idx int, src *intrv1.Interactive) int64 {
			return src.BizId
		})
		articles, err = a.svc.GetByIds(ctx, ids)
		ctx.JSON(http.StatusOK, Result{Msg: "OK", Data: slice.Map[domain.Article, ArticleVO](articles, func(idx int, src domain.Article) ArticleVO {
			return ArticleVO{
				Id:    src.Id,
				Title: src.Title,
				//Abstract: res.Abstract(),
				Status:  src.Status.ToUnt8(),
				Ctime:   src.CTime,
				Utime:   src.UTime,
				Content: src.Content,
			}
		}),
		})
	default:
		//前端传错了或者有人乱发
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		return
	}

}

func (a *ArticleHandler) Like(ctx *gin.Context, req LikeReq, claim ijwt.UserClaims) (Result, error) {
	var err error
	if req.Like {
		_, err = a.interSvc.Like(ctx, &intrv1.LikeReq{Aid: req.Id, Uid: claim.Uid, Biz: a.biz})
	} else {
		_, err = a.interSvc.UnLike(ctx, &intrv1.UnLikeReq{Aid: req.Id, Uid: claim.Uid, Biz: a.biz})
	}
	if err != nil {
		return Result{Msg: "系统错误"}, err
	}
	return Result{Msg: "OK"}, nil
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
	res, err := a.svc.GetById(ctx, aid)
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

func (a *ArticleHandler) PubDetail(ctx *gin.Context, claim ijwt.UserClaims) (Result, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return Result{Msg: "系统错误"}, err
	}
	//异步操作，拿到res数据后再操作
	//开启一个封装了sync.WaitGroup的errGroup，用来等待异步组执行完并进行错误处理
	var eg errgroup.Group
	var interactiveData *intrv1.Interactive
	var res domain.Article
	//这样异步，io操作等待时是并行等待，总能省下时间
	eg.Go(func() error {
		var er error
		getResp, er := a.interSvc.Get(ctx, &intrv1.GetReq{Biz: a.biz, BizId: id, Uid: claim.Uid})
		if er != nil {
			zap.L().Error("文章点赞收藏等信息获取失败")
			return er
		}
		interactiveData = getResp.Inter
		return nil
	})
	eg.Go(func() error {
		res, err = a.svc.GetPublishedById(ctx, id)
		return err
	})
	err = eg.Wait()
	if err != nil {
		return Result{Msg: "系统错误"}, err
	}
	go func() {
		_, er := a.interSvc.IncreaseReadCount(ctx, &intrv1.IncreaseReadCountReq{Biz: a.biz, BizId: res.Id})
		if er != nil {
			zap.L().Error("阅读量增加失败")
			return
		}
		return
	}()
	return Result{
		Data: ArticleVO{
			Id:       res.Id,
			Title:    res.Title,
			Abstract: res.Abstract(),
			//Status:   res.Status.ToUnt8(),
			Ctime:      res.CTime,
			Utime:      res.UTime,
			Content:    res.Content,
			CollectCnt: interactiveData.CollectCnt,
			LikeCnt:    interactiveData.LikeCnt,
			ReadCnt:    interactiveData.ReadCnt,
			Liked:      interactiveData.Liked,
			Collected:  interactiveData.Collected,
		},
	}, nil
}

func (a *ArticleHandler) Collect(ctx *gin.Context, req CollectReq, claims ijwt.UserClaims) (ginx.Result, error) {
	//这里就没做取消收藏的功能
	_, err := a.interSvc.Collect(ctx, &intrv1.CollectReq{Biz: a.biz, BizId: req.Id, Cid: req.Cid, Uid: claims.Uid})
	if err != nil {
		return Result{Msg: "系统错误"}, err
	}
	return Result{Msg: "OK"}, nil
}
