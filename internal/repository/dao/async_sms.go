package dao

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type AsyncSmsDao interface {
	Insert(ctx context.Context, sms AsyncSms) error
	GetWaitingSms(ctx context.Context) (AsyncSms, error)
	MarkSuccess(ctx context.Context, id int64) error
	MarkFailure(ctx context.Context, id int64) error
}

type GORMAsyncSmsDao struct {
	db *gorm.DB
}

func (G *GORMAsyncSmsDao) Insert(ctx context.Context, sms AsyncSms) error {
	now := time.Now().UnixMilli()
	sms.UTime = now
	sms.CTime = now
	return G.db.WithContext(ctx).Create(&sms).Error
}

func (G *GORMAsyncSmsDao) GetWaitingSms(ctx context.Context) (AsyncSms, error) {
	var sms AsyncSms
	//整数间互减，把time类型转换为相同单位(ms)的
	now := time.Now().UnixMilli()
	endTime := now - time.Minute.Milliseconds()
	err := G.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//关系型数据库中的锁是依附于事务的
		//只找1分钟前的短信发送
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Model(&sms).Where("u_time < ? and retry_time > 0 ", endTime).First(&sms).Error
		if err != nil {
			return err
		}
		//更新utime，考虑到前面的短信发送服务时间限制在2s，1分钟绝对够goroutine发完短信，之后还会标记成功/失败，保证不会出现重复发送问题
		return tx.Where("id = ?", sms.Id).Updates(map[string]any{"u_time": now}).Error
	})
	return sms, err
}

func (G *GORMAsyncSmsDao) MarkSuccess(ctx context.Context, id int64) error {
	now := time.Now().UnixMilli()
	return G.db.WithContext(ctx).Model(&AsyncSms{}).Where("id=?", id).Updates(map[string]any{"u_time": now, "retry_time": 0}).Error //重发成功，直接置0
}

func (G *GORMAsyncSmsDao) MarkFailure(ctx context.Context, id int64) error {
	now := time.Now().UnixMilli()
	return G.db.WithContext(ctx).Model(&AsyncSms{}).Where("id=?", id).Updates(map[string]any{"u_time": now, "retry_time": gorm.Expr("retry_time - 1")}).Error
}

type AsyncSms struct {
	Id        int64 `gorm:"primaryKey,autoIncrement"`
	TplId     string
	Args      []byte
	Phone     string
	RetryTime int64
	CTime     int64
	UTime     int64
}
