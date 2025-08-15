package repository

import (
	"context"
	"database/sql"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository/cache"
	"learning_go/webook/internal/repository/dao"
	"time"
)

var (
	ErrUserDuplicate = dao.ErrUserDuplicate
	ErrUserNotFound  = dao.ErrUserNotFound
)

type UserRepository struct {
	Dao   *dao.UserDAO
	Cache *cache.UserCache
}

func NewUserRepository(dao *dao.UserDAO, cache *cache.UserCache) *UserRepository {
	return &UserRepository{Dao: dao, Cache: cache}
}

func (this *UserRepository) FindByEmail(ctx context.Context, u domain.User) (domain.User, error) {
	user, err := this.Dao.FindByEmail(ctx, u.Email)
	if err != nil {
		return domain.User{}, err
	}
	return this.entityToDomain(user), err
}

func (this *UserRepository) FindByPhone(ctx context.Context, phone string) (domain.User, error) {
	user, err := this.Dao.FindByPhone(ctx, phone)
	if err != nil {
		return domain.User{}, err
	}
	return this.entityToDomain(user), err
}

// 这里也要传web服务的context？
func (this *UserRepository) Create(ctx context.Context, u domain.User) error {
	return this.Dao.Insert(ctx, this.domainToEntity(u))
}

func (this *UserRepository) Update(ctx context.Context, u domain.User) error {
	return this.Dao.Update(ctx, dao.User{Id: u.Id, Name: u.Name, Birthday: u.Birthday, Introduce: u.Introduce})
}

func (this *UserRepository) FindById(ctx context.Context, userId int64) (domain.User, error) {
	//先从cache里找，找不到去数据库DAO里找，找到了写回cache
	user, err := this.Cache.Get(ctx, userId)
	if err == nil {
		return user, nil
	}
	//考虑如果redis崩了导致错误，要不要从数据库加载user？
	//面试时考虑：加载，万一redis崩了，对数据库进行限流保护
	//实践就简单点：就直接不加载，return 错误

	//下列实现是无论redis崩没崩，找不到就去数据库
	ue, err := this.Dao.FindById(ctx, userId)
	if err != nil {
		//数据库崩了
		return domain.User{}, err
	}
	u := this.entityToDomain(ue)

	//开goroutine协程，会出现缓存一致性问题，但是只要用到了缓存，就不指望强数据一致性？？？
	go func() {
		err = this.Cache.Set(ctx, u)
		if err != nil {
			//几乎可以确定是redis或者到redis的网络崩了，要日志
			//return
		}
	}()
	return u, err
}

func (ur *UserRepository) domainToEntity(u domain.User) dao.User {
	return dao.User{Id: u.Id, Email: sql.NullString{String: u.Email, Valid: u.Email != ""}, Phone: sql.NullString{String: u.Phone, Valid: u.Phone != ""}, Password: u.Password, Ctime: u.Ctime.UnixMilli()}
}

func (ur *UserRepository) entityToDomain(u dao.User) domain.User {
	return domain.User{Id: u.Id, Email: u.Email.String, Password: u.Password, Ctime: time.UnixMilli(u.Ctime), Phone: u.Phone.String}
}
