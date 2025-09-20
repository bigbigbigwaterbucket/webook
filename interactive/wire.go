//go:build wireinject

package main

import (
	"github.com/google/wire"
	"learning_go/webook/interactive/grpc"
	"learning_go/webook/interactive/ioc"
	"learning_go/webook/interactive/repository"
	"learning_go/webook/interactive/repository/cache"
	"learning_go/webook/interactive/repository/dao"
	"learning_go/webook/interactive/service"
	"time"
)

var (
	thirdProvider          = wire.NewSet(ioc.InitDB, ioc.InitRedis)
	interactiveSvcProvider = wire.NewSet(wire.Value(time.Minute),
		dao.NewGORMInteractiveDao,
		cache.NewRedisInteractiveCache,
		repository.NewCachedInteractiveRepository,
		service.NewInteractiveServiceI)
)

func InitGRPCService() *grpc.InteractiveServiceServer {
	wire.Build(thirdProvider, interactiveSvcProvider, grpc.NewInteractiveServiceServer)
	return nil
}
