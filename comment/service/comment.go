package service

import (
	"context"
	"learning_go/webook/comment/domain"
	"learning_go/webook/comment/repository"
)

type CommentService interface {
	//流式加载时，minId和offset的概念差不多，但是通过主键来进行分页可以用上索引加快查询速度
	GetCommentList(ctx context.Context, biz string, bizId int64, minId int64, limit int64) ([]domain.Comment, error)
	GetMoreReplies(ctx context.Context, rid int64, maxId int64, limit int64) ([]domain.Comment, error)
	CreateComment(ctx context.Context, comment domain.Comment) error
	DeleteComment(ctx context.Context, id int64) error
}

type CommentServiceI struct {
	repo repository.CachedCommentRepository
}

func (c *CommentServiceI) GetCommentList(ctx context.Context, biz string, bizId int64, minId int64, limit int64) ([]domain.Comment, error) {
	return c.repo.FindCommentByBiz(ctx, biz, bizId, minId, limit)
}

func (c *CommentServiceI) GetMoreReplies(ctx context.Context, rid int64, maxId int64, limit int64) ([]domain.Comment, error) {
	return c.repo.FindMoreReplies(ctx, rid, maxId, limit)
}

func (c *CommentServiceI) CreateComment(ctx context.Context, comment domain.Comment) error {
	return c.repo.Insert(ctx, comment)
}

func (c *CommentServiceI) DeleteComment(ctx context.Context, id int64) error {
	return c.repo.DeleteById(ctx, id)
}
