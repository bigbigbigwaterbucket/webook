package ioc

import (
	"learning_go/webook/pkg/redisx"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedis() redis.Cmdable {
	type Config struct {
		Addr string `yaml:"addr"`
	}
	var config1 Config
	err := viper.UnmarshalKey("redis", &config1)
	if err != nil {
		panic(err)
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr: config1.Addr})
	redisClient.AddHook(redisx.NewPrometheusHook(prometheus.SummaryOpts{
		Namespace: "waterbucket", Subsystem: "user",
		Name: "redis", Help: "redis执行时间检测与是否命中检测",
		Objectives: map[float64]float64{
			0.5:  0.01,
			0.75: 0.01,
			0.9:  0.005,
			0.99: 0.001,
		},
	}))
	return redisClient
}
