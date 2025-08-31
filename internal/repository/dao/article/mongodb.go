package article

import (
	"context"
	"fmt"
	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"learning_go/webook/internal/domain"
	"time"
)

type MongoDBArticleDao struct {
	//这两暂时还用不到
	//client   *mongo.Client //传指针，保证单例，只需要连接一个
	//database *mongo.Database
	//制作库
	collection *mongo.Collection
	//线上库
	liveCollection *mongo.Collection
	node           *snowflake.Node
}

func NewMongoDBArticleDao(col *mongo.Collection, liveCol *mongo.Collection, node *snowflake.Node) *MongoDBArticleDao {
	return &MongoDBArticleDao{collection: col, liveCollection: liveCol, node: node}
}

// InitMongoDBCollection只负责建表（collection），如果想获取表，mongoDB需要根据表名来获取
func InitMongoDBCollection(db *mongo.Database) error {
	//_ = db.Collection("articles")
	//_ = db.Collection("publish_articles")
	return InitCollectionIndex(db)
}

func InitCollectionIndex(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	index := []mongo.IndexModel{
		{
			//id业务里使用，一般都会建一个唯一索引
			Keys:    bson.D{bson.E{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			//索引设计的ESR原则，Equal、Sort、Range
			//这里制作库需要按作者查询内容，并且按ctime排序
			Keys: bson.D{bson.E{Key: "author_id", Value: 1},
				bson.E{Key: "ctime", Value: 1},
			},
			Options: options.Index(),
		},
	}
	//在子文档/嵌套字段上建立索引
	indexLive := []mongo.IndexModel{
		{
			//id业务里使用，一般都会建一个唯一索引
			Keys:    bson.D{bson.E{Key: "article.id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			//索引设计的ESR原则，Equal、Sort、Range
			//这里制作库需要按作者查询内容，并且按ctime排序
			Keys: bson.D{bson.E{Key: "article.author_id", Value: 1},
				bson.E{Key: "article.c_time", Value: 1},
			},
			Options: options.Index(),
		},
	}
	_, err := db.Collection("articles").Indexes().CreateMany(ctx, index)
	if err != nil {
		return err
	}
	_, err = db.Collection("publish_articles").Indexes().CreateMany(ctx, indexLive)
	return err
}

func (m *MongoDBArticleDao) Insert(ctx context.Context, art Article) (int64, error) {
	now := time.Now().UnixMilli()
	art.CTime = now
	art.UTime = now
	id := m.node.Generate().Int64()
	art.Id = id
	_, err := m.collection.InsertOne(ctx, art)
	//这里不能拿到自增主键，只有mongoDB内部的id，是这样子的:ObjectID("64e8aaf9a1234abcd567ef01")，对应_id字段
	//这个东西类型是[12]byte，没法转成int64
	//res.InsertedID
	//所以这里使用雪花算法生成了一个id
	return id, err
}

func (m *MongoDBArticleDao) UpdateById(ctx context.Context, art Article) (int64, error) {
	filter := bson.M{"id": art.Id, "author_id": art.AuthorId}
	//update := bson.D{bson.E{"$set", bson.M{""}}}
	now := time.Now().UnixMilli()
	art.UTime = now
	//bson.M可以作为一个bson.D{bson.E{}}的嵌套，bson.M可以放写操作，也可以放读（直接写筛选字段）
	uRes, err := m.collection.UpdateOne(ctx, filter, bson.M{"$set": Article{
		Title:   art.Title,
		Content: art.Content,
		UTime:   art.UTime,
		Status:  domain.ArticleStatusUnPublished.ToUnt8(), //更新文章后记得把文章状态置为未发表
	}})
	if err != nil {
		return art.Id, err
	}
	if uRes.ModifiedCount == 0 {
		zap.L().Warn("文章更新失败", zap.Error(fmt.Errorf("更新失败，创作者非法 aId:%d, authorId:%d", art.Id, art.AuthorId)))
		return art.Id, fmt.Errorf("更新失败，创作者非法 aId:%d, authorId:%d", art.Id, art.AuthorId)
	}
	return art.Id, err
}

func (m *MongoDBArticleDao) Sync(ctx context.Context, art Article) (int64, error) {
	//邓明没用事务，说是没找到事务
	//但是我问gpt却说有
	//怀疑邓明平时编程就是靠 a. 自动补全查看方法，找不到也懒得搜索看文档。。。
	var id int64
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if art.Id > 0 {
		//文档在制作库有
		id, err = m.UpdateById(ctx, art)
		if err != nil {
			return id, err
		}
	} else {
		id, err = m.Insert(ctx, art)
		if err != nil {
			return id, err
		}
	}
	art.Id = id
	//由于这里没做事务，因此只能简单重试
	publishArt := PublishArticle{Article: art}
	return m.UpdateOrInsert(ctx, publishArt)
}

func (m *MongoDBArticleDao) UpdateOrInsert(ctx context.Context, publishArt PublishArticle) (int64, error) {
	now := time.Now().UnixMilli()
	publishArt.UTime = now
	var err error
	for i := 0; i < 1; i++ {
		//线上库要实现upsert
		//$setOnInsert字段其实是在set的基础上重新创建字段，因此要避免和$set字段更新的值冲突
		_, err = m.liveCollection.UpdateOne(ctx,
			bson.M{"article.id": publishArt.Id},
			//这里插入的是是article结构体字段，不会像mysql一样需要拆成基础字段，而是直接插入一个article结构体bson
			bson.M{"$set": bson.M{"article.title": publishArt.Title, "article.content": publishArt.Content,
				"article.u_time": publishArt.UTime,
				"article.status": publishArt.Status,
				//"article.c_time": 0, 不能有冲突字段
			}, "$setOnInsert": bson.M{"article.c_time": now, "article.id": publishArt.Id, "article.author_id": publishArt.AuthorId}},
			options.Update().SetUpsert(true))
		if err == nil {
			break
		}
	}
	return publishArt.Id, err
}

func (m *MongoDBArticleDao) SyncStatus(ctx *gin.Context, id int64, uid int64, status uint8) error {
	//TODO implement me
	panic("implement me")
}
