package gormx

import (
	"context"
	"database/sql"
	"gorm.io/gorm"
)

type WriteSplitPool struct {
	master gorm.ConnPool
	slaves []gorm.ConnPool
}

// 事务里的查询用的也是QueryContext，也就是你这里装饰的queryContext，所以查询也会走从集群
func (w *WriteSplitPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return w.master.(gorm.TxBeginner).BeginTx(ctx, opts)
}

func (w *WriteSplitPool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return w.master.PrepareContext(ctx, query)
}

func (w *WriteSplitPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return w.master.ExecContext(ctx, query)
}

func (w *WriteSplitPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	//读操作这里需要实现一个负载均衡策略，例如轮询、随机、加权轮询、加权随机、平滑加权轮询
	panic("implement me")
}

func (w *WriteSplitPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	//TODO implement me
	panic("implement me")
}
