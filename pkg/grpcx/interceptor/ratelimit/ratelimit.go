package ratelimit

import (
	"context"
	"learning_go/webook/pkg/ratelimit"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RatelimitInterceptorBuilder struct {
	limiter ratelimit.Limiter
	key     string
}

// 在服务端限流
func (b *RatelimitInterceptorBuilder) BuildServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		limit, er := b.limiter.Limit(ctx, b.key)
		if er != nil {
			zap.L().Error("redis限流出错", zap.Error(er))
			//保守策略:仍然限流
			return nil, status.Errorf(codes.ResourceExhausted, "限流")
		}
		if limit {
			return nil, status.Errorf(codes.ResourceExhausted, "限流")
		}
		return handler(ctx, req)
	}
}

// 在客户端限流
func (b *RatelimitInterceptorBuilder) BuildClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		limit, er := b.limiter.Limit(ctx, b.key)
		if er != nil {
			zap.L().Error("redis限流出错", zap.Error(er))
			//保守策略:仍然限流
			return status.Errorf(codes.ResourceExhausted, "限流")
		}
		if limit {
			return status.Errorf(codes.ResourceExhausted, "限流")
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// 针对特定服务进行限流，服务级限流
func (b *RatelimitInterceptorBuilder) BuildServerInterceptorService() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if strings.HasPrefix(info.FullMethod, "/UserService") {
			limit, er := b.limiter.Limit(ctx, "limiter:service:user:UserService")
			if er != nil {
				zap.L().Error("redis限流出错", zap.Error(er))
				//保守策略:仍然限流
				return nil, status.Errorf(codes.ResourceExhausted, "限流")
			}
			if limit {
				return nil, status.Errorf(codes.ResourceExhausted, "限流")
			}
		}
		return handler(ctx, req)
	}
}
