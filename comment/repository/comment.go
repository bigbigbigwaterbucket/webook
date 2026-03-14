package repository

import (
	"context"
	"database/sql"
	"learning_go/webook/comment/domain"
	"learning_go/webook/comment/repository/dao"
	"math"

	"github.com/ecodeclub/ekit/slice"
	"golang.org/x/sync/errgroup"
)

type CommentRepository interface {
	FindCommentByBiz(ctx context.Context, biz string, bizId int64, minId int64, limit int64) ([]domain.Comment, error)
	Insert(ctx context.Context, comment domain.Comment) error
	DeleteById(ctx context.Context, id int64) error
	FindMoreReplies(ctx context.Context, rid int64, maxId int64, limit int64) ([]domain.Comment, error)
}

type CachedCommentRepository struct {
	dao dao.CommentDao
}

func NewCachedCommentRepository(dao dao.CommentDao) CommentRepository {
	return &CachedCommentRepository{dao: dao}
}

func (c *CachedCommentRepository) FindCommentByBiz(ctx context.Context, biz string, bizId int64, minId int64, limit int64) ([]domain.Comment, error) {
	// 最新评论的缓存效果不是很好
	// 在这里缓存第一页，缓存咩有，就去找数据库
	// 也可以考虑定时刷新缓存
	// 拿到的就是顶级评论
	var daos []dao.CommentEntity
	daos, err := c.dao.FindCommentByBiz(ctx, biz, bizId, minId, limit)
	if err != nil {
		return nil, err
	}
	res := slice.Map[dao.CommentEntity, domain.Comment](daos, func(idx int, src dao.CommentEntity) domain.Comment {
		return c.toDomain(src)
	})
	var eg errgroup.Group
	for i, d := range daos {
		//go 1.22之后range循环会创建临时变量了
		//i := i
		//d := d
		//注意循环并发的变量冲突问题（第二个并发循环用的是上一个循环的临时变量
		eg.Go(func() error {
			var childs []dao.CommentEntity
			childs, er := c.dao.FindMoreRepliesByRid(ctx, d.Id, math.MaxInt64, 3)
			if er != nil {
				return er
			}
			res[i].ChildrenComment = slice.Map[dao.CommentEntity, domain.Comment](childs, func(idx int, src dao.CommentEntity) domain.Comment {
				return c.toDomain(src)
			})
			return nil
		})
	}
	return res, eg.Wait()
}

func (c *CachedCommentRepository) Insert(ctx context.Context, comment domain.Comment) error {
	return c.dao.Insert(ctx, c.toEntity(comment))
}

func (c *CachedCommentRepository) DeleteById(ctx context.Context, id int64) error {
	return c.dao.DeleteById(ctx, id)
}

// 详情
func (c *CachedCommentRepository) FindMoreReplies(ctx context.Context, rid int64, maxId int64, limit int64) ([]domain.Comment, error) {
	var res []dao.CommentEntity
	res, err := c.dao.FindMoreRepliesByPid(ctx, rid, maxId, limit)
	if err != nil {
		return nil, err
	}
	return slice.Map[dao.CommentEntity, domain.Comment](res, func(idx int, src dao.CommentEntity) domain.Comment {
		return c.toDomain(src)
	}), nil
}

// toDomain 不会填充子评论
func (c *CachedCommentRepository) toDomain(comment dao.CommentEntity) domain.Comment {
	rootId := int64(0)
	if comment.RootId.Valid {
		rootId = comment.RootId.Int64
	}
	pid := int64(0)
	if comment.PID.Valid {
		pid = comment.PID.Int64
	}
	return domain.Comment{Id: comment.Id, Uid: comment.Uid, Biz: comment.Biz, BizId: comment.BizId, Content: comment.Content,
		RootId: rootId, PID: pid, CTime: comment.CTime, UTime: comment.UTime}
}

// toEntity 不会填充子评论
func (c *CachedCommentRepository) toEntity(comment domain.Comment) dao.CommentEntity {
	rootId := sql.NullInt64{Valid: true, Int64: comment.RootId}
	pid := sql.NullInt64{Valid: true, Int64: comment.PID}
	return dao.CommentEntity{Id: comment.Id, Uid: comment.Uid, Biz: comment.Biz, BizId: comment.BizId, Content: comment.Content,
		RootId: rootId, PID: pid, CTime: comment.CTime, UTime: comment.UTime}
}
