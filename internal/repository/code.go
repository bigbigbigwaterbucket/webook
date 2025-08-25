package repository

import (
	"context"
	"fmt"
	"learning_go/webook/internal/repository/cache"
	"learning_go/webook/internal/repository/dao"
)

type CodeRepositoryI struct {
	codeCache cache.CodeCache
	codeDao   dao.CodeDao
}

type CodeRepository interface {
	Store(ctx context.Context, biz, phone, code string) error
	Verify(ctx context.Context, biz, phone, inputCode string) error
	CreateRetry(ctx context.Context, biz, phone, code string) error
	DeleteRetry(ctx context.Context, biz, phone string) error
	FindByKey(ctx context.Context, biz, phone string) (string, error)
}

// 接口本身就是引用类型，可以直接接受结构体指针，并调用结构体实现的接口方法
func NewCodeRepository(cc cache.CodeCache, cd dao.CodeDao) *CodeRepositoryI {
	return &CodeRepositoryI{codeCache: cc, codeDao: cd}
}

func (repo *CodeRepositoryI) Store(ctx context.Context, biz, phone, code string) error {
	return repo.codeCache.Set(ctx, biz, phone, code)
}

func (repo *CodeRepositoryI) Verify(ctx context.Context, biz, phone, inputCode string) error {
	return repo.codeCache.Verify(ctx, biz, phone, inputCode)
}

func (repo *CodeRepositoryI) CreateRetry(ctx context.Context, biz, phone, code string) error {
	return repo.codeDao.Insert(ctx, dao.Code{Code: code, Key: repo.key(biz, phone)})
}

func (repo *CodeRepositoryI) DeleteRetry(ctx context.Context, biz, phone string) error {
	return repo.codeDao.Delete(ctx, repo.key(biz, phone))
}

func (repo *CodeRepositoryI) FindByKey(ctx context.Context, biz, phone string) (string, error) {
	code, err := repo.codeDao.FindByKey(ctx, repo.key(biz, phone))
	if err != nil {
		return "", err
	}
	return code.Code, err
}

func (repo *CodeRepositoryI) key(biz, phone string) string {
	return fmt.Sprintf("phone_code:%s:%s", biz, phone)
}
