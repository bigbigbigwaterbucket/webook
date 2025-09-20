package startup

import (
	"context"
	"github.com/redis/go-redis/v9"
	"sync"
)

var (
	redisCmd  redis.Cmdable
	onceRedis sync.Once
)

func InitRedis() redis.Cmdable {
	url := "localhost:6379"
	onceRedis.Do(func() {
		redisCmd = redis.NewClient(&redis.Options{Addr: url})
		//不会循环的...
		for err := redisCmd.Ping(context.Background()).Err(); err != nil; {
			panic(err)
		}
	})
	return redisCmd
}
