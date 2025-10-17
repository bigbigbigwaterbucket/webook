package client

import (
	"context"
	"learning_go/webook/api/proto/gen/interactive/intrv1"
	"learning_go/webook/interactive/domain"
	"learning_go/webook/interactive/service"

	"github.com/ecodeclub/ekit/slice"
	"google.golang.org/grpc"
)

// LocalInteractiveServiceClient 为了实现灰度发布，把本地服务伪装成rpc服务
type LocalInteractiveServiceClient struct {
	svc service.InteractiveService
}

func NewLocalInteractiveServiceClient(svc service.InteractiveService) intrv1.InteractiveServiceClient { //grpc的client不用组合占位结构体（保持向后兼容的
	return &LocalInteractiveServiceClient{svc: svc}
}

func (l *LocalInteractiveServiceClient) IncreaseReadCount(ctx context.Context, in *intrv1.IncreaseReadCountReq, opts ...grpc.CallOption) (*intrv1.IncreaseReadCountResp, error) {
	err := l.svc.IncreaseReadCount(ctx, in.GetBiz(), in.GetBizId())
	return &intrv1.IncreaseReadCountResp{}, err
}

func (l *LocalInteractiveServiceClient) Like(ctx context.Context, in *intrv1.LikeReq, opts ...grpc.CallOption) (*intrv1.LikeResp, error) {
	err := l.svc.Like(ctx, in.GetAid(), in.GetUid(), in.GetBiz())
	return &intrv1.LikeResp{}, err
}

func (l *LocalInteractiveServiceClient) UnLike(ctx context.Context, in *intrv1.UnLikeReq, opts ...grpc.CallOption) (*intrv1.UnLikeResp, error) {
	err := l.svc.UnLike(ctx, in.GetAid(), in.GetUid(), in.GetBiz())
	return &intrv1.UnLikeResp{}, err
}

func (l *LocalInteractiveServiceClient) Get(ctx context.Context, in *intrv1.GetReq, opts ...grpc.CallOption) (*intrv1.GetResp, error) {
	inter, err := l.svc.Get(ctx, in.GetBiz(), in.GetBizId(), in.GetUid())
	if err != nil {
		return &intrv1.GetResp{}, err
	}
	return &intrv1.GetResp{Inter: l.toDao(inter)}, nil
}

func (l *LocalInteractiveServiceClient) Collect(ctx context.Context, in *intrv1.CollectReq, opts ...grpc.CallOption) (*intrv1.CollectResp, error) {
	err := l.svc.Collect(ctx, in.GetBiz(), in.GetBizId(), in.GetCid(), in.GetUid())
	return &intrv1.CollectResp{}, err
}

func (l *LocalInteractiveServiceClient) GetLikeTop(ctx context.Context, in *intrv1.GetLikeTopReq, opts ...grpc.CallOption) (*intrv1.GetLikeTopResp, error) {
	inters, err := l.svc.GetLikeTop(ctx, in.GetBiz(), in.GetTopNum())
	if err != nil {
		return &intrv1.GetLikeTopResp{}, err
	}
	return &intrv1.GetLikeTopResp{Inters: slice.Map[domain.Interactive, *intrv1.Interactive](inters, func(idx int, src domain.Interactive) *intrv1.Interactive {
		return l.toDao(src)
	})}, nil
}

func (l *LocalInteractiveServiceClient) GetByIds(ctx context.Context, in *intrv1.GetByIdsReq, opts ...grpc.CallOption) (*intrv1.GetByIdsResp, error) {
	interMap, err := l.svc.GetByIds(ctx, in.GetBiz(), in.GetIds())
	if err != nil {
		return &intrv1.GetByIdsResp{}, err
	}
	grpcRes := make(map[int64]*intrv1.Interactive, len(interMap))
	for key, intr := range interMap {
		grpcRes[key] = l.toDao(intr)
	}
	return &intrv1.GetByIdsResp{InterMaps: grpcRes}, nil
}

func (i *LocalInteractiveServiceClient) toDao(intr domain.Interactive) *intrv1.Interactive {
	return &intrv1.Interactive{
		Id:         intr.Id,
		Biz:        intr.Biz,
		BizId:      intr.BizId,
		CollectCnt: intr.CollectCnt,
		Collected:  intr.Collected,
		LikeCnt:    intr.LikeCnt,
		Liked:      intr.Liked,
		ReadCnt:    intr.ReadCnt}
}
