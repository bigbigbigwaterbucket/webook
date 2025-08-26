package service

import (
	"context"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository/article"
)

type ArticleService interface {
	Save(ctx context.Context, article domain.Article) (int64, error)
	Publish(ctx *gin.Context, art domain.Article) error
}

type ArticleServiceI struct {
	repo   article.ArticleRepository
	author article.ArticleAuthorRepository
	reader article.ArticleReaderRepository
}

func (a *ArticleServiceI) Publish(ctx *gin.Context, art domain.Article) error {
	var (
		aid int64
		err error
	)
	//这个函数应当是一个“事务”，不应该执行一半就停下来，应当具有原子性
	//为了维护原子性，这里在下边多了个重试机制，实际上还可以加指数退避
	//实际上也可以通过使用MQ实现异步容错机制
	//更高级的：使用cannal
	if art.Author.Id > 0 {
		err = a.author.Update(ctx, art)
		if err != nil {
			return err
		}
	} else {
		aid, err = a.author.Create(ctx, art)
		if err != nil {
			return err
		}
	}
	art.Id = aid
	//重试机制
	for i := 0; i < 3; i++ {
		aid, err = a.reader.Save(ctx, art)
		if err == nil {
			return nil
		}
		zap.L().Error("部分失败:保存到线上库重试失败", zap.Int64("aid", art.Id), zap.Error(err))
	}
	if err != nil {
		zap.L().Error("部分失败:保存到线上库彻底失败", zap.Int64("aid", art.Id), zap.Error(err))
	}
	return err
}

func NewArticleServiceI(repo article.ArticleRepository) *ArticleServiceI {
	return &ArticleServiceI{repo: repo}
}

func (a *ArticleServiceI) Save(ctx context.Context, article domain.Article) (int64, error) {
	if article.Id > 0 {
		return a.repo.Update(ctx, article)
	}
	return a.repo.Create(ctx, article)
}
