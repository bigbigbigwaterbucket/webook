package article

import "gorm.io/gorm"

func InitTable(db *gorm.DB) error {
	return db.AutoMigrate(&Article{}, &PublishArticle{})
}
