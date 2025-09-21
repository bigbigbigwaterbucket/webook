package client

import (
	"context"
	"github.com/ecodeclub/ekit/syncx/atomicx"
	"google.golang.org/grpc"
	"learning_go/webook/api/proto/gen/interactive/intrv1"
	"math/rand"
)

type GreyScaleInteractiveServiceClient struct {
	local  intrv1.InteractiveServiceClient
	remote intrv1.InteractiveServiceClient
	//使用随机数+阈值实现灰度发布
	// threshold 小于阈值%的将走本地
	threshold *atomicx.Value[int]
}

func NewGreyScaleInteractiveServiceClient(local, remote intrv1.InteractiveServiceClient) *GreyScaleInteractiveServiceClient {
	return &GreyScaleInteractiveServiceClient{local: local, remote: remote, threshold: atomicx.NewValue[int]()}
}

func (g *GreyScaleInteractiveServiceClient) UpdateThreshold(threshold int) {
	g.threshold.Store(threshold)
}

func (g *GreyScaleInteractiveServiceClient) client() intrv1.InteractiveServiceClient {
	th := g.threshold.Load()
	rd := rand.Int31n(100) //0-99
	if int(rd) < th {
		return g.local
	} else {
		return g.remote
	}
}

func (g *GreyScaleInteractiveServiceClient) IncreaseReadCount(ctx context.Context, in *intrv1.IncreaseReadCountReq, opts ...grpc.CallOption) (*intrv1.IncreaseReadCountResp, error) {
	return g.client().IncreaseReadCount(ctx, in, opts...)
}

func (g *GreyScaleInteractiveServiceClient) Like(ctx context.Context, in *intrv1.LikeReq, opts ...grpc.CallOption) (*intrv1.LikeResp, error) {
	return g.client().Like(ctx, in, opts...)
}

func (g *GreyScaleInteractiveServiceClient) UnLike(ctx context.Context, in *intrv1.UnLikeReq, opts ...grpc.CallOption) (*intrv1.UnLikeResp, error) {
	return g.client().UnLike(ctx, in, opts...)
}

func (g *GreyScaleInteractiveServiceClient) Get(ctx context.Context, in *intrv1.GetReq, opts ...grpc.CallOption) (*intrv1.GetResp, error) {
	return g.client().Get(ctx, in, opts...)
}

func (g *GreyScaleInteractiveServiceClient) Collect(ctx context.Context, in *intrv1.CollectReq, opts ...grpc.CallOption) (*intrv1.CollectResp, error) {
	return g.client().Collect(ctx, in, opts...)
}

func (g *GreyScaleInteractiveServiceClient) GetLikeTop(ctx context.Context, in *intrv1.GetLikeTopReq, opts ...grpc.CallOption) (*intrv1.GetLikeTopResp, error) {
	return g.client().GetLikeTop(ctx, in, opts...)
}

func (g *GreyScaleInteractiveServiceClient) GetByIds(ctx context.Context, in *intrv1.GetByIdsReq, opts ...grpc.CallOption) (*intrv1.GetByIdsResp, error) {
	return g.client().GetByIds(ctx, in, opts...)
}
