package dao

import (
	"context"
	"gorm.io/gorm"
	"time"
)

type JobDao interface {
	Preempt(ctx context.Context) (Job, error)
}

type GORMJobDao struct {
	db *gorm.DB
}

func (G *GORMJobDao) Preempt(ctx context.Context) (Job, error) {
	// 高并发情况下，大部分都是陪太子读书
	// 100 个 goroutine
	// 要转几次？ 所有 goroutine 执行的循环次数加在一起是
	// 1+2+3+4 +5 + ... + 99 + 100  分别是第一次就抢到的，第二次就抢到的，第三次....
	// 特定一个 goroutine，最差情况下，要循环一百次
	// 因此最好一个实例只安排一个goroutine
	var res Job
	//这里是开一个额外goroutine来取任务，取不到就不会做其他事，因此可以写成无限取任务
	for {
		now := time.Now()
		err := G.db.WithContext(ctx).Model(&Job{}).Where("next_time > ? and status = ?", now.UnixMilli(), JobStatusWaiting).
			First(&res).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				//没找到，重试
				continue
			}
			//数据库崩了?，得返回
			return Job{}, err
		}
		//找到了，由于是check and do something模式，这里使用version机制实现的乐观锁来解决并发问题
		// 两个 goroutine 都拿到 id =1 的数据
		// 能不能用 utime?
		// 乐观锁，CAS 操作，compare AND Swap
		// 有一个很常见的面试刷亮点：就是用乐观锁取代 FOR UPDATE
		// 面试套路（性能优化）：曾将用了 FOR UPDATE =>性能差，还会有死锁 => 我优化成了乐观锁
		sqlRes := G.db.WithContext(ctx).Model(&Job{}).Where("id = ? and version = ?", res.Id, res.Version).
			Updates(map[string]any{
				"u_time": now.UnixMilli(),
				"status": JobStatusRunning,
			})
		//没找到不会报错
		if sqlRes.Error != nil {
			return Job{}, err
		}
		if sqlRes.RowsAffected == 0 {
			continue
		}
	}
}

type Job struct {
	Id       int64 `gorm:"primaryKey,autoIncrement"`
	Name     string
	Version  int64 `gorm:"index"`
	NextTime int64 `gorm:"index"`
	Status   int8
	CTime    int64
	UTime    int64
}

const (
	JobStatusUnknown = iota
	JobStatusWaiting
	JobStatusRunning
	JobStatusStop
)
