package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"learning_go/webook/internal/domain"
	"time"
)

type ArticleCache interface {
	GetFirstPage(ctx context.Context, uid int64) ([]domain.Article, error)
	SetFirstPage(ctx context.Context, uid int64, data []domain.Article) error
	DelFirstPage(ctx context.Context, id int64) error
}

type RedisArticleCache struct {
	client redis.Cmdable
}

func (r *RedisArticleCache) GetFirstPage(ctx context.Context, uid int64) ([]domain.Article, error) {
	dataByte, err := r.client.Get(ctx, r.key(uid)).Bytes()
	if err != nil {
		zap.L().Info("缓存未命中", zap.Int64("uid", uid))
		return nil, err
	}
	var arts []domain.Article
	err = json.Unmarshal(dataByte, &arts)
	if err != nil {
		return nil, err
	}
	return arts, err
}

func (r *RedisArticleCache) SetFirstPage(ctx context.Context, uid int64, data []domain.Article) error {
	//记得把content清空
	for _, art := range data {
		art.Content = art.Abstract()
	}
	dataByte, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.key(uid), dataByte, time.Minute*10).Err()
}

func (r *RedisArticleCache) DelFirstPage(ctx context.Context, id int64) error {
	return r.client.Del(ctx, r.key(id)).Err()
}

func (r *RedisArticleCache) key(id int64) string {
	return fmt.Sprintf("articles:first_page:%d", id)
}
