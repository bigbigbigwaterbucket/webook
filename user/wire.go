//go:build wireinject

package main

import (
	"learning_go/webook/user/ioc"
	"learning_go/webook/user/repository"
	"learning_go/webook/user/repository/cache"
	"learning_go/webook/user/repository/dao"
	"learning_go/webook/user/service"

	"github.com/google/wire"
)

var thirdProvider = wire.NewSet(ioc.InitDB, ioc.InitRedis)

func InitApp() *App {
	wire.Build(thirdProvider, dao.NewUserDAO, cache.NewUserCache, repository.NewUserRepository, service.NewUserService,
		ioc.InitGRPCXServer, wire.Struct(new(App), "*"))
	return nil
}
