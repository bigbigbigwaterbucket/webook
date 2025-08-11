//编译tag，可以解决编译时同一个包内有重复包变量如何解决，必须放在最前面
//go:build k8s

package config

var (
	Config = ConfigInfo{
		MysqlConfig{MysqlURL: "root:root@tcp(webook-mysql:3308)/webook"},
		RedisConfig{RedisURL: "webook-redis:6380"}}
)
