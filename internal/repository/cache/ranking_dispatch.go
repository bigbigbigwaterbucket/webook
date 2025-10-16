package cache

import (
	"context"
	"encoding/json"
	"learning_go/webook/article/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

type RankingCache interface {
	Set(ctx context.Context, arts []domain.Article) error
	Get(ctx context.Context) ([]domain.Article, error)
}

type RedisRankingCache struct {
	client redis.Cmdable
	key    string
}

func NewRedisRankingCache(client redis.Cmdable, key string) *RedisRankingCache {
	return &RedisRankingCache{client: client, key: key}
}

func (r *RedisRankingCache) Set(ctx context.Context, arts []domain.Article) error {
	for i, _ := range arts {
		arts[i].Content = ""
	}
	data, err := json.Marshal(arts)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.key, data, time.Hour*24).Err() //过期时间一定要长于定时计算的时间，最好永不过期，因为是替换数据，如果数据库发生故障算不出新的，也有老的
}

//func a() *int {
//	return nil
//}
//
//func b() int {
//	return nil
//}

func (r *RedisRankingCache) Get(ctx context.Context) ([]domain.Article, error) {
	var res []domain.Article
	data, err := r.client.Get(ctx, r.key).Bytes() //Result()返回的是字符串类型的数据，Bytes()则会自动把你转为字节类型
	if err != nil {
		return nil, err //nil可以赋值给引用类型
	}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return []domain.Article{}, err
	}
	return res, nil
}
