//go:build wireinject

//自定义标签，防止与wire生成的初始化函数代码冲突（同一个包重名）

package startup

import (
	"github.com/google/wire"
	"learning_go/webook/interactive/repository"
	"learning_go/webook/interactive/repository/cache"
	"learning_go/webook/interactive/repository/dao"
	"learning_go/webook/interactive/service"
	"time"
)

var thirdProvider = wire.NewSet(InitRedis, InitDB)

var interactiveSvcProvider = wire.NewSet(
	wire.Value(time.Minute),
	dao.NewGORMInteractiveDao,
	cache.NewRedisInteractiveCache,
	repository.NewCachedInteractiveRepository,
	service.NewInteractiveServiceI,
)

func InitInteractiveGRPCService() service.InteractiveService {
	wire.Build(thirdProvider, interactiveSvcProvider)
	return service.NewInteractiveServiceI(nil) //占位返回，wire通过扫描build和函数签名来生成函数
}
