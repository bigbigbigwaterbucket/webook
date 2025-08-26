package article

import (
	"context"
	"learning_go/webook/internal/domain"
)

type ArticleAuthorRepository interface {
	Create(ctx context.Context, art domain.Article) (int64, error)
	Update(ctx context.Context, article domain.Article) error
}
