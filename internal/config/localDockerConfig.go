//go:build !k8s

package config

var (
	Config = ConfigInfo{
		MysqlConfig{MysqlURL: "root:root@tcp(localhost:13316)/webook"},
		RedisConfig{RedisURL: "localhost:6379"}}
)
