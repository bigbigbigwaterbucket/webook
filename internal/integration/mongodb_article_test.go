package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"learning_go/webook/article/domain"
	"learning_go/webook/article/repository/article"
	article3 "learning_go/webook/article/repository/dao/article"
	"learning_go/webook/article/service"
	"learning_go/webook/internal/web"
	"learning_go/webook/internal/web/ijwt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// 测试套件
type ArticleTestSuiteMongoDB struct {
	suite.Suite
	server  *gin.Engine
	db      *mongo.Database
	col     *mongo.Collection
	liveCol *mongo.Collection
}

func initLoggerM() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	//设置全局logger，全局输出日志的
	//你在你的代码里就可以直接用zap.XXX来记录日志
	zap.ReplaceGlobals(logger)
	//L()获取zap里的全局L（好像是一种类型）logger
	zap.L().Info("日志载入成功")
}

func (a *ArticleTestSuiteMongoDB) SetupSuite() {
	//会在所有测试执行之前，初始化一些内容
	initLoggerM()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	//这里这个ctx只控制连接mongoDB数据库的限时
	defer cancel()
	moniter := &event.CommandMonitor{
		//每个命令执行之前执行
		Started: func(ctx context.Context, startedEvent *event.CommandStartedEvent) {
			println(startedEvent.Command)
			//打印一下要执行的命令
			fmt.Println(startedEvent.Command)
		},
		//执行成功
		Succeeded: func(ctx context.Context, succeededEvent *event.CommandSucceededEvent) {

		},
		//执行失败
		Failed: func(ctx context.Context, failedEvent *event.CommandFailedEvent) {

		},
	}
	opt := options.Client().ApplyURI("mongodb://root:example@localhost:27017").SetMonitor(moniter)
	client, err := mongo.Connect(ctx, opt)
	require.NoError(a.T(), err)
	a.db = client.Database("webook")
	err = article3.InitMongoDBCollection(a.db)
	if err != nil {
		panic(err)
	}
	require.NoError(a.T(), err)
	a.server = gin.Default()
	a.server.Use(func(context *gin.Context) {
		context.Set("user", ijwt.UserClaims{Uid: 666})
	})
	col := a.db.Collection("articles")
	liveCol := a.db.Collection("publish_articles")
	a.col = col
	a.liveCol = liveCol
	//传入机器号（实例1号
	node, err := snowflake.NewNode(1)
	assert.NoError(a.T(), err)
	artHandler := web.NewArticleHandler(service.NewArticleServiceI(article.NewCachedArticleRepository(article3.NewMongoDBArticleDao(col, liveCol, node))))
	artHandler.RegisterRouter(a.server)
}

func (s *ArticleTestSuiteMongoDB) TestABC() {
	s.T().Log("hello，这是测试套件")
}

// 测试套件的钩子函数
// 测试组件的特殊方法TearDownTest，别拼错了,在测试结束后执行的操作，一般用来清空数据库等
func (s *ArticleTestSuiteMongoDB) TearDownTest() {
	//清空所有数据并恢复自增主键
	//s.db.Exec("TRUNCATE TABLE articles")
	//空字段可以匹配所有类型的数据
	s.col.DeleteMany(context.Background(), bson.M{})
	s.liveCol.DeleteMany(context.Background(), bson.M{})
}

