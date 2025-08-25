package smsratelimit

import (
	"context"
	"fmt"
	"learning_go/webook/internal/service/sms"
	"learning_go/webook/pkg/ratelimit"
)

// 提前预留
var errLimited = fmt.Errorf("触发限流")

type RateLimitSmsService struct {
	svc     sms.Service
	limiter ratelimit.Limiter
}

func NewRateLimitSmsService(svc sms.Service, limiter ratelimit.Limiter) *RateLimitSmsService {
	return &RateLimitSmsService{svc: svc, limiter: limiter}
}

func (s *RateLimitSmsService) Send(ctx context.Context, tpl string, args []string, number ...string) error {
	limited, err := s.limiter.Limit(ctx, "sms:mem")
	if err != nil {
		return fmt.Errorf("redis限流系统错误:%w", err)
	}
	if limited {
		return errLimited
	}
	//装饰器模式，可以在核心业务代码调用前写一些特性
	err = s.svc.Send(ctx, tpl, args, number...)
	//也可以在之后调用
	return err
}
