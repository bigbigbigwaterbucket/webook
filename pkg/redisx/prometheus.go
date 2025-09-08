package redisx

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"net"
	"strconv"
	"time"
)

type PrometheusHook struct {
	vector *prometheus.SummaryVec
}

func NewPrometheusHook(opt prometheus.SummaryOpts) *PrometheusHook {
	vector := prometheus.NewSummaryVec(opt, []string{"cmd", "key_exist_or_not"})
	prometheus.MustRegister(vector)
	return &PrometheusHook{vector: vector}
}

// 在客户端与 Redis 建立 TCP 连接时触发
func (p *PrometheusHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

// 执行单个命令前触发
func (p *PrometheusHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		var err error
		start_time := time.Now()
		defer func() {
			dur := time.Since(start_time).Milliseconds()
			keyExist := err == redis.Nil
			p.vector.WithLabelValues(cmd.Name(), strconv.FormatBool(!keyExist)).Observe(float64(dur))
		}()
		err = next(ctx, cmd)
		return err
	}
}

// 在执行pipeline 命令（批量命令）时触发
func (p *PrometheusHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		return next(ctx, cmds)
	}
}
