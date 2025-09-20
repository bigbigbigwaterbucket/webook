package grpc

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"learning_go/webook/api/proto/gen/interactive/v1"
	"learning_go/webook/interactive/domain"
	"learning_go/webook/interactive/service"
)

// 和grpc server端通信有关的操作限定在这
// 相当于web层的handler了，这里也是调用其他业务的微服务接口的地方
type InteractiveServiceServer struct {
	v1.UnimplementedInteractiveServiceServer
	svc service.InteractiveService
}

func NewInteractiveServiceServer(svc service.InteractiveService) *InteractiveServiceServer {
	return &InteractiveServiceServer{svc: svc}
}

func (i *InteractiveServiceServer) IncreaseReadCount(ctx context.Context, req *v1.IncreaseReadCountReq) (*v1.IncreaseReadCountResp, error) {
	//调用req的Getxxx方法更安全，他会判断req是否为nil，防止微服务使用者调错
	err := i.svc.IncreaseReadCount(ctx, req.GetBiz(), req.GetBizId())
	return &v1.IncreaseReadCountResp{}, err
}

func (i *InteractiveServiceServer) Like(ctx context.Context, req *v1.LikeReq) (*v1.LikeResp, error) {
	err := i.svc.Like(ctx, req.GetAid(), req.GetUid(), req.GetBiz())
	return &v1.LikeResp{}, err
}

func (i *InteractiveServiceServer) UnLike(ctx context.Context, req *v1.UnLikeReq) (*v1.UnLikeResp, error) {
	//grpc的错误处理机制
	if req.GetUid() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "uid 错误") //grpc的错误机制
	}
	err := i.svc.UnLike(ctx, req.GetAid(), req.GetUid(), req.GetBiz())
	return &v1.UnLikeResp{}, err
}

func (i *InteractiveServiceServer) Get(ctx context.Context, req *v1.GetReq) (*v1.GetResp, error) {
	res, err := i.svc.Get(ctx, req.GetBiz(), req.GetBizId(), req.GetUid())
	if err != nil {
		return nil, err
	}
	return &v1.GetResp{Inter: i.toDao(res)}, err
}

func (i *InteractiveServiceServer) Collect(ctx context.Context, req *v1.CollectReq) (*v1.CollectResp, error) {
	err := i.svc.Collect(ctx, req.GetBiz(), req.GetBizId(), req.GetCid(), req.GetUid())
	return &v1.CollectResp{}, err
}

func (i *InteractiveServiceServer) GetLikeTop(ctx context.Context, req *v1.GetLikeTopReq) (*v1.GetLikeTopResp, error) {
	res, err := i.svc.GetLikeTop(ctx, req.GetBiz(), req.GetTopNum())
	if err != nil {
		return nil, err
	}
	grpcRes := slice.Map[domain.Interactive, *v1.Interactive](res, func(idx int, src domain.Interactive) *v1.Interactive {
		return i.toDao(src)
	})
	//grpc的结构体都是传指针,repeated也是结构体指针数组
	return &v1.GetLikeTopResp{
		Inters: grpcRes,
	}, nil
}

func (i *InteractiveServiceServer) GetByIds(ctx context.Context, req *v1.GetByIdsReq) (*v1.GetByIdsResp, error) {
	res, err := i.svc.GetByIds(ctx, req.GetBiz(), req.GetIds())
	if err != nil {
		return nil, err
	}
	grpcRes := make(map[int64]*v1.Interactive, len(res))
	for key, intr := range res {
		grpcRes[key] = i.toDao(intr)
	}
	return &v1.GetByIdsResp{InterMaps: grpcRes}, nil
}

func (i *InteractiveServiceServer) toDao(intr domain.Interactive) *v1.Interactive {
	return &v1.Interactive{
		Id:         intr.Id,
		Biz:        intr.Biz,
		BizId:      intr.BizId,
		CollectCnt: intr.CollectCnt,
		Collected:  intr.Collected,
		LikeCnt:    intr.LikeCnt,
		Liked:      intr.Liked,
		ReadCnt:    intr.ReadCnt}
}
