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
	pattern atomicx.Value[string] //需要viper热配置的变量或者会改变的成员都要考虑并发安全
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
			zap.L().Error("")
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
			zap.L().Error("")
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
