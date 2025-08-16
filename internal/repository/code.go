package repository

import (
	"context"
	"learning_go/webook/internal/repository/cache"
)

type CodeRepositoryI struct {
	codeCache cache.CodeCache
}

type CodeRepository interface {
	Store(ctx context.Context, biz, phone, code string) error
	Verify(ctx context.Context, biz, phone, inputCode string) error
}

// 接口本身就是引用类型，可以直接接受结构体指针，并调用结构体实现的接口方法
func NewCodeRepository(cc cache.CodeCache) *CodeRepositoryI {
	return &CodeRepositoryI{codeCache: cc}
}

func (repo *CodeRepositoryI) Store(ctx context.Context, biz, phone, code string) error {
	return repo.codeCache.Set(ctx, biz, phone, code)
}

func (repo *CodeRepositoryI) Verify(ctx context.Context, biz, phone, inputCode string) error {
	return repo.codeCache.Verify(ctx, biz, phone, inputCode)
}
