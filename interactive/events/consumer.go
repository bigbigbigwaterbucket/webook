package events

import (
	"context"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"learning_go/webook/interactive/repository"
	"learning_go/webook/pkg/mysarama"
	"time"
)

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
