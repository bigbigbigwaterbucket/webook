package scheduler

import (
	"context"
	"errors"
	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"learning_go/webook/pkg/ginx"
	"learning_go/webook/pkg/gormx"
	"learning_go/webook/pkg/migrator"
	"learning_go/webook/pkg/migrator/events/fixer"
	"learning_go/webook/pkg/migrator/validator"
	"time"
)

const migratorTopic = "migrator_topic"

type MigratorScheduler[t migrator.Entity] struct {
	doubleWritePool *gormx.DoubleWritePool
	producer        fixer.Producer
	pattern         string
	srcValidator    *validator.Validator[t]
	dstValidator    *validator.Validator[t]
	stopFunc        func()
}

func NewMigratorScheduler[t migrator.Entity](srcDB *gorm.DB, dstDB *gorm.DB, producer sarama.SyncProducer, batchSize int) *MigratorScheduler[t] {
	dw := gormx.NewDoubleWritePool(srcDB.ConnPool, dstDB.ConnPool, gormx.PatternSrcOnly)
	p := fixer.NewSaramaProducer(producer, migratorTopic)
	srcV := validator.NewValidator[t](srcDB, dstDB, p, "SRC", batchSize, 0, 0)
	dstV := validator.NewValidator[t](dstDB, srcDB, p, "DST", batchSize, 0, 0)
	return &MigratorScheduler[t]{doubleWritePool: dw, producer: p, pattern: gormx.PatternSrcOnly,
		srcValidator: srcV, dstValidator: dstV, stopFunc: func() {
		}}
}

func (m *MigratorScheduler[t]) RegisterRouters(server *gin.RouterGroup) {
	server.POST("/src_only", ginx.Wrapper(m.SrcOnly))
	server.POST("/src_first", ginx.Wrapper(m.SrcFirst))
	server.POST("/dst_only", ginx.Wrapper(m.DstOnly))
	server.POST("/dst_first", ginx.Wrapper(m.DstFirst))
	server.POST("/start_full", ginx.Wrapper(m.StartFullValidate))
	server.POST("/stop_full", ginx.Wrapper(m.StopFullValidate))
	server.POST("/start_incr", ginx.WrapperReq[IncreaseReq](m.StartIncrValidate))
	server.POST("/stop_incr", ginx.Wrapper(m.StopIncrValidate))
}

func (m *MigratorScheduler[t]) SrcOnly(ctx *gin.Context) (ginx.Result, error) {
	m.pattern = gormx.PatternSrcOnly
	m.doubleWritePool.Pattern(gormx.PatternSrcOnly)
	return ginx.Result{Msg: "OK"}, nil
}

func (m *MigratorScheduler[t]) SrcFirst(ctx *gin.Context) (ginx.Result, error) {
	m.pattern = gormx.PatternSrcFirst
	m.doubleWritePool.Pattern(gormx.PatternSrcFirst)
	return ginx.Result{Msg: "OK"}, nil
}

func (m *MigratorScheduler[t]) DstOnly(ctx *gin.Context) (ginx.Result, error) {
	m.pattern = gormx.PatternDstOnly
	m.doubleWritePool.Pattern(gormx.PatternDstOnly)
	return ginx.Result{Msg: "OK"}, nil
}

func (m *MigratorScheduler[t]) DstFirst(ctx *gin.Context) (ginx.Result, error) {
	m.pattern = gormx.PatternDstFirst
	m.doubleWritePool.Pattern(gormx.PatternDstFirst)
	return ginx.Result{Msg: "OK"}, nil
}

func (m *MigratorScheduler[t]) StartFullValidate(ctx *gin.Context) (ginx.Result, error) {
	//防止没停就开始新的
	m.stopFunc()
	ctxV, stopFunc := context.WithCancel(context.Background())
	m.stopFunc = stopFunc
	var err error
	switch m.pattern {
	case gormx.PatternSrcOnly, gormx.PatternSrcFirst:
		go func() {
			err = m.srcValidator.Utime(0).SleepInterval(0).Validate(ctxV)
		}()
	case gormx.PatternDstOnly, gormx.PatternDstFirst:
		go func() {
			err = m.dstValidator.Utime(0).SleepInterval(0).Validate(ctxV)
		}()
	default:
		return ginx.Result{Msg: "系统错误"}, errors.New("未知的pattern模式")
	}
	if err != nil {
		zap.L().Error("开启全量校验错误", zap.Error(err))
		return ginx.Result{Msg: "系统错误"}, err
	}
	return ginx.Result{Msg: "OK"}, nil
}

func (m *MigratorScheduler[t]) StopFullValidate(ctx *gin.Context) (ginx.Result, error) {
	m.stopFunc()
	return ginx.Result{Msg: "OK"}, nil
}

func (m *MigratorScheduler[t]) StartIncrValidate(ctx *gin.Context, req IncreaseReq) (ginx.Result, error) {
	m.stopFunc()
	ctxV, stopFunc := context.WithCancel(context.Background())
	m.stopFunc = stopFunc
	var err error
	switch m.pattern {
	case gormx.PatternSrcOnly, gormx.PatternSrcFirst:
		go func() {
			err = m.srcValidator.Utime(req.Utime).SleepInterval(time.Duration(req.SleepInterval)).Validate(ctxV)
		}()
	case gormx.PatternDstOnly, gormx.PatternDstFirst:
		go func() {
			err = m.dstValidator.Utime(req.Utime).SleepInterval(time.Duration(req.SleepInterval)).Validate(ctxV)
		}()
	default:
		return ginx.Result{Msg: "系统错误"}, errors.New("未知的pattern模式")
	}
	if err != nil {
		zap.L().Error("开启增量校验错误", zap.Error(err))
		return ginx.Result{Msg: "系统错误"}, err
	}
	return ginx.Result{Msg: "OK"}, nil
}

func (m *MigratorScheduler[t]) StopIncrValidate(ctx *gin.Context) (ginx.Result, error) {
	m.stopFunc()
	return ginx.Result{Msg: "OK"}, nil
}

type IncreaseReq struct {
	Utime         int64 `json:"utime"`
	SleepInterval int64 `json:"sleepInterval"`
}
