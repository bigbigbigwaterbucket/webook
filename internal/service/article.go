package service

import (
	"context"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository"
)

type ArticleService interface {
	Save(ctx context.Context, article domain.Article) (int64, error)
}

type ArticleServiceI struct {
	repo repository.ArticleRepository
}

func NewArticleServiceI(repo repository.ArticleRepository) *ArticleServiceI {
	return &ArticleServiceI{repo: repo}
}

func (a *ArticleServiceI) Save(ctx context.Context, article domain.Article) (int64, error) {
	return a.repo.Create(ctx, article)
}
