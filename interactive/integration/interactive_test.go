package integration

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"learning_go/webook/interactive/integration/startup"
	"learning_go/webook/interactive/repository/dao"
	"testing"
	"time"
)

type InteractiveServiceSuite struct {
	suite.Suite
	db    *gorm.DB
	redis redis.Cmdable
}

func (i *InteractiveServiceSuite) SetupSuite() {
	i.db = startup.InitDB()
	i.redis = startup.InitRedis()
}

func (i *InteractiveServiceSuite) TearDownTest() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err := i.db.Exec("TRUNCATE TABLE `interactives`").Error
	assert.NoError(i.T(), err)
	err = i.db.Exec("TRUNCATE TABLE `user_like_bizs`").Error
	assert.NoError(i.T(), err)
	err = i.db.Exec("TRUNCATE TABLE `user_collection_bizs`").Error
	assert.NoError(i.T(), err)
	// 清空 Redis
	err = i.redis.FlushDB(ctx).Err()
	assert.NoError(i.T(), err)
}

func (i *InteractiveServiceSuite) TestIncreaseReadCount() {
	testCases := []struct {
		name    string
		before  func(*testing.T)
		after   func(*testing.T)
		biz     string
		bizId   int64
		wantErr error
	}{
		{
			name: "success to increase",
			before: func(t *testing.T) {
				now := time.Now()
				err := i.db.Model(dao.Interactive{}).Create(&dao.Interactive{ //create方法要传指针
					BizId:   456,
					Biz:     "article",
					ReadCnt: 3,
					UTime:   now.UnixMilli(),
					CTime:   now.UnixMilli(),
				}).Error
				require.NoError(t, err)
				err = i.redis.HSet(context.Background(), "interactive:article:456", "read_cnt", 3).Err()
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var res dao.Interactive
				err := i.db.Model(dao.Interactive{}).Where("biz_id = ?", 456).First(&res).Error
				require.NoError(t, err)
				assert.True(t, res.CTime > 0)
				assert.True(t, res.UTime > 0)
				assert.Equal(t, res, dao.Interactive{
					Id:      1,
					BizId:   456,
					Biz:     "article",
					ReadCnt: 4,
					UTime:   res.UTime,
					CTime:   res.CTime,
				})
				rc, err := i.redis.HGet(context.Background(), "interactive:article:456", "read_cnt").Int()
				assert.NoError(t, err)
				assert.Equal(t, 4, rc)
			},
			biz:     "article",
			bizId:   456,
			wantErr: nil,
		},
		{
			name: "无缓存",
			before: func(t *testing.T) {
				now := time.Now()
				err := i.db.Model(dao.Interactive{}).Create(&dao.Interactive{ //create方法要传指针
					Id:      2,
					BizId:   789,
					Biz:     "article",
					ReadCnt: 3,
					UTime:   now.UnixMilli(),
					CTime:   now.UnixMilli(),
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				time.Sleep(time.Second)
				var res dao.Interactive
				err := i.db.Model(dao.Interactive{}).Where("biz_id = ?", 789).First(&res).Error
				require.NoError(t, err)
				assert.True(t, res.CTime > 0)
				assert.True(t, res.UTime > 0)
				assert.Equal(t, res, dao.Interactive{
					Id:      2,
					BizId:   789,
					Biz:     "article",
					ReadCnt: 4,
					UTime:   res.UTime,
					CTime:   res.CTime,
				})
				cnt, err := i.redis.Exists(context.Background(), "interactive:article:789").Result()
				assert.NoError(t, err)
				assert.Equal(t, int64(0), cnt)
			},
			biz:     "article",
			bizId:   789,
			wantErr: nil,
		},
		{
			name: "无缓存无数据库",
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {
				var res dao.Interactive
				err := i.db.Model(dao.Interactive{}).Where("biz_id = ?", 111).First(&res).Error
				require.NoError(t, err)
				assert.True(t, res.CTime > 0)
				assert.True(t, res.UTime > 0)
				assert.Equal(t, res, dao.Interactive{
					Id:      4, //for update跳号问题，这里id+2
					BizId:   111,
					Biz:     "article",
					ReadCnt: 1,
					UTime:   res.UTime,
					CTime:   res.CTime,
				})
				cnt, err := i.redis.Exists(context.Background(), "interactive:article:111").Result()
				assert.NoError(t, err)
				assert.Equal(t, int64(0), cnt)
			},
			biz:     "article",
			bizId:   111,
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		i.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			svc := startup.InitInteractiveGRPCService()
			err := svc.IncreaseReadCount(context.Background(), tc.biz, tc.bizId)
			assert.Equal(t, tc.wantErr, err)
			tc.after(t)
		})
	}
}

func TestInteractiveService(t *testing.T) {
	suite.Run(t, &InteractiveServiceSuite{})
}
