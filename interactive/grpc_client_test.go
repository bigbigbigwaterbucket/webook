package main

import (
	"context"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	v1 "learning_go/webook/api/proto/gen/interactive/v1"
	"net"
	"testing"
)

func TestClient(t *testing.T) {
	cc, err := grpc.Dial("localhost:8090", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return net.Dial("tcp", addr) // 强制直连，不走代理
		}),
	)
	require.NoError(t, err)
	client := v1.NewInteractiveServiceClient(cc)
	resp, err := client.Get(context.Background(), &v1.GetReq{
		Biz:   "article",
		BizId: 222,
	})
	require.NoError(t, err)
	t.Log(resp.Inter)
}
