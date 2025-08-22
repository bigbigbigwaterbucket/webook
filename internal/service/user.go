package service

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository"
)

// 每一层都有自己的邮箱冲突err，这样在测试时可以方便知道是哪一层出的错
// 也可以用全局的，但是不方便知道是哪一层出错，另外，这样传上层不知道底层用的是gorm的错误
var (
	ErrUserDuplicate = repository.ErrUserDuplicate
	//四层模型中 err在哪产生，就在哪定义
	ErrInvalidUserOrPassword = errors.New("账号/邮箱或密码不对")
)

type UserServiceI struct {
	Repo repository.UserRepository
}

type UserService interface {
	Login(ctx context.Context, user domain.User) (domain.User, error)
	SignUp(ctx context.Context, user domain.User) error
	Edit(ctx context.Context, user domain.User) error
	Profile(ctx context.Context, userId int64) (domain.User, error)
	FindOrCreateByPhone(ctx context.Context, phone string) (domain.User, error)
	// FindOrCreateByWechat 查找或者初始化
	// 随着业务增长，这边可以考虑拆分出去作为一个新的 Service
	FindOrCreateByWechat(ctx context.Context, info domain.WechatInfo) (domain.User, error)
}

func NewUserService(repo repository.UserRepository) *UserServiceI {
	return &UserServiceI{Repo: repo}
}

func (this *UserServiceI) Login(ctx context.Context, user domain.User) (domain.User, error) {
	u, err := this.Repo.FindByEmail(ctx, user)
	// 数据库没找到相应数据的错误，转换为账号/邮箱或密码不对的错误
	if err == repository.ErrUserNotFound {
		return domain.User{}, ErrInvalidUserOrPassword
	}
	//其他错误
	if err != nil {
		return domain.User{}, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(user.Password))
	if err != nil {
		//DEBUG 日志
		return domain.User{}, ErrInvalidUserOrPassword
	}
	return u, err
}

func (this *UserServiceI) SignUp(ctx context.Context, user domain.User) error {
	//考虑加密放在哪里
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hash)
	return this.Repo.Create(ctx, user)
}

func (this *UserServiceI) Edit(ctx context.Context, user domain.User) error {
	return this.Repo.Update(ctx, user)
}

func (this *UserServiceI) Profile(ctx context.Context, userId int64) (domain.User, error) {
	user, err := this.Repo.FindById(ctx, userId)
	if err != nil {
		return domain.User{}, err
	}
	return user, err
}

func (this *UserServiceI) FindOrCreateByPhone(ctx context.Context, phone string) (domain.User, error) {
	user, err := this.Repo.FindByPhone(ctx, phone)
	if err != repository.ErrUserNotFound {
		//找到了或者其他错误都进入此分支
		return user, err
	}
	//没找到，说明没有这个用户
	err = this.Repo.Create(ctx, domain.User{Phone: phone})
	if err != nil {
		return domain.User{Phone: phone}, err
	}
	//坑，会遇到主从延迟问题，创建操作慢于查找操作？（（（（
	//这里再查一遍是因为创建的时候，uid没有一并返回
	return this.Repo.FindByPhone(ctx, phone)
}

func (this *UserServiceI) FindOrCreateByWechat(ctx context.Context, info domain.WechatInfo) (domain.User, error) {
	user, err := this.Repo.FindByWechat(ctx, info.OpenId)
	if err != nil {
		if err == repository.ErrUserNotFound {
			err = this.Repo.Create(ctx, domain.User{WechatInfo: info})
			if err != nil {
				return domain.User{}, err
			}
			return this.Repo.FindByWechat(ctx, info.OpenId)
		}
		return domain.User{}, err
	}
	return user, err
}
