package validator

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ekit/syncx/atomicx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"learning_go/webook/pkg/migrator"
	"learning_go/webook/pkg/migrator/events"
	"learning_go/webook/pkg/migrator/events/fixer"
	"reflect"
	"time"
)

type Validator[t migrator.Entity] struct {
	base      *gorm.DB
	target    *gorm.DB
	producer  fixer.Producer
	direction string //为了支持通过配置文件热更新参数，需要放在结构体字段里，表示以哪个数据库为准
	batchSize int
	highload  *atomicx.Value[bool]
	uTime     int64 //全量校验时置为0即可，当然全量校验也可以限制时间，如果你数据库数据很多
	// <=0 说明直接退出校验循环
	// > 0 真的 sleep
	sleepInterval time.Duration //<=0表示全量校验，>0表示增量校验
	orderByWt     func(ctx context.Context, offset int) (t, error)
	//游标，用于保证增量校验时不漏数据
	lastUtime int64
	lastId    int64
}

func NewValidator[t migrator.Entity](base *gorm.DB, target *gorm.DB, producer fixer.Producer,
	direction string, batchSize int, uTime int64, sleepInterval time.Duration) *Validator[t] {
	res := &Validator[t]{base: base, target: target, producer: producer,
		direction: direction, batchSize: batchSize,
		highload: atomicx.NewValueOf[bool](false), uTime: uTime, sleepInterval: sleepInterval,
	}
	res.orderByWt = res.fullOrderById
	return res
}

func (v *Validator[t]) Utime(utime int64) *Validator[t] {
	v.uTime = utime
	return v
}

func (v *Validator[t]) SleepInterval(slv time.Duration) *Validator[t] {
	v.sleepInterval = slv
	return v
}

func (v *Validator[t]) Intr() *Validator[t] {
	//每次重新开启增量校验，都会更新lastUtime
	v.lastUtime = v.uTime
	v.orderByWt = v.incrOrderByCursor
	return v
}

func (v *Validator[t]) fullOrderById(ctx context.Context, offset int) (t, error) {
	var res t
	err := v.base.WithContext(ctx).Where("u_time > ? ", v.uTime).Order("u_time ASC,id ASC").Offset(offset).
		First(&res).Error
	return res, err
}

func (v *Validator[t]) incrOrderByCursor(ctx context.Context, offset int) (t, error) {
	//incr基于游标而不是offset
	var res t
	err := v.base.WithContext(ctx).Order("u_time ASC,id ASC").
		Where("u_time > ? or (u_time = ? and id > ?)", v.lastUtime, v.lastUtime, v.lastId).
		First(&res).Error
	v.lastUtime = res.Utime()
	v.lastId = res.ID()
	return res, err
}

func (v *Validator[t]) Validate(ctx context.Context) error {
	var eg errgroup.Group
	eg.Go(func() error {
		v.ValidateBaseToTarget(ctx)
		return nil
	})
	eg.Go(func() error {
		v.ValidateTargetToBase(ctx)
		return nil
	})
	return eg.Wait()
}

// 理论上来说，可以利用 count 来加速这个过程，
// 我举个例子，假如说你初始化目标表的数据是 昨天的 23:59:59 导出来的
// 那么你可以 COUNT(*) WHERE ctime < 今天的零点，count 如果相等，就说明没删除
// 这一步大多数情况下效果很好，尤其是那些软删除的。
// 如果 count 不一致，那么接下来，你理论上来说，还可以分段 count
// 比如说，我先 count 第一个月的数据，一旦有数据删除了，你还得一条条查出来

