package repository

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"go.uber.org/zap"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository/cache"
	"learning_go/webook/internal/repository/dao"
	"time"
)

type InteractiveRepository interface {
	IncreaseReadCount(ctx context.Context, biz string, bizId int64) error
	IncreaseLikeCnt(ctx context.Context, biz string, bizId int64, uid int64) error
	DecreaseLikeCnt(ctx context.Context, biz string, bizId int64, uid int64) error
	AddCollectItem(ctx context.Context, biz string, bizId int64, cid int64, uid int64) error
	GetThree(ctx context.Context, biz string, bizId int64) (domain.Interactive, error)
	Liked(ctx context.Context, biz string, bizId int64, uid int64) (bool, error)
	Collected(ctx context.Context, biz string, bizId int64, uid int64) (bool, error)
	IncreaseReadCountN(ctx context.Context, bizs []string, aids []int64) error
	GetLikeTop(ctx context.Context, biz string, topNum int64) ([]domain.Interactive, error)
	GetByIds(ctx context.Context, ids []int64) (map[int64]domain.Interactive, error)
}

type CachedInteractiveRepository struct {
	topDuration time.Duration //top数据缓存间隔，可配置
	topLast     time.Time     //top数据上一次缓存时间
	dao         dao.InteractiveDao
	cache       cache.InteractiveCache
}

func NewCachedInteractiveRepository(topDuration time.Duration, dao dao.InteractiveDao, cache cache.InteractiveCache) *CachedInteractiveRepository {
	return &CachedInteractiveRepository{topDuration: topDuration, dao: dao, cache: cache}
}

// 获取topN方案就是在redis里用zset维护10*N个数量的数据，然后每隔固定时间后，有人访问再去拉数据并更新缓存
func (c *CachedInteractiveRepository) GetLikeTop(ctx context.Context, biz string, topNum int64) ([]domain.Interactive, error) {
	var topData []domain.Interactive
	var res []dao.Interactive //注意res是10N个数据
	var err error
	//未赋值就是0值
	if c.topLast == (time.Time{}) || time.Since(c.topLast) > c.topDuration {
		res, err = c.dao.FindLikeTop(ctx, biz, topNum)
		topData = c.EntitysToDomains(res)
		//失败？重试吧
		err = c.cache.SetLikeTop(ctx, biz, topNum, topData)
		//别忘记更新缓存时间和减少数据量
		topData = topData[:topNum]
		c.topLast = time.Now()
	} else {
		//一定存在，保证redis过期时间长于duration，那么除非redis崩了，否则一定存在
		topData, err = c.cache.GetLikeTopMustPresent(ctx, biz, topNum)
	}
	return topData, err
}

func (c *CachedInteractiveRepository) IncreaseReadCountN(ctx context.Context, bizs []string, aids []int64) error {
	return c.dao.IncreaseReadCountN(ctx, bizs, aids)
}

func (c *CachedInteractiveRepository) Liked(ctx context.Context, biz string, bizId int64, uid int64) (bool, error) {
	_, err := c.dao.GetLikeInfo(ctx, biz, bizId, uid)
	switch err {
	case nil:
		return true, nil
	case dao.ErrNotFound:
		return false, nil
	default:
		return false, err
	}
}

func (c *CachedInteractiveRepository) Collected(ctx context.Context, biz string, bizId int64, uid int64) (bool, error) {
	_, err := c.dao.GetCollectInfo(ctx, biz, bizId, uid)
	switch err {
	case nil:
		return true, nil
	case dao.ErrNotFound:
		return false, nil
	default:
		return false, err
	}
}

func (c *CachedInteractiveRepository) GetThree(ctx context.Context, biz string, bizId int64) (domain.Interactive, error) {
	var res domain.Interactive
	res, err := c.cache.GetThreeByCache(ctx, biz, bizId)
	if err == nil {
		return res, nil
	}
	interEntity, err := c.dao.GetThreeByDao(ctx, biz, bizId)
	if err == nil {
		//这里还要回写一下缓存，缓存就是在这里首次设置的
		res = c.EntityToDomain(interEntity)
		if er := c.cache.Set(ctx, biz, bizId, res); er != nil {
			zap.L().Error("点赞收藏阅读回写缓存失败", zap.Error(er))
		}
		return res, nil
	}
	return domain.Interactive{}, err
}

func (c *CachedInteractiveRepository) AddCollectItem(ctx context.Context, biz string, bizId int64, cid int64, uid int64) error {
	err := c.dao.InsertCollectionItem(ctx, biz, bizId, cid, uid)
	if err != nil {
		return err
	}
	go func() {
		//这里不需要uid，只要存点赞量就可以
		er := c.cache.IncreaseCollectionCntIfPresent(ctx, biz, bizId)
		if er != nil {
			zap.L().Error("修改收藏数缓存错误", zap.Error(er))
		}
	}()
	return nil
}

func (c *CachedInteractiveRepository) IncreaseLikeCnt(ctx context.Context, biz string, bizId int64, uid int64) error {
	err := c.dao.IncreaseLikeCnt(ctx, biz, bizId, uid)
	if err != nil {
		return err
	}
	go func() {
		//这里不需要uid，只要存点赞量就可以
		er := c.cache.IncreaseLikeCountIfPresent(ctx, biz, bizId) //这里还会尝试开协程改topN的缓存
		if er != nil {
			zap.L().Error("修改点赞数缓存错误", zap.Error(er))
		}
	}()
	return nil
}

func (c *CachedInteractiveRepository) DecreaseLikeCnt(ctx context.Context, biz string, bizId int64, uid int64) error {
	err := c.dao.DecreaseLikeCnt(ctx, biz, bizId, uid)
	if err != nil {
		return err
	}
	go func() {
		//这里不需要uid，只要存点赞量就可以
		er := c.cache.DecreaseLikeCountIfPresent(ctx, biz, bizId)
		if er != nil {
			zap.L().Error("修改点赞数缓存错误", zap.Error(er))
		}
	}()
	return nil
}

func (c *CachedInteractiveRepository) IncreaseReadCount(ctx context.Context, biz string, bizId int64) error {
	//一般都会先保证数据库中数据的准确性，然后再操作redis缓存
	err := c.dao.IncreaseReadCount(ctx, biz, bizId)
	if err != nil {
		return err
	}
	//存在才更新，否则查询
	go func() {
		//异步去做，不严格关心缓存数据的准确性
		er := c.cache.IncreaseReadCountIfPresent(ctx, biz, bizId)
		if er != nil {
			zap.L().Error("修改阅读数缓存错误", zap.Error(er))
		}
	}()
	return err
}

func (c *CachedInteractiveRepository) EntityToDomain(interactive dao.Interactive) domain.Interactive {
	return domain.Interactive{
		Id:         interactive.Id,
		BizId:      interactive.BizId,
		LikeCnt:    interactive.LikeCnt,
		CollectCnt: interactive.CollectCnt,
		ReadCnt:    interactive.ReadCnt,
	}
}

func (c *CachedInteractiveRepository) EntitysToDomains(interactives []dao.Interactive) []domain.Interactive {
	return slice.Map[dao.Interactive, domain.Interactive](interactives, func(idx int, src dao.Interactive) domain.Interactive {
		return domain.Interactive{
			Id:         src.Id,
			BizId:      src.BizId,
			LikeCnt:    src.LikeCnt,
			CollectCnt: src.CollectCnt,
			ReadCnt:    src.ReadCnt,
		}
	})
}
