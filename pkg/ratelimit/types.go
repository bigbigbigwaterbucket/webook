package ratelimit

import "context"

type Limiter interface {
	//key是限流对象，也是redis中的key
	//bool是否触发限流
	Limit(ctx context.Context, key string) (bool, error)
}