func (s *ArticleTestSuiteMongoDB) TestEdit() {
	t := s.T() //使用编译套件时，要求套件结构体的方法不能显示声明testing.T，而要从套件中获取
	testCases := []struct {
		name string

		//集成测试前，准备数据
		before func(t *testing.T)
		//集成测试后，验证数据
		after func(t *testing.T)
		//预期输入
		article Article
		//预期输出
		wantCode int
		wantRes  Result[int64] //希望data是帖子的id
	}{
		{
			name: "新建帖子-保存成功",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				//检查数据库
				var art article3.Article
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				//err := s.db.Where("id=?", 1).First(&art).Error
				err := s.col.FindOne(ctx, article3.Article{AuthorId: 666}).Decode(&art)
				//err := s.col.FindOne(ctx, bson.M{"author_id": 666}).Decode(&art)
				assert.NoError(t, err)
				assert.True(t, art.CTime > 0)
				assert.True(t, art.UTime > 0)
				assert.True(t, art.Id > 0)
				art.CTime = 0
				art.UTime = 0
				art.Id = 0
				assert.Equal(t, article3.Article{
					Title:    "我的标题",
					Content:  "我的内容",
					AuthorId: 666,
					Status:   domain.ArticleStatusUnPublished.ToUnt8(),
				}, art)
			},
			article:  Article{Content: "我的内容", Title: "我的标题"},
			wantCode: 200,
			wantRes:  Result[int64]{Data: 1, Msg: "OK"},
		},
		{
			name: "更新帖子",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				_, err := s.col.InsertOne(ctx, article3.Article{Id: 2, Content: "我的内容", Title: "我的标题", AuthorId: 666, CTime: 123, UTime: 234})
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				//检查数据库
				var art article3.Article
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				//err := s.db.Where("id=?", 2).First(&art).Error
				filter := bson.M{"id": 2}
				err := s.col.FindOne(ctx, filter).Decode(&art)
				assert.NoError(t, err)
				assert.True(t, art.UTime > 234)
				art.UTime = 0
				assert.Equal(t, article3.Article{
					Id:       2,
					Title:    "新的标题",
					Content:  "新的内容",
					AuthorId: 666,
					CTime:    123,
					Status:   domain.ArticleStatusUnPublished.ToUnt8(),
				}, art)
			},
			article:  Article{Id: 2, Content: "新的内容", Title: "新的标题"},
			wantCode: 200,
			wantRes:  Result[int64]{Data: 2, Msg: "OK"},
		},
		//{
		//	name: "改一篇不存在的帖子",
		//	before: func(t *testing.T) {
		//		err := s.db.Create(article2.Article{Id: 3, Content: "我的内容", Title: "我的标题", AuthorId: 666, CTime: 123, UTime: 234,
		//			Status: domain.ArticleStatusUnPublished.ToUnt8()}).Error
		//		assert.NoError(t, err)
		//	},
		//	after: func(t *testing.T) {
		//		//检查数据库
		//		var art article2.Article
		//		err := s.db.Where("id=?", 3).First(&art).Error
		//		assert.NoError(t, err)
		//		assert.Equal(t, article2.Article{
		//			Id:       3,
		//			Title:    "我的标题",
		//			Content:  "我的内容",
		//			AuthorId: 666,
		//			CTime:    123,
		//			UTime:    234,
		//			Status:   domain.ArticleStatusUnPublished.ToUnt8(),
		//		}, art)
		//	},
		//	article:  Article{Id: 1111, Content: "新的内容", Title: "新的标题"},
		//	wantCode: 200,
		//	wantRes:  Result[int64]{Data: 0, Msg: "系统错误"},
		//},
		{
			name: "666号篡改别人(111号)的帖子",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				_, err := s.col.InsertOne(ctx, article3.Article{Id: 4, Content: "我的内容", Title: "我的标题", AuthorId: 111, CTime: 123, UTime: 234, Status: domain.ArticleStatusUnPublished.ToUnt8()})
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				//检查数据库
				var art article3.Article
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				filter := bson.M{"id": 4}
				err := s.col.FindOne(ctx, filter).Decode(&art)
				assert.NoError(t, err)
				assert.NoError(t, err)
				assert.Equal(t, article3.Article{
					Id:       4,
					Title:    "我的标题",
					Content:  "我的内容",
					AuthorId: 111,
					CTime:    123,
					UTime:    234,
					Status:   domain.ArticleStatusUnPublished.ToUnt8(),
				}, art)
			},
			article:  Article{Id: 4, Content: "新的内容", Title: "新的标题"},
			wantCode: 200,
			wantRes:  Result[int64]{Data: 0, Msg: "系统错误"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			reqBody, err := json.Marshal(tc.article)
			assert.NoError(t, err) //失败时记录，但继续执行
			req, err := http.NewRequest(http.MethodPost, "/articles/edit", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			require.NoError(t, err) //失败时停止
			resp := httptest.NewRecorder()
			//http请求进入gin框架的入口，gin的服务器会去处理这个请求
			//与http.Client的Do方法不同，Client是把请求发到外部url，在外部服务处理完后拿到外部的响应
			s.server.ServeHTTP(resp, req)

			assert.Equal(t, tc.wantCode, resp.Code)
			var webRes Result[int64]
			err = json.Unmarshal(resp.Body.Bytes(), &webRes)
			require.NoError(t, err)
			if tc.wantRes.Data > 0 {
				assert.True(t, webRes.Data > 0)
			}
			tc.after(t)
		})
	}
}

