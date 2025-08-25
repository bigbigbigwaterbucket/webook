package dao

import (
	"context"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"time"
)

type CodeDao interface {
	Insert(ctx context.Context, code Code) error
	Delete(ctx context.Context, key string) error
	FindByKey(ctx context.Context, key string) (Code, error)
}
type CodeDaoI struct {
	db *gorm.DB
}

func NewCodeDaoI(db *gorm.DB) *CodeDaoI {
	return &CodeDaoI{db: db}
}

func (c *CodeDaoI) Insert(ctx context.Context, code Code) error {
	now := time.Now().UnixMilli()
	code.CTime = now
	code.UTime = now
	err := c.db.WithContext(ctx).Create(&code).Error
	//插入时注意检查一下索引冲突err
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		const uniqueConflictsErrNo = 1062 //唯一索引冲突
		if mysqlErr.Number == uniqueConflictsErrNo {
			return ErrUserDuplicate
		} //验证码冲突
	}
	return err
}

func (c *CodeDaoI) Delete(ctx context.Context, key string) error {
	return c.db.WithContext(ctx).Where("key=?", key).Delete(&Code{}).Error //都需要传结构体指针，gorm会根据结构体类型推断表名
}

func (c *CodeDaoI) FindByKey(ctx context.Context, key string) (Code, error) {
	var code Code
	err := c.db.WithContext(ctx).Where("key=?", key).First(&code).Error
	if err != nil {
		return Code{}, err
	}
	return code, nil
}

type Code struct {
	Key   string `gorm:"primaryKey"`
	Code  string
	CTime int64
	UTime int64
}
