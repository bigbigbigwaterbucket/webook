package async

import (
	"context"
	"errors"
	"learning_go/webook/internal/repository"
	"learning_go/webook/internal/service/sms"
	"learning_go/webook/pkg/ratelimit"
	"time"
)

type AsyncSmsServiceV0 struct {
	svcs      []sms.Service
	limiter   ratelimit.Limiter
	repo      repository.CodeRepository
	retryTime int
}

func (a *AsyncSmsServiceV0) ActAsync(ctx context.Context) (bool, error) {
	limited, err := a.limiter.Limit(ctx, "sms:async")
	return limited, err
}

// 这里的sms服务锁死成发送验证码的短信服务了，但由于repo只设置了code repo，所以暂时先这样
func (a *AsyncSmsServiceV0) Send(ctx context.Context, tpl string, args []string, number ...string) error {
	act, err := a.ActAsync(ctx)
	if err != nil {
		return err
	}
	if !act {
		//没有触发异步容错，轮询发送
		for _, svc := range a.svcs {
			err = svc.Send(ctx, tpl, args, number...)
			if err == nil {
				return nil
			}
			//不是特殊错误，错误触发点不需要传错误，在出口传就可以
		}
		return errors.New("短信服务商全部发送失败")
	} else {
		//触发异步容错，即限流或是服务商崩溃
		err = a.repo.CreateRetry(ctx, "login", args[0], number[0])
		if err != nil {
			return err
		}
		go func() {
			for i := 0; i < a.retryTime; i++ {
				for _, svc := range a.svcs {
					err = svc.Send(ctx, tpl, args, number...)
					if err == nil {
						err = a.repo.DeleteRetry(ctx, "login", number[0])
						return
					}
				}
				time.Sleep(100)
			}
		}()
		//这里不知道最后是发送成功还是没成功，可以返回特定error
		//也可以返回异步任务id，之后轮询异步任务的状态
		return errors.New("短信已进入异步发送队列")
	}
}
