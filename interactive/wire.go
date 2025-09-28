//go:build wireinject

package main

import (
	"github.com/google/wire"
	"learning_go/webook/interactive/events"
	"learning_go/webook/interactive/grpc"
	"learning_go/webook/interactive/ioc"
	"learning_go/webook/interactive/repository"
	"learning_go/webook/interactive/repository/cache"
	"learning_go/webook/interactive/repository/dao"
	"learning_go/webook/interactive/service"
	"time"
)

var (
	thirdProvider          = wire.NewSet(ioc.InitBizDB, ioc.InitRedis, ioc.InitKafka, ioc.InitDstDB, ioc.InitSrcDB, ioc.InitDoubleWritePool)
	interactiveSvcProvider = wire.NewSet(wire.Value(time.Minute),
		dao.NewGORMInteractiveDao,
		cache.NewRedisInteractiveCache,
		repository.NewCachedInteractiveRepository,
		service.NewInteractiveServiceI)
	migratorSvcProvider = wire.NewSet(ioc.InitSyncProducer, ioc.InitMigratorScheduler,
		wire.Value(10)) //batchsize
)

func InitApp() *App {
	wire.Build(thirdProvider, interactiveSvcProvider, migratorSvcProvider,
		grpc.NewInteractiveServiceServer,
		events.NewInteractiveReadEventBatchConsumer,
		ioc.InitFixerConsumer,
		ioc.InitConsumers,
		ioc.InitGRPCXServer,
		wire.Struct(new(App), "*"), //要求wire构建结构体并填充所有字段
	)
	return nil
}
