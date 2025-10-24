package circuitbreaker

import (
	"context"
	"math/rand"

	"github.com/go-kratos/aegis/circuitbreaker"
	"google.golang.org/grpc"
)

type CircuitBreakerInterceptorBuilder struct {
	breaker   circuitbreaker.CircuitBreaker
	threshold int
}

func (b *CircuitBreakerInterceptorBuilder) BuildServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		err = b.breaker.Allow()
		if err == nil {
			resp, er := handler(ctx, req)
			// 借助这个区判定是不是业务错误，默认调用的服务也是grpc微服务
			//s, ok :=status.FromError(er)
			//if s != nil && s.Code() == codes.Unavailable {
			//	b.breaker.MarkFailed()
			//} else {
			//
			//}
			if er != nil {
				//这里还可以区分是业务上的错误还是系统上的错误
				b.breaker.MarkFailed()
			} else {
				b.breaker.MarkSuccess()
			}
			return resp, er
		}
		//注意限流了也要标记错误
		b.breaker.MarkFailed()
		return nil, err
	}
}

// 自己实现基于随机数+阈值和prometheus判断是否出现故障的熔断机制
func (b *CircuitBreakerInterceptorBuilder) BuildServerInterceptorV1() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any,
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if !b.allow() {
			b.threshold = b.threshold / 2
			// 这里就是触发了熔断
			//b.threshold = 0
			//time.AfterFunc(time.Minute, func() {
			//	b.threshold = 1
			//})
		}
		// 下面就是随机数判定
		randi := rand.Intn(100)
		if randi <= b.threshold {
			resp, err = handler(ctx, req)
			if err == nil && b.threshold != 0 {
				//调大 threshold
			} else if b.threshold != 0 {
				//调小 threshold
			}
		}
		return
	}
}

func (b *CircuitBreakerInterceptorBuilder) allow() bool {
	// 这里有判定节点是否健康的各种做法
	// 从prometheus 里面拿数据判定
	// prometheus.DefaultGatherer.Gather()
	return false
}
