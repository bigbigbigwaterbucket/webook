package dao

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type InteractiveDao interface {
	IncreaseReadCount(ctx context.Context, biz string, bizId int64) error
	IncreaseLikeCnt(ctx context.Context, biz string, bizId int64, uid int64) error
	DecreaseLikeCnt(ctx context.Context, biz string, bizId int64, uid int64) error
	InsertCollectionItem(ctx context.Context, biz string, bizId int64, cid int64, uid int64) error
	GetThreeByDao(ctx context.Context, biz string, bizId int64) (Interactive, error)
	GetLikeInfo(ctx context.Context, biz string, bizId int64, uid int64) (UserLikeBiz, error)
	GetCollectInfo(ctx context.Context, biz string, bizId int64, uid int64) (UserCollectionBiz, error)
}

type GORMInteractiveDao struct {
	db *gorm.DB
}

func (G *GORMInteractiveDao) GetCollectInfo(ctx context.Context, biz string, bizId int64, uid int64) (UserCollectionBiz, error) {
	var res UserCollectionBiz
	err := G.db.WithContext(ctx).Model(&UserCollectionBiz{}).Where("biz = ? and biz_id = ? and uid = ?",
		biz, bizId, uid).First(&res).Error
	return res, err
}

func (G *GORMInteractiveDao) GetLikeInfo(ctx context.Context, biz string, bizId int64, uid int64) (UserLikeBiz, error) {
	var res UserLikeBiz
	err := G.db.WithContext(ctx).Model(&UserLikeBiz{}).Where("biz = ? and biz_id = ? and uid = ? and status = ?",
		biz, bizId, uid, 1).First(&res).Error
	return res, err
}

func (G *GORMInteractiveDao) GetThreeByDao(ctx context.Context, biz string, bizId int64) (Interactive, error) {
	var res Interactive
	err := G.db.WithContext(ctx).Model(&Interactive{}).Where("biz = ? and biz_id = ?", biz, bizId).First(&res).Error
	if err != nil {
		//实际上第一次查看文章就需要通过GetThreeByDao接口获取dao的三维数据，但是这里却没有初始化，这就很坑
		//这里应该才是最先初始化点赞、收藏、阅读量的地方
		if err == ErrNotFound {
			now := time.Now().UnixMilli()
			er := G.db.WithContext(ctx).Model(&Interactive{}).Create(&Interactive{
				Biz:        biz,
				BizId:      bizId,
				ReadCnt:    0,
				CollectCnt: 0,
				LikeCnt:    0,
				CTime:      now,
				UTime:      now,
			}).Error
			if er != nil {
				return Interactive{}, er
			}
			return Interactive{Biz: biz, BizId: bizId, CTime: now, UTime: now}, nil
		}
		return Interactive{}, err
	}
	return res, err
}

func (G *GORMInteractiveDao) InsertCollectionItem(ctx context.Context, biz string, bizId int64, cid int64, uid int64) error {
	now := time.Now().UnixMilli()
	return G.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&Interactive{}).Where("biz = ? biz_id = ?", biz, bizId).Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]interface{}{
				"u_time":      now,
				"collect_cnt": gorm.Expr("collect_cnt + 1"),
			})}).Create(&Interactive{
			Biz:        biz,
			BizId:      bizId,
			ReadCnt:    0,
			CollectCnt: 1,
			LikeCnt:    0,
			CTime:      now,
			UTime:      now,
		}).Error
		if err != nil {
			return err
		}
		return tx.Model(&UserCollectionBiz{}).Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(
				map[string]interface{}{
					"u_time": now,
					"cid":    cid,
					"uid":    uid,
				},
			),
		}).Create(&UserCollectionBiz{
			Uid:   uid,
			BizId: bizId,
			Biz:   biz,
			Cid:   cid,
			CTime: now,
			UTime: now,
		}).Error
	})
}

func (G *GORMInteractiveDao) DecreaseLikeCnt(ctx context.Context, biz string, bizId int64, uid int64) error {
	now := time.Now().UnixMilli()
	return G.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&Interactive{}).Where("biz = ? and biz_id = ?", biz, bizId).Updates(
			map[string]any{
				"u_time":   now,
				"like_cnt": gorm.Expr("`like_cnt` - 1"), //加反引号为了避免与sql保留字冲突
			},
		).Error
		if err != nil {
			return err
		}
		return tx.Model(&UserLikeBiz{}).Where("uid = ? and biz_id = ? and biz = ?", uid, bizId, biz).Updates(
			map[string]any{
				"u_time": now,
				"status": 0,
			},
		).Error
	})
}

