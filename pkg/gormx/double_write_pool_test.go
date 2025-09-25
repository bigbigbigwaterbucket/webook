package gormx

import (
	"github.com/ecodeclub/ekit/syncx/atomicx"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
)

func TestDoubleWriterPool(t *testing.T) {
	srcDB, err := gorm.Open(mysql.Open("root:root@tcp(localhost:13316)/webook"))
	require.NoError(t, err)
	err = srcDB.AutoMigrate(&Interactive{})
	dstDB, err := gorm.Open(mysql.Open("root:root@tcp(localhost:13316)/webook_intr"))
	require.NoError(t, err)
	err = dstDB.AutoMigrate(&Interactive{})
	//
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: &DoubleWritePool{src: srcDB.ConnPool,
		dst: dstDB.ConnPool, pattern: atomicx.NewValueOf[string](PatternSrcFirst)}},
	))
	require.NoError(t, err)
	err = db.Create(&Interactive{Biz: "test2", BizId: 3}).Error
	require.NoError(t, err)
	err = db.Transaction(func(tx *gorm.DB) error {
		err := db.Create(&Interactive{Biz: "test_tx", BizId: 3}).Error
		if err != nil {
			return err
		}
		return db.Create(&Interactive{Biz: "test_tx", BizId: 4}).Error
	})
	require.NoError(t, err)
	err = db.Model(&Interactive{}).Where("id = ?", 1).Updates(map[string]any{
		"biz_id": 667,
	}).Error
	require.NoError(t, err)
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
