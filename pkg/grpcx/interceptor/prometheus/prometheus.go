package prometheus

import (
	"context"
	"learning_go/webook/pkg/grpcx/interceptor"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type PrometheusInterceptorBuilder struct {
	interceptor.Builder
	summary   *prometheus.SummaryVec
	SubSystem string
	NameSpace string
}

func (i *PrometheusInterceptorBuilder) Build() grpc.UnaryServerInterceptor {
	i.summary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: i.NameSpace,
		Subsystem: i.SubSystem,
		Name:      "grpc_server_handler_cost",
		Objectives: map[float64]float64{
			0.5:   0.01,
			0.9:   0.01,
			0.95:  0.01,
			0.99:  0.001,
			0.999: 0.0001,
		}}, []string{"type", "service", "method", "peer", "code"})
	prometheus.MustRegister(i.summary)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		st := time.Now()
		defer func() {
			s, m := i.splitMethodName(info.FullMethod)
			dur := time.Since(st)
			if err != nil {
				stat, _ := status.FromError(err)
				i.summary.WithLabelValues("unary", s, m, i.PeerName(ctx), stat.Code().String()).Observe(float64(dur.Milliseconds()))
			} else {
				i.summary.WithLabelValues("unary", s, m, i.PeerName(ctx), "OK").Observe(float64(dur.Milliseconds()))
			}
		}()
		return handler(ctx, req)
	}
}

func (i *PrometheusInterceptorBuilder) splitMethodName(fullMethodName string) (string, string) {
	// /UserService/GetByID
	// /user.v1.UserService/GetByID
	fullMethodName = strings.TrimPrefix(fullMethodName, "/") // remove leading slash "/"
	if i := strings.Index(fullMethodName, "/"); i >= 0 {
		return fullMethodName[:i], fullMethodName[i+1:]
	}
	return "unknown", "unknown"
}
