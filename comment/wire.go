//go:build wireinject

package main

import (
	"learning_go/webook/comment/ioc"
	"learning_go/webook/comment/repository"
	"learning_go/webook/comment/repository/dao"
	"learning_go/webook/comment/service"

	"github.com/google/wire"
)

var thirdProvider = wire.NewSet(ioc.InitDB, ioc.InitRedis)

func InitApp() *App {
	wire.Build(thirdProvider, dao.NewGORMCommentDao, repository.NewCachedCommentRepository, service.NewCommentServiceI,
		ioc.InitGRPCXServer, wire.Struct(new(App), "*"))
	return nil
}
