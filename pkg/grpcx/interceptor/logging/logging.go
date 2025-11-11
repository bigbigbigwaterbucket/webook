package logging

import (
	"context"
	"fmt"
	"learning_go/webook/pkg/grpcx/interceptor"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type LoggingInterceptorBuilder struct {
	//下面这种外部传函数的日志输出方法，能够把日志的级别交给外部函数确定，拦截器这里不需要关心日志的级别
	//fn func(msg string, fields...logger.Field)
	interceptor.Builder
}

func (l *LoggingInterceptorBuilder) Build() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		start := time.Now()
		defer func() {
			if rec := recover(); rec != nil {
				switch recType := rec.(type) {
				case error:
					err = recType
				default:
					err = fmt.Errorf("%v", rec)
				}
			}
			dur := time.Since(start)
			fileds := []zap.Field{
				zap.Int64("cost", dur.Microseconds()),
				zap.String("type", "unary"), //unary一元调用，客户端发送一个请求，服务端接受一个响应
				zap.String("method", info.FullMethod),
				zap.String("peer_name", l.PeerName(ctx)),
				zap.String("peer_ip", l.PeerIP(ctx)),
			}
			if err != nil {
				st, _ := status.FromError(err)
				fileds = append(fileds, zap.String("code", st.Code().String()), zap.String("code_msg", st.Message()))
			}
			zap.L().Info("RCP请求", fileds...)
		}()
		resp, err = handler(ctx, req)
		return
	}
}
