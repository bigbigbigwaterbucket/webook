package fixer

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"learning_go/webook/internal/migrator"
	"learning_go/webook/internal/migrator/events"
	"time"
)

type Fixer[t migrator.Entity] struct {
	base    *gorm.DB
	target  *gorm.DB
	columns []string
}

func (f *Fixer[t]) Fix(ctx context.Context, evt events.InconsistentEvent) error {
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	var srcData t
	err := f.base.WithContext(dbCtx).Where("id = ?", evt.ID).First(&srcData).Error
	cancel()
	switch err {
	case nil:
		//upsert 解决双写阶段的并发问题
		return f.target.WithContext(dbCtx).Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns(f.columns)}). //指定哪些列需要更新，具体数值从create里取得
			Create(&srcData).Error
	case gorm.ErrRecordNotFound:
		return f.target.WithContext(dbCtx).Delete(&srcData, evt.ID).Error
	default:
		return err
	}
}

// 并发问题：base 和 target 在校验时候的数据（传到这的id），到你修复的时候就变了，但是这个问题没法完全解决
func (f *Fixer[T]) FixV1(ctx context.Context, evt events.InconsistentEvent) error {
	switch evt.Type {
	case events.InconsistentUnEqual,
		events.InconsistentTargetMissing:
		// 这边要插入
		var t T
		err := f.base.WithContext(ctx).
			Where("id =?", evt.ID).First(&t).Error
		switch err {
		case gorm.ErrRecordNotFound:
			// base 也删除了这条数据
			return f.target.WithContext(ctx).
				Where("id=?", evt.ID).Delete(new(T)).Error
		case nil:
			return f.target.Clauses(clause.OnConflict{
				// 这边要更新全部列
				DoUpdates: clause.AssignmentColumns(f.columns),
			}).Create(&t).Error
		default:
			return err
		}
		// 这边要更新
	case events.InconsistentBaseMissing:
		return f.target.WithContext(ctx).
			Where("id=?", evt.ID).Delete(new(T)).Error
	default:
		return errors.New("未知的不一致类型")
	}
}
