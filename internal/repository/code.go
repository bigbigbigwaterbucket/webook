package repository

import (
	"context"
	"learning_go/webook/internal/repository/cache"
)

type CodeRepository struct {
	codeCache *cache.CodeCache
}

func NewCodeRepository(cc *cache.CodeCache) *CodeRepository {
	return &CodeRepository{codeCache: cc}
}

func (repo *CodeRepository) Store(ctx context.Context, biz, phone, code string) error {
	return repo.codeCache.Set(ctx, biz, phone, code)
}

func (repo *CodeRepository) Verify(ctx context.Context, biz, phone, inputCode string) error {
	return repo.codeCache.Verify(ctx, biz, phone, inputCode)
}
