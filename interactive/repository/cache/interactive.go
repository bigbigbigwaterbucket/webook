package cache

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"learning_go/webook/interactive/domain"
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
	SetLikeTop(ctx context.Context, biz string, topNum int64, topData []domain.Interactive) error
	GetLikeTopMustPresent(ctx context.Context, biz string, topNum int64) ([]domain.Interactive, error)
}

type RedisInteractiveCache struct {
	client redis.Cmdable
}

func (r *RedisInteractiveCache) GetLikeTopMustPresent(ctx context.Context, biz string, topNum int64) ([]domain.Interactive, error) {
	topRedisData, err := r.client.ZRevRangeWithScores(ctx, r.topKey(biz), 0, topNum-1).Result()
	if err != nil {
		return []domain.Interactive{}, err
	}
	topData := slice.Map[redis.Z, domain.Interactive](topRedisData, func(idx int, src redis.Z) domain.Interactive {
		bizId, ok := src.Member.(int64)
		if !ok {
			//redis数据存错了,,,
			return domain.Interactive{}
		}
		return domain.Interactive{
			Biz:     biz,
			BizId:   bizId,
			LikeCnt: int64(src.Score),
		}
	})
	return topData, nil
}

func (r *RedisInteractiveCache) SetLikeTop(ctx context.Context, biz string, topNum int64, topData []domain.Interactive) error {
	topRedisData := slice.Map[domain.Interactive, redis.Z](topData, func(idx int, src domain.Interactive) redis.Z {
		return redis.Z{
			Score:  float64(src.LikeCnt),
			Member: src.BizId, //这里其实只需要存bizId，而且increase业务用的也只是bizId，后续有需要再改
		}
	})
	return r.client.ZAdd(ctx, r.topKey(biz), topRedisData...).Err()
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
	//redis本质上只支持字符串类型，尽管redis对象有string、list、hash、set、zset，但底层只支持string类型
	res.LikeCnt, _ = strconv.ParseInt(hmap[fieldLikeCnt], 10, 64)
	res.CollectCnt, _ = strconv.ParseInt(hmap[fieldCollectCnt], 10, 64)
	res.ReadCnt, _ = strconv.ParseInt(hmap[fieldReadCnt], 10, 64)
	return res, nil
}

func (r *RedisInteractiveCache) IncreaseCollectionCntIfPresent(ctx context.Context, biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt, []string{r.key(biz, bizId)}, fieldCollectCnt, 1).Err()
}

// TODO:现在你还需要增加topN数据的like数，这个函数是会被并发调用的，因此你需要写lua脚本来实现并发安全。。。
func (r *RedisInteractiveCache) IncreaseLikeCountIfPresent(ctx context.Context, biz string, bizId int64) error {
	err := r.client.Eval(ctx, luaIncrCnt, []string{r.key(biz, bizId)}, fieldLikeCnt, 1).Err()
	//这里很有可能是err，因为用户访问的绝大多数不是topN文章
	go func() {
		er := r.client.ZIncrBy(ctx, r.topKey(biz), 1, strconv.FormatInt(bizId, 10)).Err()
		if er != nil {
			zap.L().Error("topN like数缓存修改失败", zap.Error(er))
		}
	}()
	return err
}

func (r *RedisInteractiveCache) DecreaseLikeCountIfPresent(ctx context.Context, biz string, bizId int64) error {
	err := r.client.Eval(ctx, luaIncrCnt, []string{r.key(biz, bizId)}, fieldLikeCnt, -1).Err()
	if err != nil {
		return err
	}
	return r.client.ZIncrBy(ctx, r.topKey(biz), -1, strconv.FormatInt(bizId, 10)).Err()
}

func (r *RedisInteractiveCache) IncreaseReadCountIfPresent(ctx context.Context, biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt, []string{r.key(biz, bizId)}, fieldReadCnt, 1).Err()
}

func (r *RedisInteractiveCache) key(biz string, bizId int64) string {
	return fmt.Sprintf("interactive:%s:%d", biz, bizId)
}

// 还是不能放topKey，否则增加like数的业务找不到具体哪个key存储top数
func (r *RedisInteractiveCache) topKey(biz string) string {
	return fmt.Sprintf("interactive:%s:top", biz)
}
