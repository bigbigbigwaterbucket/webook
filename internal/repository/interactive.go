package repository

import (
	"context"
	"go.uber.org/zap"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository/cache"
	"learning_go/webook/internal/repository/dao"
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
}

type CachedInteractiveRepository struct {
	dao   dao.InteractiveDao
	cache cache.InteractiveCache
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
			zap.L().Error("修改阅读数缓存错误", zap.Error(er))
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
		er := c.cache.IncreaseLikeCountIfPresent(ctx, biz, bizId)
		if er != nil {
			zap.L().Error("修改阅读数缓存错误", zap.Error(er))
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
			zap.L().Error("修改阅读数缓存错误", zap.Error(er))
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
		LikeCnt:    interactive.LikeCnt,
		CollectCnt: interactive.CollectCnt,
		ReadCnt:    interactive.ReadCnt,
	}
}

func NewCachedInteractiveRepository(dao dao.InteractiveDao, cache cache.InteractiveCache) *CachedInteractiveRepository {
	return &CachedInteractiveRepository{dao: dao, cache: cache}
}
