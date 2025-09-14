package service

import (
	"context"
	"go.uber.org/zap"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository"
	"time"
)

type JobService interface {
	Preempt(ctx context.Context) (domain.Job, error)
	ResetNextTime(ctx context.Context, j domain.Job) error
}

type MySQLJobService struct {
	repo       repository.JobRepository
	refreshDur time.Duration
}

func (m *MySQLJobService) ResetNextTime(ctx context.Context, j domain.Job) error {
	nt := j.NextTime()
	if nt.IsZero() {
		//不再执行，停止任务
		return m.repo.Stop(ctx, j.Id)
	}
	return m.repo.UpdateNextTime(ctx, j.Id, nt)
}

func (m *MySQLJobService) Preempt(ctx context.Context) (domain.Job, error) {
	jobRes, err := m.repo.Preempt(ctx)
	if err != nil {
		return domain.Job{}, err
	}
	//续约任务放在这里做了
	ticker := time.NewTicker(m.refreshDur)
	go func() {
		for range ticker.C {
			refreshCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := m.repo.UpdateUTime(refreshCtx, jobRes.Id)
			if err != nil {
				zap.L().Error("刷新任务utime失败", zap.Error(err))
				return
			}
		}
	}()
	//任务执行完毕要取消
	jobRes.Cancel = func() error {
		ticker.Stop()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		return m.repo.Release(ctx, jobRes.Id)
	}
	return jobRes, nil
}
