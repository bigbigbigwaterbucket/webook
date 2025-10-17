package ioc

import (
	"learning_go/webook/article/repository/dao/article"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	type Config struct {
		DSN string `yaml:"dsn"`
	}
	var config1 Config
	err := viper.UnmarshalKey("mysql", &config1)
	if err != nil {
		panic(err)
	}
	db, err := gorm.Open(mysql.Open(config1.DSN))
	if err != nil {
		panic(err)
	}
	err = article.InitTable(db)
	if err != nil {
		panic(err)
	}
	return db
}
