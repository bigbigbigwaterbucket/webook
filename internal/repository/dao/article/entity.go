package article

// Article，制作库
type Article struct {
	Id int64 `gorm:"primaryKey,autoIncrement" bson:"id,omitempty"` //指定多个tag
	//标题长度1024
	Title   string `gorm:"type=varchar(1024)" bson:"title,omitempty"` //指定sql中的字段类型
	Content string `gorm:"type=BLOB" bson:"content,omitempty"`
	//如何设计索引？
	//3、在帖子这里，有创作者查询自己创作的内容的情景
	//1、产品经理说，要按照创建时间倒叙排列
	//2、单独查询某一篇

	//以下创建了联合索引
	//关于索引设计对查询的加速，可以学学explain命令
	AuthorId int64 `gorm:"index=aid_ctime" bson:"author_id,omitempty"`
	CTime    int64 `gorm:"index=aid_ctime" bson:"c_time,omitempty"`
	UTime    int64 `bson:"u_time,omitempty"`
	Status   uint8 `bson:"status,omitempty"`
}

type PublishArticle struct {
	Article `bson:"article,omitempty"`
}