func (G *GORMInteractiveDao) IncreaseLikeCnt(ctx context.Context, biz string, bizId int64, uid int64) error {
	now := time.Now().UnixMilli()
	//开启一个跨表操作的事务,Upsert交互库与Like业务库
	return G.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//这里可以不调用where，也能根据冲突的键找到目标
		err := tx.Model(&Interactive{}).Where("biz = ? and biz_id = ?", biz, bizId).Clauses(
			clause.OnConflict{
				DoUpdates: clause.Assignments(map[string]interface{}{
					"u_time":   now,
					"like_cnt": gorm.Expr("like_cnt + 1"),
				})},
		).Create(&Interactive{
			Biz:        biz,
			BizId:      bizId,
			ReadCnt:    0,
			CollectCnt: 0,
			LikeCnt:    1,
			CTime:      now,
			UTime:      now,
		}).Error
		if err != nil {
			return err
		}
		//这里新版的mysql where乱序也会使用索引，老版本就不能，比较不智能
		return tx.Model(&UserLikeBiz{}).Where("uid = ? and biz_id = ? and biz = ?", uid, bizId, biz).Clauses(
			clause.OnConflict{
				//不需要检查是否曾经点过喜欢（前端的问题或者有人搞),结构都一样
				DoUpdates: clause.Assignments(map[string]interface{}{
					"u_time": now,
					"status": 1,
				})},
		).Create(&UserLikeBiz{
			Uid:    uid,
			Biz:    biz,
			BizId:  bizId,
			Status: 1,
			CTime:  now,
			UTime:  now,
		}).Error
	})
}

func (G *GORMInteractiveDao) IncreaseReadCount(ctx context.Context, biz string, bizId int64) error {
	now := time.Now().UnixMilli()
	//这里要避免check and do something模式，会有并发问题，最后用数据库本身自带锁的更新操作
	return G.db.WithContext(ctx).Clauses(clause.OnConflict{
		// mysql 是可以不写的
		//Columns: []clause.Column{{Name: "biz_id"}, {Name: "biz"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"read_cnt": gorm.Expr("read_cnt + 1"),
			"u_time":   now,
		})}).Create(&Interactive{ //insert xxx on duplicate key update xxx  gorm一般都要传指针
		Biz:        biz,
		BizId:      bizId,
		ReadCnt:    1,
		CollectCnt: 0,
		LikeCnt:    0,
		CTime:      now,
		UTime:      now,
	}).Error
}

func NewGORMInteractiveDao(db *gorm.DB) *GORMInteractiveDao {
	return &GORMInteractiveDao{db: db}
}

type Interactive struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	//考虑建立联合索引，通常bizId在前更好，因为区分度更高
	//这里可以创建联合unique索引
	//可以用priority指定顺序，默认按字段定义的顺序确定索引顺序
	BizId int64 `gorm:"uniqueIndex:biz_type_id"`
	//string类型gorm默认会转为 BLOB/TEXT类型，这里要显式指定一下varchar
	Biz        string `gorm:"uniqueIndex:biz_type_id;type:varchar(128)"`
	ReadCnt    int64
	CollectCnt int64
	LikeCnt    int64
	CTime      int64
	UTime      int64
}

// 记录谁点赞了什么
type UserLikeBiz struct {
	Id int64 `gorm:"primaryKey,autoincrement"`
	//UID在前，会有查询某个用户uid点赞了哪些内容的情况
	Uid   int64  `gorm:"uniqueIndex:biz_like_id"`
	BizId int64  `gorm:"uniqueIndex:biz_like_id"`
	Biz   string `gorm:"uniqueIndex:biz_like_id;type:varchar(128)"`
	// 只在 DB 层面生效的状态
	// 1- 有效，0-无效。软删除的用法
	Status uint8
	CTime  int64
	UTime  int64
}

type UserCollectionBiz struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 这算是一个冗余，因为正常来说，
	// 只需要在 Collection 中维持住 Uid 就可以
	Uid   int64  `gorm:"uniqueIndex:biz_like_id"`
	BizId int64  `gorm:"uniqueIndex:biz_like_id"`
	Biz   string `gorm:"uniqueIndex:biz_like_id;type:varchar(128)"`
	// 收藏夹 ID
	// 作为关联关系中的外键，我们这里需要索引
	Cid   int64 `gorm:"index:cid"`
	CTime int64
	UTime int64
}

type Collection struct {
	Id    int64  `gorm:"primaryKey,autoincrement"`
	Name  string `gorm:"varchar(1024)"`
	Uid   int64
	CTime int64
	UTime int64
}
