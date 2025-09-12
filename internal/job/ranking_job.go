package job

import (
	"context"
	"learning_go/webook/internal/service"
	"time"
)

type RankingJob struct {
	svc     service.RankingService
	timeout time.Duration
}

func NewRankingJob(svc service.RankingService, timeout time.Duration) *RankingJob {
	return &RankingJob{svc: svc, timeout: timeout}
}

func (r *RankingJob) Name() string {
	return "ranking" //知道哪个job出问题
}

func (r *RankingJob) Run(ctx context.Context) error {
	//这里传的ctx是为了trace调用链路
	ctx2, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.svc.TopN(ctx2)
}
