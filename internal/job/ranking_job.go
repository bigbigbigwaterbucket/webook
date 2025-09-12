package job

import (
	"context"
	rlock "github.com/gotomicro/redis-lock"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"learning_go/webook/internal/service"
	"sync"
	"time"
)

type RankingJob struct {
	cmd       *rlock.Client
	svc       service.RankingService
	timeout   time.Duration
	lock      *rlock.Lock
	localLock *sync.Mutex //防止续约协程与主协程覆写lock产生并发问题（续约失败导致r.lock = nil没清空，一直执行
	key       string
}

func NewRankingJob(client redis.Cmdable, svc service.RankingService, timeout time.Duration) *RankingJob {
	return &RankingJob{cmd: rlock.NewClient(client), key: "rlock:cron_job:ranking", svc: svc, timeout: timeout,
		localLock: &sync.Mutex{}} //显式初始化一下，增加可读性，不要依赖nil
}

func (r *RankingJob) Name() string {
	return "ranking" //知道哪个job出问题
}

func (r *RankingJob) Run(ctx context.Context) error {
	r.localLock.Lock()
	defer r.localLock.Unlock()
	if r.lock == nil {
		var err error
		//没锁，尝试抢锁
		ctxGetLock, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		//这里是设计冗余，ctxGetLock会控制超时，然后函数里还要传一个超时时间，他内部又会去控制超时
		r.lock, err = r.cmd.Lock(ctxGetLock, r.key, r.timeout, &rlock.FixIntervalRetry{
			Interval: time.Millisecond * 100,
			Max:      0, //无限重试
		}, time.Second)
		if err != nil {
			return err
		}
		go func() {
			er := r.lock.AutoRefresh(r.timeout/2, time.Second)
			if er != nil {
				zap.L().Error("热榜定时任务续约失败", zap.Error(er))
				return
			}
			//注意将拿到的锁清空，以防当前实例以为自己有锁一直去执行
			r.localLock.Lock()
			r.lock = nil
			r.localLock.Unlock()
		}()
	}

	//这里传的ctx是为了trace调用链路
	ctx2, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.svc.TopN(ctx2)
}
