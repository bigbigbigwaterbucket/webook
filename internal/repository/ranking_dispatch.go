package repository

import (
	"context"
	"learning_go/webook/article/domain"
	"learning_go/webook/internal/repository/cache"

	"go.uber.org/zap"
)

type RankingRepository interface {
	ReplaceTopN(ctx context.Context, arts []domain.Article) error
	GetTopN(ctx context.Context) ([]domain.Article, error)
}

type OnlyCachedRankingRepository struct {
	//非面向接口编程，作用是提高可读性，不然这里容易混
	redisCache *cache.RedisRankingCache
	localCache *cache.LocalRankingCache
}

func NewOnlyCachedRankingRepository(redisCache *cache.RedisRankingCache,
	localCache *cache.LocalRankingCache) *OnlyCachedRankingRepository {
	return &OnlyCachedRankingRepository{redisCache: redisCache, localCache: localCache}
}

func (o *OnlyCachedRankingRepository) GetTopN(ctx context.Context) ([]domain.Article, error) {
	var res []domain.Article
	res, err := o.localCache.Get(ctx)
	if err == nil {
		return res, nil
	}
	zap.L().Error("本地热榜缓存未命中", zap.Error(err))
	res, err = o.redisCache.Get(ctx)
	if err == nil {
		return res, nil
	}
	zap.L().Error("redis热榜缓存未命中", zap.Error(err))
	return o.localCache.ForceGet(ctx)
}

func (o *OnlyCachedRankingRepository) ReplaceTopN(ctx context.Context, arts []domain.Article) error {
	_ = o.localCache.Set(ctx, arts)
	return o.redisCache.Set(ctx, arts)
}
