package grpcx

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ecodeclub/ekit/queue"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CounterLimiter struct {
	cnt       *atomic.Int32
	threshold int32
}

func (c *CounterLimiter) NewServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		cnt := c.cnt.Add(1) //这里必须拿到+1后的临时值，否则并发会导致后续加一也影响当前请求
		//拦截器也是栈式调用
		defer func() {
			c.cnt.Add(-1)
		}()
		if cnt > c.threshold {
			return nil, status.Errorf(codes.ResourceExhausted, "限流")
		} else {
			return handler(ctx, req)
		}
	}
}

type WindowLimiter struct {
	dur       time.Duration
	cnt       *atomic.Int32
	lastStart time.Time
	threshold int32
}

func (c *WindowLimiter) NewServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		now := time.Now()
		if now.After(c.lastStart.Add(c.dur)) {
			//这种写法会导致窗口实际大小可能会比dur要大（请求触发时间晚于lastStart+dur那一时刻）但是用于"单"节点就问题不大
			c.lastStart = now
			c.cnt.Store(0)
		}
		cnt := c.cnt.Add(1)
		defer func() {
			c.cnt.Add(-1)
		}()
		if cnt > c.threshold {
			return nil, status.Errorf(codes.ResourceExhausted, "限流")
		}
		return handler(ctx, req)
	}
}

type SlidingWindowLimiter struct {
	dur       time.Duration
	queue     queue.PriorityQueue[time.Time]
	threshold int32
	mutex     *sync.Mutex
}

func (c *SlidingWindowLimiter) NewServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		now := time.Now()
		c.mutex.Lock()
		_ = c.queue.Enqueue(now)
		if c.queue.Len() <= int(c.threshold) {
			c.mutex.Unlock()
			return handler(ctx, req)
		} else {
			last := now.Add(-c.dur)
			for {
				t, _ := c.queue.Peek()
				if t.Before(last) {
					_, _ = c.queue.Dequeue()
				} else {
					break
				}
			}
			c.mutex.Unlock()
			if c.queue.Len() > int(c.threshold) {
				return nil, status.Errorf(codes.ResourceExhausted, "限流")
			}
		}
		//重复unlock会panic
		//c.mutex.Unlock()
		return handler(ctx, req)
	}
}

type TokenBucketLimiter struct {
	capacity  int
	interval  time.Duration
	bucket    chan struct{}
	closeChan chan struct{}
}

func (c *TokenBucketLimiter) NewServerInterceptor() grpc.UnaryServerInterceptor {
	ticker := time.NewTicker(c.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				select {
				//chan是固定大小，可能会溢出
				case c.bucket <- struct{}{}:
				default:
					//溢出
				}
			case <-c.closeChan:
				return
			}
		}
	}()
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		select {
		case <-c.bucket:
			return handler(ctx, req)
			//下面两种都可以，看你没令牌是否阻塞
		default:
			return nil, status.Errorf(codes.ResourceExhausted, "限流")
			//case <-ctx.Done():
			//	return nil, status.Errorf(codes.ResourceExhausted, "限流")
		}
	}
}

func (c *TokenBucketLimiter) Close() {
	c.closeChan <- struct{}{}
}
