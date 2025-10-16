package article

import (
	"context"

	"gorm.io/gorm"
)

type ReaderDao interface {
	UpdateOrInsert(ctx context.Context, article PublishArticle) error
}

type ReaderDaoI struct {
	db *gorm.DB
}

func (r *ReaderDaoI) UpdateOrInsert(ctx context.Context, article Article) error {
	//TODO implement me
	panic("implement me")
}

func NewReaderDaoI(db *gorm.DB) *ReaderDaoI {
	return &ReaderDaoI{db: db}
}
