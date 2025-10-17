//go:build wireinject

package main

import (
	"learning_go/webook/article/ioc"
	articleRepo "learning_go/webook/article/repository/article"
	"learning_go/webook/article/repository/cache"
	articleDao "learning_go/webook/article/repository/dao/article"
	"learning_go/webook/article/service"

	"github.com/google/wire"
)

var (
	thirdProvider = wire.NewSet(ioc.InitDB, ioc.InitRedis)
)

func InitApp() *App {
	wire.Build(thirdProvider, articleDao.NewGormArticleDao, cache.NewRedisArticleCache,
		articleRepo.NewCachedArticleRepository, service.NewArticleServiceI,
		ioc.InitGRPCXServer, wire.Struct(new(App), "*")) //要求wire构建结构体并填充所有字段
	return nil
}
