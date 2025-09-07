package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

type MiddleWareBuilder struct {
	Namespace string
	Subsystem string
	Name      string //只有name是必备的
	Help      string //应该也是帮助识别一个业务的
	// 这一个实例名字，你可以考虑使用 本地 IP，
	// 又或者在启动的时候配置一个 ID
	InstanceID string
}

func (m *MiddleWareBuilder) Builder() gin.HandlerFunc {
	//在prometheus的监控指标中加一些信息，方便识别是哪个业务
	labelNames := []string{"method", "pattern", "status"}
	summary := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		//标记一个公司一个组的一个业务
		Namespace: m.Namespace,
		Subsystem: m.Subsystem,
		Name:      m.Name + "_resp_time",
		Help:      m.Help,
		//应该是标记是哪个实例，方便定位问题
		ConstLabels: map[string]string{
			"instanceId": m.InstanceID,
		},
		Objectives: map[float64]float64{
			//分位数:误差，越大误差要求越小
			0.5:   0.01,
			0.75:  0.01,
			0.90:  0.01,
			0.99:  0.001,
			0.999: 0.0001,
		},
	}, labelNames)
	//prometheus注册指标对象
	//MustRegister 的作用就是把你的指标对象加入 Prometheus 全局 registry，方便 promhttp.Handler() 收集
	prometheus.MustRegister(summary)
	//这里没用vector形式（vector就是可以在prometheus查询界面查status、pattern等特定属性的指标）这里不需要查
	//这里只是统计当前时刻“活跃/正在运行”的请求的数量
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		//这里标记业务不能重复哦
		Namespace: m.Namespace,
		Subsystem: m.Subsystem,
		Name:      m.Name + "_active_req",
		Help:      "活跃的请求数量", // 会作为注释放在prometheus的查询界面
		//应该是标记是哪个实例，方便定位问题
		ConstLabels: map[string]string{
			"instanceId": m.InstanceID,
		},
	})
	prometheus.MustRegister(gauge)
	return func(ctx *gin.Context) {
		stime := time.Now()
		gauge.Inc() //进入活跃阶段，正在处理请求数+1
		//中间件栈式调用，先让后面的路由处理函数处理完然后调用回来
		ctx.Next()
		runTime := time.Since(stime).Milliseconds()
		gauge.Dec() //处理完成，活跃数-1
		pattern := ctx.FullPath()
		if pattern == "" {
			//让测试知道这里不是bug
			pattern = "unknown"
		}
		//observe处填写响应时间，这个要你自己统计记录runtime
		summary.WithLabelValues(ctx.Request.Method, pattern, strconv.Itoa(ctx.Writer.Status())).Observe(float64(runTime))
	}
}
