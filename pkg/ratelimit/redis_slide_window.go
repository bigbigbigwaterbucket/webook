package ratelimit

import (
	"context"
	_ "embed" //想要用go:embed嵌入字符串必须导入这个包，前面加个_应该是因为是在注释里用到的,没有实际用到
	"github.com/redis/go-redis/v9"
	"time"
)

//go:embed slide_window.lua
var luaSlideWindow string

type RedisSlidingWindow struct {
	cmd redis.Cmdable
	//阈值
	rate int
	//窗口大小
	interval time.Duration
	//interval内可以处理rate个请求
}

func NewRedisSlidingWindow(cmd redis.Cmdable, rate int, interval time.Duration) *RedisSlidingWindow {
	return &RedisSlidingWindow{cmd: cmd, rate: rate, interval: interval}
}

func (r *RedisSlidingWindow) Limit(ctx context.Context, key string) (bool, error) {
	return r.cmd.Eval(ctx, luaSlideWindow, []string{key},
		r.interval.Milliseconds(), r.rate, time.Now().UnixMilli()).Bool()
}
