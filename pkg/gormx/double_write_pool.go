package gormx

import (
	"context"
	"database/sql"
	"errors"
	"github.com/ecodeclub/ekit/syncx/atomicx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 双装饰器模式
type DoubleWritePool struct {
	src     gorm.ConnPool
	dst     gorm.ConnPool
	pattern *atomicx.Value[string] //需要viper热配置的变量或者会改变的成员都要考虑并发安全
}

//想让你的connPool支持事务，需要实现以下两个接口其中一个：
// TxBeginner tx beginner
//type TxBeginner interface {
//	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
//}
//
// ConnPoolBeginner conn pool beginner
//type ConnPoolBeginner interface {
//	BeginTx(ctx context.Context, opts *sql.TxOptions) (ConnPool, error)
//}

// 在每次执行事务前都会执行，这里返回的CoonPool似乎要求实现sql.Tx接口
// Tx sql.Tx interface
//
//	type Tx interface {
//		ConnPool
//		TxCommitter
//		StmtContext(ctx context.Context, stmt *sql.Stmt) *sql.Stmt
//	}
func (d *DoubleWritePool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	p := d.pattern.Load()
	switch p {
	case PatternSrcFirst:
		//开启一个事务，这里可以这样断言是因为mysql底层的ConnPool实际上是一个*sql.DB
		srcTx, err := d.src.(gorm.TxBeginner).BeginTx(ctx, opts)
		if err != nil {
			return nil, err
		}
		dstTx, err := d.dst.(gorm.TxBeginner).BeginTx(ctx, opts)
		if err != nil {
			zap.L().Error("dst库开启事务失败", zap.Error(err))
		}
		return &DoubleWritePoolTx{srcTx, dstTx, p}, nil
	case PatternSrcOnly:
		srcTx, err := d.src.(gorm.TxBeginner).BeginTx(ctx, opts)
		if err != nil {
			return nil, err
		}
		return &DoubleWritePoolTx{src: srcTx, pattern: p}, nil
	case PatternDstFirst:
		//开启一个事务，这里可以这样断言是因为mysql底层的ConnPool实际上是一个*sql.DB
		dstTx, err := d.dst.(gorm.TxBeginner).BeginTx(ctx, opts)
		if err != nil {
			return nil, err
		}
		srcTx, err := d.src.(gorm.TxBeginner).BeginTx(ctx, opts)
		if err != nil {
			zap.L().Error("dst库开启事务失败", zap.Error(err))
		}
		return &DoubleWritePoolTx{srcTx, dstTx, p}, nil
	case PatternDstOnly:
		dstTx, err := d.dst.(gorm.TxBeginner).BeginTx(ctx, opts)
		if err != nil {
			return nil, err
		}
		return &DoubleWritePoolTx{dst: dstTx, pattern: p}, nil
	default:
		return nil, errors.New("未知的双写模式")
	}
}

const (
	PatternSrcOnly  = "SRC_ONLY"
	PatternSrcFirst = "SRC_FIRST"
	PatternDstOnly  = "DST_ONLY"
	PatternDstFirst = "DST_FIRST"
)

func (d *DoubleWritePool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	//请看statement结构体定义
	//type Stmt struct {
	//	// Immutable:
	//	db        *DB    // where we came from
	//	query     string // that created the Stmt
	//直接包含db与query语句的，而且还是私有成员，你没法返回一个双写statement
	return nil, errors.New("不支持prepare")
}

func (d *DoubleWritePool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	switch d.pattern.Load() {
	case PatternSrcOnly:
		return d.src.ExecContext(ctx, query, args...)
	case PatternSrcFirst:
		res, err := d.src.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		_, err = d.dst.ExecContext(ctx, query, args...)
		if err != nil {
			zap.L().Error("目标库双写失败", zap.Error(err))
		}
		return res, nil
	case PatternDstOnly:
		return d.dst.ExecContext(ctx, query, args...)
	case PatternDstFirst:
		res, err := d.dst.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		_, err = d.src.ExecContext(ctx, query, args...)
		if err != nil {
			zap.L().Error("目标库双写失败", zap.Error(err))
		}
		return res, nil
	default:
		//没法指定row的err
		panic("未知的双写模式")
	}
}

func (d *DoubleWritePool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	switch d.pattern.Load() {
	case PatternSrcFirst, PatternSrcOnly:
		return d.src.QueryContext(ctx, query, args...)
	case PatternDstOnly, PatternDstFirst:
		return d.dst.QueryContext(ctx, query, args...)
	default:
		//没法指定row的err
		panic("未知的双写模式")
	}
}

func (d *DoubleWritePool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	switch d.pattern.Load() {
	case PatternSrcFirst, PatternSrcOnly:
		return d.src.QueryRowContext(ctx, query, args...)
	case PatternDstOnly, PatternDstFirst:
		return d.dst.QueryRowContext(ctx, query, args...)
	default:
		//没法指定row的err
		panic("未知的双写模式")
	}
}

// 这里另外定义了一个结构体，专门用于处理事务，其实就只需要多实现commit和rollback操作，其他操作没变化，还是会走双写ctx
type DoubleWritePoolTx struct {
	src     *sql.Tx
	dst     *sql.Tx
	pattern string //这里不需要原子量，因为不会修改，不会修改就没有并发问题（常量有啥并发问题
}

// 提交前执行
func (d *DoubleWritePoolTx) Commit() error {
	switch d.pattern {
	case PatternSrcFirst:
		err := d.src.Commit()
		if err != nil {
			//源库提交失败，目标库就不提交了，只会产生更多的数据不一致，会让之后目标库被删
			return err
		}
		//dst甚至可以为nil，但是这里其实不会传nil，除非有bug
		if d.dst != nil {
			err = d.dst.Commit()
			if err != nil {
				zap.L().Error("", zap.Error(err))
			}
		}
		return nil
	case PatternSrcOnly:
		err := d.src.Commit()
		return err
	case PatternDstFirst:
		err := d.dst.Commit()
		if err != nil {
			return err
		}
		err = d.src.Commit()
		if err != nil {
			zap.L().Error("", zap.Error(err))
		}
		return nil
	case PatternDstOnly:
		err := d.dst.Commit()
		return err
	default:
		return errors.New("未知的双写模式")
	}
}

// 回滚前执行
func (d *DoubleWritePoolTx) Rollback() error {
	switch d.pattern {
	case PatternSrcFirst:
		err := d.src.Rollback()
		if err != nil {
			//源库回滚失败，目标库也不回滚了
			return err
		}
		err = d.dst.Rollback()
		if err != nil {
			zap.L().Error("", zap.Error(err))
		}
		return nil
	case PatternSrcOnly:
		err := d.src.Rollback()
		return err
	case PatternDstFirst:
		err := d.dst.Rollback()
		if err != nil {
			return err
		}
		err = d.src.Rollback()
		if err != nil {
			zap.L().Error("", zap.Error(err))
		}
		return nil
	case PatternDstOnly:
		err := d.dst.Rollback()
		return err
	default:
		return errors.New("未知的双写模式")
	}
}

func (d *DoubleWritePoolTx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, errors.New("不支持prepare")
}

func (d *DoubleWritePoolTx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	switch d.pattern {
	case PatternSrcOnly:
		return d.src.ExecContext(ctx, query, args...)
	case PatternSrcFirst:
		res, err := d.src.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		_, err = d.dst.ExecContext(ctx, query, args...)
		if err != nil {
			zap.L().Error("目标库双写失败", zap.Error(err))
		}
		return res, nil
	case PatternDstOnly:
		return d.dst.ExecContext(ctx, query, args...)
	case PatternDstFirst:
		res, err := d.dst.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		_, err = d.src.ExecContext(ctx, query, args...)
		if err != nil {
			zap.L().Error("目标库双写失败", zap.Error(err))
		}
		return res, nil
	default:
		//没法指定row的err
		panic("未知的双写模式")
	}
}

func (d *DoubleWritePoolTx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	switch d.pattern {
	case PatternSrcFirst, PatternSrcOnly:
		return d.src.QueryContext(ctx, query, args...)
	case PatternDstOnly, PatternDstFirst:
		return d.dst.QueryContext(ctx, query, args...)
	default:
		panic("未知的双写模式")
	}
}

func (d *DoubleWritePoolTx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	switch d.pattern {
	case PatternSrcFirst, PatternSrcOnly:
		return d.src.QueryRowContext(ctx, query, args...)
	case PatternDstOnly, PatternDstFirst:
		return d.dst.QueryRowContext(ctx, query, args...)
	default:
		panic("未知的双写模式")
	}
}