// 由于存在base库中数据被删除，但是target库仍然存在的情况，需要反向检查删除多了的数据
// 软删除其实不需要反向校验
func (v *Validator[t]) ValidateTargetToBase(ctx context.Context) {
	offset := 0
	//不能和另一个校验共享lastId和lastUtime
	lastUtime := v.uTime
	lastId := int64(0)
	for {
		if v.highload.Load() {
			//高负荷，挂起等待负荷降低
		}
		var datas []t
		var err error
		if v.sleepInterval <= 0 {
			err = v.target.WithContext(ctx).
				Where("u_time > ?", v.uTime).
				Select("id").
				// WHERE 条件二分查找 COUNT
				Offset(offset).Limit(v.batchSize).
				Order("id").Find(&datas).Error
		} else {
			err = v.target.WithContext(ctx).
				Where("u_time > ? or (u_time = ? and id > ?)", lastUtime, lastUtime, lastId).
				Limit(v.batchSize).Order("u_time ASC,id ASC").
				Find(&datas).Error
			if len(datas) == 0 {
				time.Sleep(v.sleepInterval)
				continue
			}
			//最后一个是u_time最大的，id最大的
			lastUtime = datas[len(datas)-1].Utime()
			lastId = datas[len(datas)-1].ID()
		}
		//这里需要额外判断，不能依赖gorm.ErrRecordNotFound，因为查询多条数据时不会返回notFound错误
		//上面已经判断了
		//if len(datas) == 0 {
		//	if v.sleepInterval <= 0 {
		//		return
		//	}
		//	time.Sleep(v.sleepInterval)
		//	continue
		//}
		switch err {
		case context.DeadlineExceeded, context.Canceled: //超时或被主动停止
			return
		case nil:
			ids := slice.Map[t, int64](datas, func(idx int, src t) int64 {
				return src.ID()
			})
			var dataTargets []t
			er := v.base.WithContext(ctx).Where("id in ?", ids).Find(&dataTargets).Error
			switch er {
			case nil:
				targetIds := slice.Map[t, int64](dataTargets, func(idx int, src t) int64 {
					return src.ID()
				})
				//只支持可比较类型
				diffs := slice.DiffSet(ids, targetIds)
				v.notifyBaseMissing(diffs)
			case gorm.ErrRecordNotFound:
				v.notifyBaseMissing(ids)
			default:
				zap.L().Error("重检查询出错", zap.Error(err))
			}
		case gorm.ErrRecordNotFound:
			if v.sleepInterval <= 0 {
				return
			}
			time.Sleep(v.sleepInterval)
			continue

		default:
			zap.L().Error("重检查询出错", zap.Error(err))
			//这里不continue，以防这一个批次数据出错导致offset卡住
		}
		offset += len(datas)
		if len(datas) < v.batchSize {
			if v.sleepInterval <= 0 {
				return
			}
			time.Sleep(v.sleepInterval)
		}
	}
}

func (v *Validator[t]) ValidateBaseToTarget(ctx context.Context) {
	offset := 0
	for {
		if v.highload.Load() {
			//高负荷，挂起等待负荷降低
		}
		var data t
		data, err := v.orderByWt(ctx, offset)
		//按id升序，保证后续插入的数据不影响offset，即不重复查
		//也可以按照例如CTime等列排序
		//err := v.base.WithContext(ctx).Model(&data).Where("u_time > ?", v.uTime).
		//	Order("id").Offset(offset).First(&data).Error
		switch err {
		case context.Canceled, context.DeadlineExceeded:
			return
		case nil:
			// 准备比较数据
			var dataTarget t
			er := v.target.Model(&data).Where("id = ?", data.ID()).First(&dataTarget).Error
			switch er {
			case nil:
				//data==dataTarget是不行的，结构体不能直接比较
				//reflect.DeepEqual(data,dataTarget)  动态确定类型并比较，原则上可以（不能异构场景
				var tType any
				tType = data
				//检查是否实现CompareTo方法
				src, ok := tType.(interface{ CompareTo(e migrator.Entity) bool })
				if !ok {
					if !reflect.DeepEqual(data, dataTarget) {
						v.notify(data.ID(), events.InconsistentUnEqual)
					}
				} else {
					if !src.CompareTo(dataTarget) {
						v.notify(data.ID(), events.InconsistentUnEqual)
					}
				}
			case gorm.ErrRecordNotFound:
				//base有的target没有，需要修复
				v.notify(data.ID(), events.InconsistentTargetMissing)
			default:
				zap.L().Error("target库查询出错", zap.Error(err))
			}
		case gorm.ErrRecordNotFound:
			//offset到底了，全量校验结束了
			if v.sleepInterval <= 0 {
				return
			}
			time.Sleep(v.sleepInterval)
			continue
		default:
			//可以不管，也可以认为数据不一致，尝试去修复（虽然修了也大概率没用
			zap.L().Error("base库查询出错", zap.Error(err))
		}
		offset++
	}
}

func (v *Validator[t]) notify(id int64, typ string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	//目标库对象不一致，以base为准
	er := v.producer.ProduceInconsistentMessage(ctx, events.InconsistentEvent{ID: id, Direction: v.direction, Type: typ})
	cancel()
	if er != nil {
		zap.L().Error("发送数据不一致消息失败", zap.Error(er))
	}
}

func (v *Validator[t]) notifyBaseMissing(ids []int64) {
	for _, id := range ids {
		v.notify(id, events.InconsistentBaseMissing)
	}
}
