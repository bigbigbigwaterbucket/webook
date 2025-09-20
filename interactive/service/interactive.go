package service

import (
	"context"
	"golang.org/x/sync/errgroup"
	"learning_go/webook/interactive/domain"
	"learning_go/webook/interactive/repository"
)

type InteractiveService interface {
	IncreaseReadCount(ctx context.Context, biz string, bizId int64) error
	Like(ctx context.Context, aid int64, uid int64, biz string) error
	UnLike(ctx context.Context, aid int64, uid int64, biz string) error
	Get(ctx context.Context, biz string, bizId int64, uid int64) (domain.Interactive, error)
	Collect(ctx context.Context, biz string, bizId int64, cid int64, uid int64) error
	GetLikeTop(ctx context.Context, biz string, topNum int64) ([]domain.Interactive, error)
	GetByIds(ctx context.Context, biz string, ids []int64) (map[int64]domain.Interactive, error)
}
type InteractiveServiceI struct {
	repo repository.InteractiveRepository
}

func (i *InteractiveServiceI) GetByIds(ctx context.Context, biz string, ids []int64) (map[int64]domain.Interactive, error) {
	inters, err := i.repo.GetByIds(ctx, biz, ids)
	if err != nil {
		return map[int64]domain.Interactive{}, err
	}
	res := make(map[int64]domain.Interactive, len(inters))
	for _, inter := range inters {
		//注意这里是id而不是bizId
		res[inter.BizId] = inter
	}
	return res, nil
}

func (i *InteractiveServiceI) GetLikeTop(ctx context.Context, biz string, topNum int64) ([]domain.Interactive, error) {
	return i.repo.GetLikeTop(ctx, biz, topNum)
}

func (i *InteractiveServiceI) Get(ctx context.Context, biz string, bizId int64, uid int64) (domain.Interactive, error) {
	res, err := i.repo.GetThree(ctx, biz, bizId)
	if err != nil {
		//发生错误尽量别返回res，防止出现缺失数据
		return domain.Interactive{}, err
	}
	//这里检验一下uid，防止漏放进来？
	if uid > 0 {
		var eg errgroup.Group
		eg.Go(func() error {
			var er error
			res.Liked, er = i.repo.Liked(ctx, biz, bizId, uid)
			return er
		})
		eg.Go(func() error {
			var er error
			res.Collected, er = i.repo.Collected(ctx, biz, bizId, uid)
			return er
		})
		er2 := eg.Wait()
		if er2 != nil {
			return domain.Interactive{}, er2
		}
	}
	return res, nil

}

func (i *InteractiveServiceI) Collect(ctx context.Context, biz string, bizId int64, cid int64, uid int64) error {
	return i.repo.AddCollectItem(ctx, biz, bizId, cid, uid)
}

func NewInteractiveServiceI(repo repository.InteractiveRepository) InteractiveService {
	return &InteractiveServiceI{repo: repo}
}

func (i *InteractiveServiceI) IncreaseReadCount(ctx context.Context, biz string, bizId int64) error {
	return i.repo.IncreaseReadCount(ctx, biz, bizId)
}

func (i *InteractiveServiceI) Like(ctx context.Context, aid int64, uid int64, biz string) error {
	return i.repo.IncreaseLikeCnt(ctx, biz, aid, uid)
}

func (i *InteractiveServiceI) UnLike(ctx context.Context, aid int64, uid int64, biz string) error {
	return i.repo.DecreaseLikeCnt(ctx, biz, aid, uid)
}
