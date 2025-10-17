package grpc

import (
	"context"
	"learning_go/webook/api/proto/gen/article/artv1"
	"learning_go/webook/article/domain"
	"learning_go/webook/article/service"

	"github.com/ecodeclub/ekit/slice"
)

type ArticleServiceServer struct {
	artv1.UnimplementedArticleServiceServer
	svc service.ArticleService
}

func NewArticleServiceServer(svc service.ArticleService) *ArticleServiceServer {
	return &ArticleServiceServer{svc: svc}
}

func (a *ArticleServiceServer) Save(ctx context.Context, req *artv1.SaveReq) (*artv1.SaveResp, error) {
	aid, err := a.svc.Save(ctx, a.toDomain(req.GetArticle()))
	return &artv1.SaveResp{ArticleId: aid}, err
}

func (a *ArticleServiceServer) Publish(ctx context.Context, req *artv1.PublishReq) (*artv1.PublishResp, error) {
	aid, err := a.svc.Publish(ctx, a.toDomain(req.GetArticle()))
	return &artv1.PublishResp{ArticleId: aid}, err
}

func (a *ArticleServiceServer) Withdraw(ctx context.Context, req *artv1.WithdrawReq) (*artv1.WithdrawResp, error) {
	err := a.svc.Withdraw(ctx, req.GetArticleId(), req.GetUserId())
	return &artv1.WithdrawResp{}, err
}

func (a *ArticleServiceServer) List(ctx context.Context, req *artv1.ListReq) (*artv1.ListResp, error) {
	arts, err := a.svc.List(ctx, req.GetUserId(), req.GetOffset(), req.GetLimit())
	return &artv1.ListResp{
		Articles: slice.Map[domain.Article, *artv1.Article](arts, func(idx int, src domain.Article) *artv1.Article {
			return a.toDao(src)
		}),
	}, err
}

func (a *ArticleServiceServer) GetById(ctx context.Context, req *artv1.GetByIdReq) (*artv1.GetByIdResp, error) {
	art, err := a.svc.GetById(ctx, req.GetArticleId())
	return &artv1.GetByIdResp{
		Article: a.toDao(art),
	}, err
}

func (a *ArticleServiceServer) GetPublishedById(ctx context.Context, req *artv1.GetPublishedByIdReq) (*artv1.GetPublishedByIdResp, error) {
	art, err := a.svc.GetPublishedById(ctx, req.ArticleId)
	return &artv1.GetPublishedByIdResp{
		Article: a.toDao(art),
	}, err
}

func (a *ArticleServiceServer) GetByIds(ctx context.Context, req *artv1.GetByIdsReq) (*artv1.GetByIdsResp, error) {
	arts, err := a.svc.GetByIds(ctx, req.GetArticleIds())
	return &artv1.GetByIdsResp{Articles: slice.Map[domain.Article, *artv1.Article](arts, func(idx int, src domain.Article) *artv1.Article {
		return a.toDao(src)
	})}, err
}

func (a *ArticleServiceServer) toDomain(article *artv1.Article) domain.Article {
	return domain.Article{
		Id:      article.Id,
		Title:   article.Title,
		Content: article.Content,
		Author:  domain.Author{Id: article.Author.Id, Name: article.Author.Name},
		Status:  domain.ArticleStatus(article.ArticleStatus),
		CTime:   article.Ctime,
		UTime:   article.Utime,
	}
}

func (a *ArticleServiceServer) toDao(article domain.Article) *artv1.Article {
	return &artv1.Article{
		Id:            article.Id,
		Title:         article.Title,
		Content:       article.Content,
		Author:        &artv1.Author{Id: article.Author.Id, Name: article.Author.Name},
		ArticleStatus: uint32(article.Status),
		Ctime:         article.CTime,
		Utime:         article.UTime,
	}
}

func (a *ArticleServiceServer) mustEmbedUnimplementedArticleServiceServer() {
	//TODO implement me
	panic("implement me")
}
