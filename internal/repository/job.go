package repository

import (
	"context"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository/dao"
	"time"
)

type JobRepository interface {
	Preempt(ctx context.Context) (domain.Job, error)
	UpdateUTime(ctx context.Context, jid int64) error
	Release(ctx context.Context, jid int64) error
	UpdateNextTime(ctx context.Context, jid int64, nt time.Time) error
	Stop(ctx context.Context, jid int64) error
}

type OnlyGORMJobRepository struct {
	dao dao.JobDao
}

func (o *OnlyGORMJobRepository) Stop(ctx context.Context, jid int64) error {
	return o.dao.Stop(ctx, jid)
}

func (o *OnlyGORMJobRepository) UpdateNextTime(ctx context.Context, jid int64, nt time.Time) error {
	return o.dao.UpdateNextTime(ctx, jid, nt)
}

func (o *OnlyGORMJobRepository) Release(ctx context.Context, jid int64) error {
	return o.dao.ReleaseStatus(ctx, jid)
}

func (o *OnlyGORMJobRepository) UpdateUTime(ctx context.Context, jid int64) error {
	return o.dao.UpdateUTime(ctx, jid)
}

func (o *OnlyGORMJobRepository) Preempt(ctx context.Context) (domain.Job, error) {
	jRes, err := o.dao.Preempt(ctx)
	if err != nil {
		return domain.Job{}, err
	}
	return o.EntityToDomain(jRes), nil
}

func (o *OnlyGORMJobRepository) EntityToDomain(jRes dao.Job) domain.Job {
	return domain.Job{Id: jRes.Id, Name: jRes.Name,
		CTime: jRes.CTime, UTime: jRes.UTime, Exe: jRes.Exe, CronDurTime: jRes.CronDurTime}
}
