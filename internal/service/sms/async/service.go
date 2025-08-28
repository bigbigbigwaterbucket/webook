package async

import (
	"context"
	"go.uber.org/zap"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository"
	"learning_go/webook/internal/service/sms"
	"time"
)

type AsyncSmsService struct {
	svc  sms.Service
	repo repository.AsyncSmsRepository
}

func NewAsyncSmsService(svc sms.Service, repo repository.AsyncSmsRepository) *AsyncSmsService {
	res := &AsyncSmsService{svc: svc, repo: repo}
	//在这里调用，一个web服务只有一个异步容错协程，百万并发下也有很大性能损耗
	res.StartSync()
	return res
}

func (a *AsyncSmsService) StartSync() {
	for {
		a.SendAsync()
	}
}

func (a *AsyncSmsService) SendAsync() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	smsA, err := a.repo.PreemptWaitingSms(ctx)
	//无论是否抢到，都取消定时ctx，ctx里有个计时器timer会抢占资源
	cancel()
	switch err {
	case nil:
		//当前服务的协程抢到了sms服务
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
		//正常与数据库/第三方服务交互时，都应该加个限时ctx，防止服务阻塞
		//这里已经是异步容错了，可能已经崩溃，限时ctx更有必要！！！
		defer cancel()
		err = a.svc.Send(ctx, smsA.TplId, smsA.Args, smsA.Phone)
		if err != nil {
			zap.L().Error("执行异步发送失败", zap.Error(err))
		}
		err = a.repo.ReportScheduleResult(ctx, smsA.Id, err == nil)
		if err != nil {
			zap.L().Error("异步发送汇报失败", zap.Error(err))
		}
	}
}

func (a *AsyncSmsService) Send(ctx context.Context, tpl string, args []string, number ...string) error {
	if a.needAsync() {
		err := a.repo.AddAsyncSms(ctx, domain.Sms{TplId: tpl, Args: args, Phone: number[0], RetryTime: 3})
		if err != nil {
			zap.L().Error("数据库存储异步短信失败")
		}
		return err
	} else {
		err := a.svc.Send(ctx, tpl, args, number...)
		return err
	}
}

func (a *AsyncSmsService) needAsync() bool {
	return true
}
