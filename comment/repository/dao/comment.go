package dao

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

type CommentDao interface {
	FindCommentByBiz(ctx context.Context, biz string, bizId int64, minId int64, limit int64) ([]CommentEntity, error)
	Insert(ctx context.Context, comment CommentEntity) error
	DeleteById(ctx context.Context, id int64) error
	//下面两个函数获取到的评论时间相反
	FindMoreRepliesByRid(ctx context.Context, rid int64, minId int64, limit int64) ([]CommentEntity, error)
	FindMoreRepliesByPid(ctx context.Context, pid int64, maxId int64, limit int64) ([]CommentEntity, error)
}

type GORMCommentDao struct {
	db *gorm.DB
}

func (G *GORMCommentDao) FindCommentByBiz(ctx context.Context, biz string, bizId int64, minId int64, limit int64) ([]CommentEntity, error) {
	var res []CommentEntity
	err := G.db.WithContext(ctx).Where("biz = ? and biz_id = ? and id < ? and root_id is NULL", biz, bizId, minId).Limit(int(limit)).Find(&res).Error
	return res, err
}

func (G *GORMCommentDao) Insert(ctx context.Context, comment CommentEntity) error {
	return G.db.WithContext(ctx).Create(&comment).Error
}

func (G *GORMCommentDao) DeleteById(ctx context.Context, id int64) error {
	return G.db.WithContext(ctx).Delete(&CommentEntity{Id: id}).Error
}

func (G *GORMCommentDao) FindMoreRepliesByRid(ctx context.Context, rid int64, minId int64, limit int64) ([]CommentEntity, error) {
	var res []CommentEntity
	//order确保顺序
	err := G.db.WithContext(ctx).Where("root_id = ? and id < ? ", rid, minId).Limit(int(limit)).Order("Id DESC").Find(&res).Error
	return res, err
}

func (G *GORMCommentDao) FindMoreRepliesByPid(ctx context.Context, pid int64, maxId int64, limit int64) ([]CommentEntity, error) {
	var res []CommentEntity
	//order确保顺序
	err := G.db.WithContext(ctx).Where("p_i_d = ? and id > ? ", pid, maxId).Limit(int(limit)).Order("Id ASC").Find(&res).Error
	return res, err
}

type CommentEntity struct {
	Id      int64 `gorm:"primaryKey,autoIncrement"`
	Uid     int64
	Biz     string `gorm:"index:biz_type"`
	BizId   int64  `gorm:"index:biz_type"`
	Content string
	RootId  sql.NullInt64 `gorm:"index"`
	//外键与引用两件套，mysql中只会存pid，gorm的预加载可以查出parentComment，不预加载就为nil
	PID           sql.NullInt64
	ParentComment *CommentEntity `gorm:"ForeignKey:PID;AssociationForeignKey:Id;constraint:OnDelete:CASCADE"`
	CTime         int64
	UTime         int64
}
