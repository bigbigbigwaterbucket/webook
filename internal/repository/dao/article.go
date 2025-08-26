package dao

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

type ArticleDao interface {
	Insert(ctx context.Context, art Article) (int64, error)
	UpdateById(ctx context.Context, article Article) (int64, error)
}

type GormArticleDao struct {
	db *gorm.DB
}

func (g *GormArticleDao) Insert(ctx context.Context, art Article) (int64, error) {
	now := time.Now().UnixMilli()
	art.CTime = now
	art.UTime = now
	//id不是由art传进来的，而是由数据库创建数据后得到id反传过来
	err := g.db.WithContext(ctx).Create(&art).Error //create实际上也会反过来赋值id
	return art.Id, err
}

func (g *GormArticleDao) UpdateById(ctx context.Context, art Article) (int64, error) {
	now := time.Now().UnixMilli()
	art.UTime = now
	//g.db.Updates(&art)
	//上述更新方法依赖gorm的零值，即art字段中为零值的不更新，，这样做法是不好的，因为不清晰实际更新的到底是什么
	//通过map更新，那么零值也会被写入到数据库，更清晰
	res := g.db.WithContext(ctx).Model(&art).Where("id=? and author_Id =?", art.Id, art.AuthorId).Updates(map[string]any{
		"Title":   art.Title,
		"Content": art.Content,
		"UTime":   art.UTime,
	})
	//在数据库这验证文章id与作者id是否匹配，防止别人篡改文章，相比在业务层查询并验证，性能更高（只查1次）
	if res.Error != nil {
		return art.Id, res.Error
	}
	//没有更新任何数据
	if res.RowsAffected == 0 {
		//id不对/作者id不对，正常前端不应该不出现这种情况
		zap.L().Warn("文章更新失败", zap.Error(fmt.Errorf("更新失败，创作者非法 aId:%d, authorId:%d", art.Id, art.AuthorId)))
		return art.Id, fmt.Errorf("更新失败，创作者非法 aId:%d, authorId:%d", art.Id, art.AuthorId)
	}
	return art.Id, nil
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
