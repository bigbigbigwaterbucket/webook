package mysarama

import (
	"encoding/json"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type Handler[T any] struct {
	Consume func(*sarama.ConsumerMessage, T) error
}

func (h *Handler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Handler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Handler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	for msg := range msgs {
		var t T
		err := json.Unmarshal(msg.Value, &t)
		if err != nil {
			zap.L().Error("消息反序列化失败", zap.String("topic", msg.Topic),
				zap.Int32("Partition", msg.Partition), zap.Int64("Offset", msg.Offset),
				zap.Error(err))
			//不中断，持续消费
			session.MarkMessage(msg, "")
		}
		err = h.Consume(msg, t)
		if err != nil {
			zap.L().Error("消息消费失败", zap.String("topic", msg.Topic),
				zap.Int32("Partition", msg.Partition), zap.Int64("Offset", msg.Offset),
				zap.Error(err))
		}
		session.MarkMessage(msg, "")
	}
	return nil
}
