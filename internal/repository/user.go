package repository

import (
	"context"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository/dao"
)

var (
	ErrUserDuplicateEmail = dao.ErrUserDuplicateEmail
	ErrUserNotFound       = dao.ErrUserNotFound
)

type UserRepository struct {
	Dao *dao.UserDAO
}

func (this *UserRepository) FindByEmail(ctx context.Context, u domain.User) (domain.User, error) {
	user, err := this.Dao.FindByEmail(ctx, u.Email)
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{Id: user.Id, Email: user.Email, Password: user.Password}, err
}

// 这里也要传web服务的context？
func (this *UserRepository) Create(ctx context.Context, u domain.User) error {
	return this.Dao.Insert(ctx, dao.User{Email: u.Email, Password: u.Password})
}

func (this *UserRepository) Update(ctx context.Context, u domain.User) error {
	return this.Dao.Update(ctx, dao.User{Id: u.Id, Name: u.Name, Birthday: u.Birthday, Introduce: u.Introduce})
}

func (this *UserRepository) FindById(ctx context.Context, userId int64) (domain.User, error) {
	//先从cache里找，找不到去数据库DAO里找，找到了写回cache
	user, err := this.Dao.FindById(ctx, userId)
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{Email: user.Email, Id: userId, Name: user.Name, Introduce: user.Introduce, Birthday: user.Birthday}, err
}
