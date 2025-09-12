package job

import "context"

// 暴露服务接口的一种方式，可以用于执行定时任务
type Job interface {
	Name() string
	Run(ctx context.Context) error
}