func (s *ArticleTestSuiteMongoDB) TestArticle_Publish() {
	t := s.T()

	testCases := []struct {
		name string
		// 要提前准备数据
		before func(t *testing.T)
		// 验证并且删除数据
		after func(t *testing.T)
		req   Article

		// 预期响应
		wantCode   int
		wantResult Result[int64]
	}{
		{
			name: "新建帖子并发表",
			before: func(t *testing.T) {
				// 什么也不需要做
			},
			after: func(t *testing.T) {
				// 验证一下数据
				var art article3.Article
				err := s.col.FindOne(context.Background(), bson.M{"author_id": 666}).Decode(&art)
				assert.NoError(t, err)
				//s.db.Where("author_id = ?", 666).First(&art)
				assert.Equal(t, "hello，你好", art.Title)
				assert.Equal(t, "随便试试", art.Content)
				assert.Equal(t, int64(666), art.AuthorId)
				assert.True(t, art.CTime > 0)
				assert.True(t, art.UTime > 0)
				var publishedArt article3.PublishArticle
				//这里的查询要小心，你插入的是带article结构体字段的bson，因此需要嵌套查询article.author_id字段！！！
				err = s.liveCol.FindOne(context.Background(), bson.M{"article.author_id": 666}).Decode(&publishedArt)
				assert.NoError(t, err)
				assert.Equal(t, "hello，你好", publishedArt.Title)
				assert.Equal(t, "随便试试", publishedArt.Content)
				assert.Equal(t, int64(666), publishedArt.AuthorId)
				assert.True(t, publishedArt.CTime > 0)
				assert.True(t, publishedArt.UTime > 0)
			},
			req: Article{
				Title:   "hello，你好",
				Content: "随便试试",
			},
			wantCode: 200,
			wantResult: Result[int64]{
				Msg: "OK",
			},
		},
		{
			// 制作库有，但是线上库没有
			name: "更新帖子并新发表",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				// 模拟已经存在的帖子，并且是已经发布的帖子
				_, err := s.col.InsertOne(ctx, &article3.Article{
					Id:       2,
					Title:    "我的标题",
					Content:  "我的内容",
					CTime:    456,
					UTime:    234,
					AuthorId: 666,
				})
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				// 验证一下数据
				var art article3.Article
				err := s.col.FindOne(ctx, bson.D{bson.E{Key: "id", Value: 2}}).Decode(&art)
				assert.NoError(t, err)
				assert.Equal(t, int64(2), art.Id)
				assert.Equal(t, "新的标题", art.Title)
				assert.Equal(t, "新的内容", art.Content)
				assert.Equal(t, int64(666), art.AuthorId)
				// 创建时间没变
				assert.Equal(t, int64(456), art.CTime)
				// 更新时间变了
				assert.True(t, art.UTime > 234)
				var publishedArt article3.PublishArticle
				err = s.liveCol.FindOne(ctx, bson.M{"article.id": 2}).Decode(&publishedArt)
				assert.NoError(t, err)
				assert.Equal(t, int64(2), art.Id)
				assert.Equal(t, "新的标题", art.Title)
				assert.Equal(t, "新的内容", art.Content)
				assert.Equal(t, int64(666), art.AuthorId)
				assert.True(t, publishedArt.CTime > 0)
				assert.True(t, publishedArt.UTime > 0)
			},
			req: Article{
				Id:      2,
				Title:   "新的标题",
				Content: "新的内容",
			},
			wantCode: 200,
			wantResult: Result[int64]{
				Msg: "OK",
			},
		},
		{
			name: "更新帖子，并且重新发表",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				art := article3.Article{
					Id:       3,
					Title:    "我的标题",
					Content:  "我的内容",
					CTime:    456,
					UTime:    234,
					AuthorId: 666,
				}
				// 模拟已经存在的帖子，并且是已经发布的帖子
				_, err := s.col.InsertOne(ctx, &art)
				assert.NoError(t, err)
				part := article3.PublishArticle{art}
				_, err = s.liveCol.InsertOne(ctx, &part)
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				// 验证一下数据
				var art article3.Article
				err := s.col.FindOne(ctx, bson.D{bson.E{Key: "id", Value: 3}}).Decode(&art)
				assert.NoError(t, err)
				assert.Equal(t, int64(3), art.Id)
				assert.Equal(t, "新的标题", art.Title)
				assert.Equal(t, "新的内容", art.Content)
				assert.Equal(t, int64(666), art.AuthorId)
				// 创建时间没变
				assert.Equal(t, int64(456), art.CTime)
				// 更新时间变了
				assert.True(t, art.UTime > 234)

				var part article3.PublishArticle
				err = s.liveCol.FindOne(ctx, bson.D{bson.E{Key: "article.id", Value: 3}}).Decode(&part)
				assert.NoError(t, err)
				assert.Equal(t, int64(3), part.Id)
				assert.Equal(t, "新的标题", part.Title)
				assert.Equal(t, "新的内容", part.Content)
				assert.Equal(t, int64(666), part.AuthorId)
				// 创建时间没变
				assert.Equal(t, int64(456), part.CTime)
				// 更新时间变了
				assert.True(t, part.UTime > 234)
			},
			req: Article{
				Id:      3,
				Title:   "新的标题",
				Content: "新的内容",
			},
			wantCode: 200,
			wantResult: Result[int64]{
				Msg: "OK",
			},
		},
		{
			name: "更新别人的帖子，并且发表失败",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				art := article3.Article{
					Id:      4,
					Title:   "我的标题",
					Content: "我的内容",
					CTime:   456,
					UTime:   234,
					// 注意。这个 AuthorID 我们设置为另外一个人的ID
					AuthorId: 789,
				}
				// 模拟已经存在的帖子，并且是已经发布的帖子
				_, err := s.col.InsertOne(ctx, &art)
				assert.NoError(t, err)
				part := article3.PublishArticle{art}
				_, err = s.liveCol.InsertOne(ctx, &part)
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 更新应该是失败了，数据没有发生变化
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				// 验证一下数据
				var art article3.Article
				err := s.col.FindOne(ctx, bson.D{bson.E{Key: "id", Value: 4}}).Decode(&art)
				assert.NoError(t, err)
				assert.Equal(t, int64(4), art.Id)
				assert.Equal(t, "我的标题", art.Title)
				assert.Equal(t, "我的内容", art.Content)
				assert.Equal(t, int64(456), art.CTime)
				assert.Equal(t, int64(234), art.UTime)
				assert.Equal(t, int64(789), art.AuthorId)

				var part article3.PublishArticle
				// 数据没有变化
				err = s.liveCol.FindOne(ctx, bson.D{bson.E{Key: "article.id", Value: 4}}).Decode(&part)
				assert.NoError(t, err)
				assert.Equal(t, int64(4), part.Id)
				assert.Equal(t, "我的标题", part.Title)
				assert.Equal(t, "我的内容", part.Content)
				assert.Equal(t, int64(789), part.AuthorId)
				// 创建时间没变
				assert.Equal(t, int64(456), part.CTime)
				// 更新时间没变
				assert.Equal(t, int64(234), part.UTime)
			},
			req: Article{
				Id:      4,
				Title:   "新的标题",
				Content: "新的内容",
			},
			wantCode: 200,
			wantResult: Result[int64]{
				Msg: "系统错误",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			data, err := json.Marshal(tc.req)
			// 不能有 error
			assert.NoError(t, err)
			req, err := http.NewRequest(http.MethodPost,
				"/articles/publish", bytes.NewReader(data))
			assert.NoError(t, err)
			req.Header.Set("Content-Type",
				"application/json")
			recorder := httptest.NewRecorder()

			s.server.ServeHTTP(recorder, req)
			code := recorder.Code
			assert.Equal(t, tc.wantCode, code)
			if code != http.StatusOK {
				return
			}
			// 反序列化为结果
			// 利用泛型来限定结果必须是 int64
			var result Result[int64]
			err = json.Unmarshal(recorder.Body.Bytes(), &result)
			assert.NoError(t, err)
			if result.Data > 0 {
				result.Data = 0
			}
			assert.Equal(t, tc.wantResult, result)
			tc.after(t)
		})
	}
}

func TestArticleM(t *testing.T) {
	suite.Run(t, &ArticleTestSuiteMongoDB{})
}
