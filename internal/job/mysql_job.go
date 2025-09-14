package job

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/service"
	"time"
)

type Executor interface {
	Execute(ctx context.Context, job domain.Job) error
	RegisterFun(name string, fn func(ctx context.Context, job domain.Job)) error
}

type LocalExecutor struct {
	exeFuncs map[string]func(ctx context.Context, job domain.Job) error
}

func (l *LocalExecutor) RegisterFun(name string, fn func(ctx context.Context, job domain.Job) error) {
	//l.exeFuncs = append(l.exeFuncs, name, fn)  map类型不能用append添加元素
	l.exeFuncs[name] = fn
}

func (l *LocalExecutor) Execute(ctx context.Context, job domain.Job) error {
	fn, ok := l.exeFuncs[job.Exe]
	if !ok {
		return errors.New("未找到任务，是否注册？")
	}
	return fn(ctx, job)
}

type Schedular struct {
	svc     service.JobService
	exe     Executor
	limiter semaphore.Weighted //信号量限制执行任务的最大协程数
}

func (s *Schedular) Schedule(ctx context.Context) error {
	for {
		dbCtx, cancel := context.WithTimeout(ctx, time.Second)
		//在一秒内持续去尝试拿到任务
		defer cancel()
		jobRes, err := s.svc.Preempt(dbCtx)
		if err != nil {
			zap.L().Error("调度器抢占任务失败", zap.Error(err))
		}
		err = s.limiter.Acquire(ctx, 1)
		if err != nil {
			return err
		}
		go func() {
			//执行
			err = s.exe.Execute(ctx, jobRes)
			if err != nil {
				zap.L().Error("执行任务失败", zap.Error(err))
			}
			//释放
			err = jobRes.Cancel()
			if err != nil {
				zap.L().Error("释放任务失败", zap.Error(err))
			}
			//reset这里的io操作是固定的，因此可以自己设置超时时间，前面的job的执行时间在调度器这一层是无法确定的，因此没设置
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err = s.svc.ResetNextTime(ctx, jobRes)
			if err != nil {
				zap.L().Error("重置任务下一次执行时间失败", zap.Error(err))
			}
			s.limiter.Release(1)
		}()
	}
}
