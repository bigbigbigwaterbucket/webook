package service

import (
	"context"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository"
	"time"
)

type JobService interface {
	Preempt(ctx context.Context) error
}

type MySQLJobService struct {
	repo       repository.JobRepository
	refreshDur time.Duration
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
			m.repo.Refresh(ctx)
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
