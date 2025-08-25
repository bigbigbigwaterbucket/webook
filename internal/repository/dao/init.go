package dao

import "gorm.io/gorm"

// 垃圾设计，没有根据注解自动扫描建表的功能，java只要加上@Entity注解和配置，可以开启自动建表
func InitTable(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Code{}, &Article{})
}
