package config

type ConfigInfo struct {
	MysqlConfig
	RedisConfig
}

type MysqlConfig struct {
	MysqlURL string
}

type RedisConfig struct {
	RedisURL string
}
