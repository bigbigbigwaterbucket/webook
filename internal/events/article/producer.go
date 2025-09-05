package articleEvent

import (
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"learning_go/webook/internal/repository"
	"learning_go/webook/pkg/mysarama"
	"time"
)

type ReadEvent struct {
	Aid int64 `json:"aid"`
	Uid int64 `json:"uid"`
}

const topicReadEvent = "article_read_topic"

// 事件生产者就是往消息队列中写入消息的
type Producer interface {
	ProduceReadEvent(read ReadEvent) error
}

type SaramaSyncProducer struct {
	producer sarama.SyncProducer
}

func (s *SaramaSyncProducer) ProduceReadEvent(read ReadEvent) error {
	data, err := json.Marshal(read)
	if err != nil {
		return err
	}
	_, _, err = s.producer.SendMessage(&sarama.ProducerMessage{Topic: topicReadEvent, Value: sarama.ByteEncoder(data)})
	return err
}

type InteractiveReadEventConsumer struct {
	client sarama.Client
	repo   repository.InteractiveRepository
}

func (i *InteractiveReadEventConsumer) Consume(msg *sarama.ConsumerMessage, read ReadEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := i.repo.IncreaseReadCount(ctx, "article", read.Aid)
	return err
}

func (i *InteractiveReadEventConsumer) Start() error {
	cg, err := sarama.NewConsumerGroupFromClient("interactive", i.client)
	if err != nil {
		return err
	}
	go func() {
		err = cg.Consume(context.Background(), []string{topicReadEvent}, &mysarama.Handler[ReadEvent]{Consume: i.Consume})
		if err != nil {
			zap.L().Error("消费错误，退出消费循环", zap.Error(err))
		}
	}()
	return nil
}
