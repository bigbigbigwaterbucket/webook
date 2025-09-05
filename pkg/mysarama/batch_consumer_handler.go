package mysarama

import (
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"time"
)

type BatchHandler[T any] struct {
	Consume func([]*sarama.ConsumerMessage, []T) error
}

func NewBatchHandler[T any](consume func([]*sarama.ConsumerMessage, []T) error) *BatchHandler[T] {
	return &BatchHandler[T]{Consume: consume}
}

func (h *BatchHandler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *BatchHandler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *BatchHandler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	const batchSize = 10 //这里的batchSize不再是一次可以开的最大协程数，而是一次批量处理的最大消息数
	var msgAll []*sarama.ConsumerMessage
	var ts []T
	for {
		msgAll = make([]*sarama.ConsumerMessage, 0, batchSize) //size和容量
		ts = make([]T, 0, batchSize)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		done := true
		for i := 0; i < batchSize && done; i++ {
			select {
			case <-ctx.Done():
				done = false
			case msg, ok := <-msgs: //这里不会有通道关闭自动退出for循环，要手动判断
				if !ok {
					//取出的消息已经无效了，不需要append了
					cancel()
					//相当于break出for循环
					done = false
					continue
				}
				var t T
				err := json.Unmarshal(msg.Value, &t)
				if err != nil {
					zap.L().Error("消息反序列化失败", zap.String("topic", msg.Topic),
						zap.Int32("Partition", msg.Partition), zap.Int64("Offset", msg.Offset),
						zap.Error(err))
					session.MarkMessage(msg, "")
					continue
				}
				//保证长度一样
				msgAll = append(msgAll, msg)
				ts = append(ts, t)
			}
		}
		//捕获0条就别执行了，会凭空增加数据库调用
		if len(msgAll) == 0 {
			cancel()
			continue
		}
		err := h.Consume(msgAll, ts)
		if err != nil {
			zap.L().Error("消息批量处理失败")
		} else {
			for _, msg := range msgAll {
				session.MarkMessage(msg, "")
			}
		}
		cancel()
	}

	//for msg := range msgs {
	//	var t T
	//	err := json.Unmarshal(msg.Value, &t)
	//	if err != nil {
	//		zap.L().Error("消息反序列化失败", zap.String("topic", msg.Topic),
	//			zap.Int32("Partition", msg.Partition), zap.Int64("Offset", msg.Offset),
	//			zap.Error(err))
	//		//不中断，持续消费
	//		session.MarkMessage(msg, "")
	//	}
	//	err = h.Consume(msg, t)
	//	if err != nil {
	//		zap.L().Error("消息消费失败", zap.String("topic", msg.Topic),
	//			zap.Int32("Partition", msg.Partition), zap.Int64("Offset", msg.Offset),
	//			zap.Error(err))
	//	}
	//	session.MarkMessage(msg, "")
	//}
}
