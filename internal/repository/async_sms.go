package repository

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository/dao"
)

type AsyncSmsRepository interface {
	AddAsyncSms(ctx context.Context, sms domain.Sms) error
	PreemptWaitingSms(ctx context.Context) (domain.Sms, error)
	ReportScheduleResult(ctx context.Context, id int64, res bool) error
}

type AsyncSmsRepositoryI struct {
	dao dao.AsyncSmsDao
}

func (a *AsyncSmsRepositoryI) AddAsyncSms(ctx context.Context, sms domain.Sms) error {
	args, err := json.Marshal(sms.Args)
	if err != nil {
		zap.L().Error("参数序列化错误", zap.Error(err))
		return err
	}
	err = a.dao.Insert(ctx, dao.AsyncSms{TplId: sms.TplId, Phone: sms.Phone, RetryTime: sms.RetryTime, Args: args})
	return err
}

func (a *AsyncSmsRepositoryI) PreemptWaitingSms(ctx context.Context) (domain.Sms, error) {
	sms, err := a.dao.GetWaitingSms(ctx)
	if err != nil {
		return domain.Sms{}, err
	}
	var args []string
	err = json.Unmarshal(sms.Args, &args)
	return domain.Sms{Id: sms.Id, TplId: sms.TplId, Args: args, Phone: sms.Phone, RetryTime: sms.RetryTime}, err
}

func (a *AsyncSmsRepositoryI) ReportScheduleResult(ctx context.Context, id int64, res bool) error {
	if res {
		//发送成功
		return a.dao.MarkSuccess(ctx, id)
	} else {
		//发送失败
		return a.dao.MarkFailure(ctx, id)
	}
}
