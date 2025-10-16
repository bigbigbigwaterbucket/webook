package article

import (
	"context"
	"learning_go/webook/article/domain"
)

type ArticleReaderRepository interface {
	Save(ctx context.Context, art domain.Article) (int64, error)
}
