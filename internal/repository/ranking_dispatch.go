package repository

import (
	"context"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository/cache"
)

type RankingRepository interface {
	ReplaceTopN(ctx context.Context, arts []domain.Article) error
}

type OnlyCachedRankingRepository struct {
	cache cache.RankingCache
}

func NewOnlyCachedRankingRepository(cache cache.RankingCache) *OnlyCachedRankingRepository {
	return &OnlyCachedRankingRepository{cache: cache}
}

func (o *OnlyCachedRankingRepository) ReplaceTopN(ctx context.Context, arts []domain.Article) error {
	return o.cache.Set(ctx, arts)
}
