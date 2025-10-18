package repository

import (
	"context"
	cachemocks "learning_go/webook/internal/repository/cache/mocks"
	daomocks "learning_go/webook/internal/repository/dao/mocks"
	"learning_go/webook/user/domain"
	repository2 "learning_go/webook/user/repository"
	"learning_go/webook/user/repository/cache"
	"learning_go/webook/user/repository/dao"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUserRepositoryI_FindById(t *testing.T) {
	testCacses := []struct {
		name string
		mock func(*gomock.Controller) (dao.UserDAO, cache.UserCache)

		userId   int64
		wantUser domain.User
		wantErr  error
	}{
		{
			name: "数据库找到",
			mock: func(controller *gomock.Controller) (dao.UserDAO, cache.UserCache) {
				c := cachemocks.NewMockUserCache(controller)
				c.EXPECT().Get(gomock.Any(), int64(12)).Return(domain.User{}, repository2.ErrUserNotFound)
				c.EXPECT().Set(gomock.Any(), domain.User{Id: 12, Ctime: time.UnixMilli(0)}).Return(nil)
				d := daomocks.NewMockUserDAO(controller)
				d.EXPECT().FindById(gomock.Any(), int64(12)).Return(dao.User{Id: 12}, nil)
				return d, c
			},
			userId: 12,
			//这里Ctime是从dao的int64 0转换而来的， time.UnixMilli(0)是必须要赋的，不然默认初始化的是0000000-00:00
			wantUser: domain.User{Id: 12, Ctime: time.UnixMilli(0)},
			wantErr:  nil,
		},
	}
	for _, tc := range testCacses {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := repository2.NewUserRepository(tc.mock(ctrl))
			user, err := repo.FindById(context.Background(), tc.userId)
			assert.Equal(t, tc.wantUser, user)
			assert.Equal(t, tc.wantErr, err)
			time.Sleep(time.Second) //暂停1s等待goroutine执行完毕，这样期望的Set函数就能执行完
		})
	}
}
