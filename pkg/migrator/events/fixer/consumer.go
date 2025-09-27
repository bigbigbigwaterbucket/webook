package fixer

import (
	"context"
	"errors"
	"github.com/IBM/sarama"
	"gorm.io/gorm"
	"learning_go/webook/pkg/migrator"
	"learning_go/webook/pkg/migrator/events"
	"learning_go/webook/pkg/migrator/fixer"
	"learning_go/webook/pkg/mysarama"
)

type SaramaConsumer[t migrator.Entity] struct {
	client   sarama.Client
	srcFixer *fixer.Fixer[t]
	dstFixer *fixer.Fixer[t]
	topics   []string
}

func NewSaramaConsumer[t migrator.Entity](client sarama.Client, src *gorm.DB, dst *gorm.DB, topics []string) (*SaramaConsumer[t], error) {
	srcFixer, err := fixer.NewOverWriteFixer[t](src, dst)
	if err != nil {
		return nil, err
	}
	dstFixer, err := fixer.NewOverWriteFixer[t](dst, src)
	if err != nil {
		return nil, err
	}
	return &SaramaConsumer[t]{client: client, srcFixer: srcFixer, dstFixer: dstFixer, topics: topics}, nil
}

func (s *SaramaConsumer[t]) Consume(msg *sarama.ConsumerMessage, evt events.InconsistentEvent) error {
	switch evt.Direction {
	case "SRC":
		return s.srcFixer.Fix(context.Background(), evt)
	case "DST":
		return s.dstFixer.Fix(context.Background(), evt)
	default:
		return errors.New("未知的direction数据库")
	}
}

func (s *SaramaConsumer[t]) Start() error {
	cs, err := sarama.NewConsumerGroupFromClient("migrator-fix", s.client)
	if err != nil {
		return err
	}
	return cs.Consume(context.Background(), s.topics, mysarama.NewHandler[events.InconsistentEvent](s.Consume))
}
