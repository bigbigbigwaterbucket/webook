package dao

import (
	"context"
	"gorm.io/gorm"
	"time"
)

type ArticleDao interface {
	Insert(ctx context.Context, art Article) (int64, error)
}

type GormArticleDao struct {
	db *gorm.DB
}

func (g *GormArticleDao) Insert(ctx context.Context, art Article) (int64, error) {
	now := time.Now().UnixMilli()
	art.CTime = now
	art.UTime = now
	err := g.db.WithContext(ctx).Create(&art).Error //create实际上也会反过来赋值id
	return art.Id, err
}

func NewGormArticleDao(db *gorm.DB) *GormArticleDao {
	return &GormArticleDao{db: db}
}

// Article，制作库
type Article struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	//标题长度1024
	Title   string `gorm:"type=varchar(1024)"` //指定sql中的字段类型
	Content string `gorm:"type=BLOB"`
	//如何设计索引？
	//3、在帖子这里，有创作者查询自己创作的内容的情景
	//1、产品经理说，要按照创建时间倒叙排列
	//2、单独查询某一篇

	//以下创建了联合索引
	//关于索引设计对查询的加速，可以学学explain命令
	AuthorId int64 `gorm:"index=aid_ctime"`
	CTime    int64 `gorm:"index=aid_ctime"`
	UTime    int64
}
