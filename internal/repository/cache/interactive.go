package cache

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/redis/go-redis/v9"
	"learning_go/webook/internal/domain"
	"strconv"
	"time"
)

var (
	//go:embed lua/interactive_incr_cnt.lua
	luaIncrCnt string
)

const (
	fieldReadCnt    = "read_cnt"
	fieldCollectCnt = "collect_cnt"
	fieldLikeCnt    = "like_cnt"
)

type InteractiveCache interface {
	IncreaseReadCountIfPresent(ctx context.Context, biz string, bizId int64) error
	IncreaseLikeCountIfPresent(ctx context.Context, biz string, bizId int64) error
	DecreaseLikeCountIfPresent(ctx context.Context, biz string, bizId int64) error
	IncreaseCollectionCntIfPresent(ctx context.Context, biz string, bizId int64) error
	GetThreeByCache(ctx context.Context, biz string, bizId int64) (domain.Interactive, error)
	Set(ctx context.Context, biz string, bizId int64, interactive domain.Interactive) error
}

type RedisInteractiveCache struct {
	client redis.Cmdable
}

func NewRedisInteractiveCache(client redis.Cmdable) *RedisInteractiveCache {
	return &RedisInteractiveCache{client: client}
}

func (r *RedisInteractiveCache) Set(ctx context.Context, biz string, bizId int64, interactive domain.Interactive) error {
	err := r.client.HSet(ctx, r.key(biz, bizId),
		fieldReadCnt, interactive.ReadCnt,
		fieldLikeCnt, interactive.LikeCnt,
		fieldCollectCnt, interactive.CollectCnt).Err()
	if err != nil {
		return err
	}
	//设置hash表的过期时间
	return r.client.Expire(ctx, r.key(biz, bizId), time.Minute*15).Err()
}

func (r *RedisInteractiveCache) GetThreeByCache(ctx context.Context, biz string, bizId int64) (domain.Interactive, error) {
	// 如果使用 HMGet，即便缓存中没有对应的 key，也不会返回 error
	hmap, err := r.client.HGetAll(ctx, r.key(biz, bizId)).Result()
	if err != nil {
		return domain.Interactive{}, err
	}
	var res domain.Interactive
	//同事设置了key，但没设置参数...
	if len(hmap) == 0 {
		// 缓存不存在
		return domain.Interactive{}, ErrorKeyNotExist
	}
	//理论上来说这里没有err
	res.LikeCnt, _ = strconv.ParseInt(hmap[fieldLikeCnt], 10, 64)
	res.CollectCnt, _ = strconv.ParseInt(hmap[fieldCollectCnt], 10, 64)
	res.ReadCnt, _ = strconv.ParseInt(hmap[fieldReadCnt], 10, 64)
	return res, nil
}

func (r *RedisInteractiveCache) IncreaseCollectionCntIfPresent(ctx context.Context, biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt, []string{r.key(biz, bizId)}, fieldCollectCnt, 1).Err()
}

func (r *RedisInteractiveCache) IncreaseLikeCountIfPresent(ctx context.Context, biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt, []string{r.key(biz, bizId)}, fieldLikeCnt, 1).Err()
}

func (r *RedisInteractiveCache) DecreaseLikeCountIfPresent(ctx context.Context, biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt, []string{r.key(biz, bizId)}, fieldLikeCnt, -1).Err()
}

func (r *RedisInteractiveCache) IncreaseReadCountIfPresent(ctx context.Context, biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt, []string{r.key(biz, bizId)}, fieldReadCnt, 1).Err()
}

func (r *RedisInteractiveCache) key(biz string, bizId int64) string {
	return fmt.Sprintf("interactive:%s:%d", biz, bizId)
}
