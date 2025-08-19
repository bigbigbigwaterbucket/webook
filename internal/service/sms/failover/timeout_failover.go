package failover

import (
	"context"
	"learning_go/webook/internal/service/sms"
	"sync/atomic"
)

type TimeOutFailover struct {
	svcs []sms.Service
	idx  int32
	//连续超时次数
	cnt int32
	//阈值
	threshold int32
}

func (t *TimeOutFailover) Send(ctx context.Context, tpl string, args []string, number ...string) error {
	//load函数能保证我在读的时候，另一个线程同时写，保证我读到的要么是0，要么是x，不会是写入一半的值
	//保证如果另一个线程写完后，我一定能读到
	idx := atomic.LoadInt32(&t.idx)
	cnt := atomic.LoadInt32(&t.cnt)
	if cnt > t.threshold {
		newidx := (idx + 1) % int32(len(t.svcs))
		//原子操作：check and do something，不用多说，保证check and do这一个流程是上锁的，即读的时候就上锁，保证是指定值，再改
		if atomic.CompareAndSwapInt32(&t.idx, idx, newidx) {
			//是我修改了idx，计数归零
			atomic.StoreInt32(&t.cnt, 0)
		}
		//这里还是会有并发问题，考虑极端情况：当前线程改完newidx后，还没置零cnt，另一个线程拿到了cnt，又发现超时过多，那么又会对idx+1
		//所以这里的t.idx可能变成idx+2，无论这里用哪种，都会额外加一，略过一个服务
		//idx = t.idx
		idx = newidx
	}
	err := t.svcs[idx].Send(ctx, tpl, args, number...)
	switch err {
	//短信服务也需要go的context，用来控制是否超时
	case context.DeadlineExceeded:
		atomic.AddInt32(&t.cnt, 1)
		return err
	case nil:
		//发出去就断开连续超时次数了
		atomic.StoreInt32(&t.cnt, 0)
		return nil
	default:
		//这里可以自由发挥，比如自动换下一个服务
		//也可以直接return，让用户去重试
		return err
	}
}

func NewTimeLimitFailover(svcs []sms.Service, threshold int32) *TimeOutFailover {
	return &TimeOutFailover{svcs: svcs, cnt: 0, threshold: threshold, idx: 0}
}
