package repository

import (
	"context"
	"learning_go/webook/internal/domain"
)

type RankingRepository interface {
	ReplaceTopN(ctx context.Context, arts []domain.Article) error
}
