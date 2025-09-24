package validator

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ekit/syncx/atomicx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"learning_go/webook/internal/migrator"
	"learning_go/webook/internal/migrator/events"
	"reflect"
	"time"
)

type Validator[t migrator.Entity] struct {
	base      *gorm.DB
	target    *gorm.DB
	producer  events.Producer
	direction string //为了支持通过配置文件热更新参数，需要放在结构体字段里，表示以哪个数据库为准
	batchSize int
	highload  *atomicx.Value[bool]
}

func NewValidator[t migrator.Entity](base *gorm.DB, target *gorm.DB, producer events.Producer, direction string, batchSize int) *Validator[t] {
	return &Validator[t]{base: base, target: target, producer: producer,
		direction: direction, batchSize: batchSize, highload: atomicx.NewValueOf[bool](false)}
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
func (v *Validator[t]) ValidateTargetToBase(ctx context.Context) {
	offset := -v.batchSize
	for {
		if v.highload.Load() {
			//高负荷，挂起等待负荷降低
		}
		offset += v.batchSize
		var datas []t
		err := v.target.WithContext(ctx).Order("id").Offset(offset).Find(&datas).Error
		switch err {
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
				continue
			}
		case gorm.ErrRecordNotFound:
			return
		default:
			zap.L().Error("重检查询出错", zap.Error(err))
			continue
		}
		if len(datas) < v.batchSize {
			return
		}
	}
}

func (v *Validator[t]) ValidateBaseToTarget(ctx context.Context) {
	offset := -1
	for {
		if v.highload.Load() {
			//高负荷，挂起等待负荷降低
		}
		offset++
		var data t
		//按id升序，保证后续插入的数据不影响offset，即不重复查
		//也可以按照例如CTime等列排序
		err := v.base.WithContext(ctx).Model(&data).Order("id").Offset(offset).First(&data).Error
		switch err {
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
				continue
			}
		case gorm.ErrRecordNotFound:
			//offset到底了，全量校验结束了
			return
		default:
			//可以不管，也可以认为数据不一致，尝试去修复（虽然修了也大概率没用
			zap.L().Error("base库查询出错", zap.Error(err))
			continue
		}
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
