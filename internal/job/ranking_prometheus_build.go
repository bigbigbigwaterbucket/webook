package job

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type PrometheusJobBuilder struct {
	summary *prometheus.SummaryVec
	trace   trace.Tracer
}

func NewPrometheusJobBuilder() *PrometheusJobBuilder {
	summaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "waterbucket",
		Subsystem: "webook",
		Name:      "cron_job",
		Help:      "统计定时任务的执行情况",
		Objectives: map[float64]float64{
			0.5:  0.05,
			0.75: 0.05,
			0.9:  0.01,
			0.99: 0.001,
		},
	}, []string{"name", "success"})
	prometheus.MustRegister(summaryVec)
	return &PrometheusJobBuilder{summary: summaryVec, trace: otel.GetTracerProvider().Tracer("webook/internal/job")}
}

// builder模式结构体需要传递新增成员，被build对象作为函数参数传进来/返回
func (p *PrometheusJobBuilder) Build(job Job) CronJobFunc {
	name := job.Name()
	return func() error {
		var success bool
		start := time.Now()
		zap.L().Info("cron_job开始执行", zap.String("name", name))
		ctx, span := p.trace.Start(context.Background(), name)
		//ctx和span是绑定的，传ctx就会记录调用链路
		defer span.End()
		defer func() {
			dur := time.Since(start)
			zap.L().Info("cron_job执行结束", zap.String("name", name))
			p.summary.WithLabelValues(name, strconv.FormatBool(success)).Observe(float64(dur.Milliseconds()))
		}()
		err := job.Run(ctx)
		success = err == nil
		if err != nil {
			zap.L().Error("cron_job执行失败", zap.String("name", name))
		}
		return nil
	}
}

type CronJobFunc func() error

func (f CronJobFunc) Run() {
	_ = f()
}
