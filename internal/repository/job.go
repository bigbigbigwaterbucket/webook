package repository

import (
	"context"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository/dao"
	"time"
)

type JobRepository interface {
	Preempt(ctx context.Context) (domain.Job, error)
	Refresh(ctx context.Context)
	Release(ctx context.Context, id int64) error
}

type OnlyGORMJobRepository struct {
	dao dao.JobDao
}

func (o *OnlyGORMJobRepository) Refresh(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (o *OnlyGORMJobRepository) Preempt(ctx context.Context) (domain.Job, error) {
	jRes, err := o.dao.Preempt(ctx)
	if err != nil {
		return domain.Job{}, err
	}
	return domain.Job{Id: jRes.Id, Name: jRes.Name, NextTime: time.UnixMilli(jRes.NextTime),
		CTime: jRes.CTime, UTime: jRes.UTime}, nil
}
