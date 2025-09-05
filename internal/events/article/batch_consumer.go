package articleEvent

import (
	"context"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"learning_go/webook/internal/repository"
	"learning_go/webook/pkg/mysarama"
	"time"
)

type InteractiveReadEventBatchConsumer struct {
	client sarama.Client //传client而不是consumer是因为这是一个类似于数据库服务的独立服务
	repo   repository.InteractiveRepository
}

func NewInteractiveReadEventBatchConsumer(client sarama.Client, repo repository.InteractiveRepository) *InteractiveReadEventBatchConsumer {
	return &InteractiveReadEventBatchConsumer{client: client, repo: repo}
}

func (i *InteractiveReadEventBatchConsumer) Consume(msgs []*sarama.ConsumerMessage, read []ReadEvent) error {
	//长度应当是相等的，，，但如果你不信任同事/自己，还要再判断下
	bizs := make([]string, 0, len(read))
	aids := make([]int64, 0, len(read))
	for i := 0; i < len(read); i++ {
		bizs = append(bizs, "article")
		aids = append(aids, read[i].Aid)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := i.repo.IncreaseReadCountN(ctx, bizs, aids)
	return err
}

func (i *InteractiveReadEventBatchConsumer) Start() error {
	cg, err := sarama.NewConsumerGroupFromClient("interactive", i.client)
	if err != nil {
		return err
	}
	go func() {
		//开一个协程进入无穷循环
		err = cg.Consume(context.Background(), []string{topicReadEvent}, mysarama.NewBatchHandler[ReadEvent](i.Consume))
		if err != nil {
			//你没写退出，应该不会到这里
			zap.L().Error("消费错误，退出消费循环", zap.Error(err))
		}
	}()
	return nil
}
