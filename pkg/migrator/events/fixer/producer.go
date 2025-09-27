package fixer

import (
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
	"learning_go/webook/pkg/migrator/events"
)

type Producer interface {
	ProduceInconsistentMessage(ctx context.Context, eve events.InconsistentEvent) error
}

type SaramaProducer struct {
	p     sarama.SyncProducer
	topic string
}

func NewSaramaProducer(p sarama.SyncProducer, topic string) *SaramaProducer {
	return &SaramaProducer{p: p, topic: topic}
}

func (s *SaramaProducer) ProduceInconsistentMessage(ctx context.Context, eve events.InconsistentEvent) error {
	data, err := json.Marshal(eve)
	if err != nil {
		return err
	}
	_, _, err = s.p.SendMessage(&sarama.ProducerMessage{Topic: s.topic, Key: sarama.ByteEncoder(data)})
	return err
}
