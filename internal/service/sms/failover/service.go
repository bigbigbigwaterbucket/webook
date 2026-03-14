package failover

import (
	"context"
	"errors"
	"learning_go/webook/internal/service/sms"
	"log"
	"sync/atomic"
)

type FailoverSmsService struct {
	svcs []sms.Service
	idx  uint64
}

func NewFailoverSmsService(svcs []sms.Service) *FailoverSmsService {
	return &FailoverSmsService{idx: 0, svcs: svcs}
}

// 简单遍历
func (f *FailoverSmsService) Send(ctx context.Context, tpl string, args []string, number ...string) error {
	for _, svc := range f.svcs {
		err := svc.Send(ctx, tpl, args, number...)
		if err == nil {
			return nil
		}
		//日志记录，哪个服务发送失败了，报错什么
		log.Println(err)
	}
	return errors.New("全部服务都尝试过，都失败了")
}

// 轮询
func (f *FailoverSmsService) SendV1(ctx context.Context, tpl string, args []string, number ...string) error {
	idx := atomic.AddUint64(&f.idx, 1) //自增原子操作，加之前和之后自动加锁解锁
	length := uint64(len(f.svcs))
	for i := idx; i < idx+length; i++ {
		err := f.svcs[int(i%length)].Send(ctx, tpl, args, number...)
		switch err {
		case nil:
			return nil
			//整个流程超时或者被主动取消了
		case context.DeadlineExceeded, context.Canceled:
			return err
		default:
			log.Println(err)
		}
		log.Println(err)
	}
	return errors.New("全部服务都尝试过，都失败了")
}
