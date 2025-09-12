package article

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type ArticleDao interface {
	Insert(ctx context.Context, art Article) (int64, error)
	UpdateById(ctx context.Context, art Article) (int64, error)
	Sync(ctx context.Context, art Article) (int64, error)
	UpdateOrInsert(ctx context.Context, art PublishArticle) (int64, error)
	SyncStatus(ctx context.Context, id int64, uid int64, status uint8) error
	GetByAuthor(ctx context.Context, uid int64, offset int64, limit int64) ([]Article, error)
	GetByArticleId(ctx context.Context, aid int64) (Article, error)
	GetPubByArticleId(ctx context.Context, aid int64) (PublishArticle, error)
	GetByArticleIds(ctx context.Context, aids []int64) ([]Article, error)
	GetRankingList(ctx context.Context, startTime time.Time, offset int, topNum int) ([]Article, error)
}

type GormArticleDao struct {
	db *gorm.DB
}

func (g *GormArticleDao) GetRankingList(ctx context.Context, startTime time.Time, offset int, topNum int) ([]Article, error) {
	var res []Article
	//按更新时间降序排列，保证取的数据不会重复
	err := g.db.WithContext(ctx).Model(Article{}).Where("u_time < ?", startTime.UnixMilli()).Order("u_time desc").
		Offset(offset).Limit(topNum).Find(&res).Error //有offset关键字可以直接用
	return res, err
}

func (g *GormArticleDao) GetByArticleIds(ctx context.Context, aids []int64) ([]Article, error) {
	var articles []Article
	err := g.db.WithContext(ctx).Model(Article{}).Where("id in ?", aids).Find(&articles).Error
	if err != nil {
		return []Article{}, nil
	}
	return articles, err
}

func (g *GormArticleDao) GetPubByArticleId(ctx context.Context, aid int64) (PublishArticle, error) {
	var pubArt PublishArticle
	res := g.db.WithContext(ctx).Model(&PublishArticle{}).Where("id=?", aid).First(&pubArt)
	return pubArt, res.Error
}

func (g *GormArticleDao) GetByArticleId(ctx context.Context, aid int64) (Article, error) {
	var art Article
	res := g.db.WithContext(ctx).Model(&Article{}).Where("id=?", aid).First(&art)
	return art, res.Error
}

func (g *GormArticleDao) GetByAuthor(ctx context.Context, uid int64, offset int64, limit int64) ([]Article, error) {
	var arts []Article
	//会自动跳过20行
	//SELECT * FROM article WHERE author_id=? ORDER BY u_time DESC LIMIT 10 OFFSET 20;
	//设计order by语句时，最好让order by中的数据命中索引
	err := g.db.WithContext(ctx).Model(&Article{}).Where("author_id=?", uid).
		Offset(int(offset)).
		Limit(int(limit)).
		//升序排序 u_time ASC
		//混合 u_time ASC ,xxx DESC
		Order("u_time DESC").
		Find(&arts).Error
	return arts, err
}

func (g *GormArticleDao) SyncStatus(ctx context.Context, id int64, uid int64, status uint8) error {
	err := g.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		res := tx.WithContext(ctx).Model(&Article{}).Where("id = ? and author_id = ?", id, uid).
			Updates(map[string]any{"status": status, "u_time": now})
		//最好不要习惯性写.error，记得考虑一下RowsAffected属性！
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			zap.L().Warn("id与作者不匹配", zap.Int64("id", id), zap.Int64("uid", uid))
			return errors.New("id与作者不匹配")
		}
		err := tx.WithContext(ctx).Model(&PublishArticle{}).Where("id = ? and author_id = ?", id, uid).
			Updates(map[string]any{"status": status, "u_time": now}).Error
		return err
	})
	return err
}

// 在dao层处理事务，那么默认制作库与线上库是同库不同表了
func (g *GormArticleDao) Sync(ctx context.Context, article Article) (int64, error) {
	var (
		id  int64
		err error
	)
	// tx-->transaction
	//这里tx不会重新创建连接
	//gorm的事务闭包，gorm帮助管理事务的生命周期
	//commit、rollback、begin都不需要我们考虑
	err = g.db.Transaction(func(tx *gorm.DB) error {
		var err error
		//这里重新又封装了一下tx，为的是使用GormArticleDao定义好的方法
		gdao := NewGormArticleDao(tx)
		if article.Id > 0 {
			id, err = gdao.UpdateById(ctx, article)
		} else {
			id, err = gdao.Insert(ctx, article)
		}
		if err != nil {
			//让gorm自动回滚
			return err
		}
		//之前是值传递，article的id并没有被反向更新到sync函数的局部变量！！
		article.Id = id
		//操作线上库（表）
		id, err = gdao.UpdateOrInsert(ctx, PublishArticle{Article: article})
		return err
	})
	return id, err
}

func (g *GormArticleDao) UpdateOrInsert(ctx context.Context, art PublishArticle) (int64, error) {
	now := time.Now().UnixMilli()
	art.CTime = now
	art.UTime = now
	//创建字句，例如where、on duplicate key等等
	err := g.db.WithContext(ctx).Clauses(clause.OnConflict{
		//字句冲突可选项：
		//哪些列冲突时触发：
		//Columns: []clause.Column{clause.Column{Name: "id"}},
		//数据冲突时啥都不干
		//DoNothing: true,
		//数据冲突，且符合where条件时就会执行DoUpdates
		//Where: clause.Where{}
		DoUpdates: clause.Assignments(map[string]any{
			"title":   art.Title,
			"content": art.Content,
			"u_time":  art.UTime,
			"status":  art.Status,
		})}).Create(&art).Error
	//最终生成的子句: Insert xxx on duplicate key update xxx
	//这里不需要开启事务，因为是一条sql语句，正常不需要
	return art.Id, err
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
		"u_time":  art.UTime,
		"Status":  art.Status,
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
