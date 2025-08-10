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
	ErrUserDuplicateEmail = repository.ErrUserDuplicateEmail
	//四层模型中 err在哪产生，就在哪定义
	ErrInvalidUserOrPassword = errors.New("账号/邮箱或密码不对")
)

type UserService struct {
	Repo *repository.UserRepository
}

func (this *UserService) Login(ctx context.Context, user domain.User) (domain.User, error) {
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

func (this *UserService) SignUp(ctx context.Context, user domain.User) error {
	//考虑加密放在哪里
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hash)
	return this.Repo.Create(ctx, user)
}

func (this *UserService) Edit(ctx context.Context, user domain.User) error {
	return this.Repo.Update(ctx, user)
}

func (this *UserService) Profile(ctx context.Context, userId int64) (domain.User, error) {
	user, err := this.Repo.FindById(ctx, userId)
	if err != nil {
		return domain.User{}, err
	}
	return user, err
}
