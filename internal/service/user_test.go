package service

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository"
	repomocks "learning_go/webook/internal/repository/mocks"
	"testing"
	"time"
)

func TestUserServiceI_Login(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name string
		//构造依赖
		mock func(ctrl *gomock.Controller) repository.UserRepository
		//构造输入
		//context context.Context
		user domain.User
		//预期输出
		wantUser domain.User
		wantErr  error
	}{
		{
			name: "登陆成功",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				userRepo := repomocks.NewMockUserRepository(ctrl)
				userRepo.EXPECT().FindByEmail(gomock.Any(), domain.User{Email: "123@qq.com", Password: "757268", Ctime: now}).Return(domain.User{Email: "123@qq.com", Password: "$2a$10$kNPmg/6SGOIex9zUB6dXU.E5io7kJMX5.MpTvb8X01JD/WF5dbll2", Ctime: now}, nil)
				return userRepo
			},
			user:     domain.User{Email: "123@qq.com", Password: "757268", Ctime: now},
			wantUser: domain.User{Email: "123@qq.com", Password: "$2a$10$kNPmg/6SGOIex9zUB6dXU.E5io7kJMX5.MpTvb8X01JD/WF5dbll2", Ctime: now},
			wantErr:  nil,
		},
		{
			name: "用户未注册",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				userRepo := repomocks.NewMockUserRepository(ctrl)
				userRepo.EXPECT().FindByEmail(gomock.Any(), domain.User{Email: "123@qq.com", Password: "757268", Ctime: now}).
					Return(domain.User{}, repository.ErrUserNotFound)
				return userRepo
			},
			user:     domain.User{Email: "123@qq.com", Password: "757268", Ctime: now},
			wantUser: domain.User{},
			wantErr:  ErrInvalidUserOrPassword,
		},
		{
			name: "DB错误",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				userRepo := repomocks.NewMockUserRepository(ctrl)
				userRepo.EXPECT().FindByEmail(gomock.Any(), domain.User{Email: "123@qq.com", Password: "757268", Ctime: now}).
					Return(domain.User{}, errors.New("DB错误"))
				return userRepo
			},
			user:     domain.User{Email: "123@qq.com", Password: "757268", Ctime: now},
			wantUser: domain.User{},
			wantErr:  errors.New("DB错误"),
		},
		{
			name: "账号/密码不正确",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				userRepo := repomocks.NewMockUserRepository(ctrl)
				userRepo.EXPECT().FindByEmail(gomock.Any(), domain.User{Email: "123@qq.com", Password: "757268", Ctime: now}).Return(domain.User{Email: "123@qq.com", Password: "$2a$10$kNPmg/6SGOIX5.MpTvb8X01JD/WF5dbll2", Ctime: now}, nil)
				return userRepo
			},
			user:     domain.User{Email: "123@qq.com", Password: "757268", Ctime: now},
			wantUser: domain.User{},
			wantErr:  ErrInvalidUserOrPassword,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			userRepo := tc.mock(ctrl)
			userSvc := NewUserService(userRepo)
			// context不影响login单元测试，login执行时没有实际用到context
			user, err := userSvc.Login(context.Background(), tc.user)
			assert.Equal(t, tc.wantUser, user)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestEncrypt(t *testing.T) {
	EP, err := bcrypt.GenerateFromPassword([]byte("757268"), bcrypt.DefaultCost)
	if err == nil {
		t.Log(string(EP))
	}
}
