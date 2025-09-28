package ioc

import (
	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"learning_go/webook/interactive/repository/dao"
	"learning_go/webook/pkg/ginx"
	"learning_go/webook/pkg/gormx"
	"learning_go/webook/pkg/migrator/events/fixer"
	"learning_go/webook/pkg/migrator/scheduler"
)

//func NewMigratorScheduler[t migrator.Entity]
//(srcDB *gorm.DB, dstDB *gorm.DB, producer sarama.SyncProducer, batchSize int) *MigratorScheduler[t]

func InitMigratorScheduler(dw *gormx.DoubleWritePool, db1 SrcDB, db2 DstDB, producer sarama.SyncProducer, batchSize int) *gin.Engine {
	server := gin.Default()
	sch := scheduler.NewMigratorScheduler[dao.Interactive](dw, db1, db2, producer, batchSize)
	sg := server.Group("/migrator")
	//包变量的一大恶心之处：要用包函数，包函数依赖包变量，这里不能忘记初始化
	ginx.InitOpt(prometheus.CounterOpts{
		Namespace: "waterbucket",
		Subsystem: "webook_grpc_interactive",
		Name:      "http_migrator",
		Help:      "数据迁移业务http错误码统计",
	})
	sch.RegisterRouters(sg)
	return server
}

var topics = []string{scheduler.MigratorTopic}

func InitFixerConsumer(client sarama.Client, src SrcDB, dst DstDB) *fixer.SaramaConsumer[dao.Interactive] {
	sc, err := fixer.NewSaramaConsumer[dao.Interactive](client, src, dst, topics)
	if err != nil {
		panic(err)
	}
	return sc
}
