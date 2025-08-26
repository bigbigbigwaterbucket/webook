package article

import (
	"context"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository/dao"
)

type ArticleRepository interface {
	Create(ctx context.Context, art domain.Article) (int64, error)
	Update(ctx context.Context, article domain.Article) (int64, error)
}

type CachedArticleRepository struct {
	dao dao.ArticleDao
}

func NewCachedArticleRepository(dao dao.ArticleDao) *CachedArticleRepository {
	return &CachedArticleRepository{dao: dao}
}

func (c *CachedArticleRepository) Create(ctx context.Context, art domain.Article) (int64, error) {
	return c.dao.Insert(ctx, dao.Article{Id: art.Id, Title: art.Title, Content: art.Content, AuthorId: art.Author.Id})
}

func (c *CachedArticleRepository) Update(ctx context.Context, art domain.Article) (int64, error) {
	return c.dao.UpdateById(ctx, dao.Article{Id: art.Id, Title: art.Title, Content: art.Content, AuthorId: art.Author.Id})
}
