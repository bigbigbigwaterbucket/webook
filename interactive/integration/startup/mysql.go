package startup

import (
	"context"
	"database/sql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"learning_go/webook/internal/repository/dao"
	"log"
	"sync"
	"time"
)

// db连接保持单例，用once保证并发安全
var (
	db        *gorm.DB
	onceMysql sync.Once
)

func InitDB() *gorm.DB {
	onceMysql.Do(func() {
		dsn := "root:root@tcp(localhost:13316)/webook"
		sqlDb, err := sql.Open("mysql", dsn)
		if err != nil {
			panic(err)
		}
		for {
			//连接后数据库可能还没准备好，通过ping命令等待数据库准备好
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			err := sqlDb.PingContext(ctx)
			cancel()
			if err == nil {
				break
			}
			log.Panicln("等待mysql连接")
		}
		db, err = gorm.Open(mysql.Open(dsn))
		if err != nil {
			panic(err)
		}
		err = dao.InitTable(db)
		if err != nil {
			panic(err)
		}
		//debug模式会输出sql日志
		db = db.Debug()
	})
	return db
}
