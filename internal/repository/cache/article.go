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
	//只缓存创作库的第一页，并且不缓存Content
	GetFirstPage(ctx context.Context, uid int64) ([]domain.Article, error)
	SetFirstPage(ctx context.Context, uid int64, data []domain.Article) error
	DelFirstPage(ctx context.Context, id int64) error
	Set(ctx context.Context, art domain.Article) error
	Get(ctx context.Context, aid int64) (domain.Article, error)
	// 注意到即使是已经发表的文章，存进去和取出来的数据结构仍然是制作库的article，缓存只服务于前端，因此要便于前端展示
	GetPub(ctx context.Context, aid int64) (domain.Article, error)
	// SetPub 正常来说，创作者和读者的 Redis 集群要分开，因为读者是一个核心中的核心
	SetPub(ctx context.Context, aid int64, res domain.Article) error
}

type RedisArticleCache struct {
	client redis.Cmdable
}

func (r *RedisArticleCache) GetPub(ctx context.Context, aid int64) (domain.Article, error) {
	pArtByte, err := r.client.Get(ctx, r.readerKey(aid)).Bytes()
	if err != nil {
		return domain.Article{}, err
	}
	var art domain.Article
	err = json.Unmarshal(pArtByte, &art)
	return art, err
}

func (r *RedisArticleCache) SetPub(ctx context.Context, aid int64, res domain.Article) error {
	pArtByte, err := json.Marshal(res)
	if err != nil {
		return err
	}
	//这里的过期时间可以调整，例如根据帖主的粉丝量来决定，或者根据推广流量...
	//这里的时间与作品的预期受欢迎程度正相关
	return r.client.Set(ctx, r.readerKey(aid), pArtByte, time.Minute*30).Err()
}

func (r *RedisArticleCache) Get(ctx context.Context, aid int64) (domain.Article, error) {
	artByte, err := r.client.Get(ctx, r.authorKey(aid)).Bytes()
	if err != nil {
		return domain.Article{}, err
	}
	var art domain.Article
	err = json.Unmarshal(artByte, &art)
	return art, err
}

func (r *RedisArticleCache) Set(ctx context.Context, art domain.Article) error {
	artByte, err := json.Marshal(art)
	if err != nil {
		return err
	}
	//这里是预缓存，因此时间要短（预测list后创作者会点第一个文章
	return r.client.Set(ctx, r.authorKey(art.Id), artByte, time.Minute).Err()
}

func NewRedisArticleCache(client redis.Cmdable) *RedisArticleCache {
	return &RedisArticleCache{client: client}
}

func (r *RedisArticleCache) GetFirstPage(ctx context.Context, uid int64) ([]domain.Article, error) {
	dataByte, err := r.client.Get(ctx, r.FirstPageKey(uid)).Bytes()
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
	return r.client.Set(ctx, r.FirstPageKey(uid), dataByte, time.Minute*10).Err()
}

func (r *RedisArticleCache) DelFirstPage(ctx context.Context, id int64) error {
	return r.client.Del(ctx, r.FirstPageKey(id)).Err()
}

// 制作库第一篇文章缓存
func (r *RedisArticleCache) authorKey(aid int64) string {
	return fmt.Sprintf("articles:author:%d", aid)
}

// 线上库刚发表、刚查询的文章缓存
func (r *RedisArticleCache) readerKey(aid int64) string {
	return fmt.Sprintf("articles:reader:%d", aid)
}

func (r *RedisArticleCache) FirstPageKey(uid int64) string {
	return fmt.Sprintf("articles:first_page:%d", uid)
}
