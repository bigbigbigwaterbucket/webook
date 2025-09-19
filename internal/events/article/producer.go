package articleEvent

import (
	"encoding/json"
	"github.com/IBM/sarama"
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
