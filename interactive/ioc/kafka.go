package ioc

import (
	"github.com/IBM/sarama"
	"github.com/spf13/viper"
	"learning_go/webook/interactive/events"
	"learning_go/webook/interactive/repository/dao"
	"learning_go/webook/pkg/migrator/events/fixer"
	"learning_go/webook/pkg/mysarama"
)

func InitKafka() sarama.Client {
	type Config struct {
		Addr string `yaml:"addr"`
	}
	var config1 Config
	err := viper.UnmarshalKey("kafka", &config1)
	if err != nil {
		panic(err)
	}
	var address = []string{config1.Addr}
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	client, err := sarama.NewClient(address, saramaConfig)
	if err != nil {
		panic(err)
	}
	return client
}

func InitConsumers(intr *events.InteractiveReadEventBatchConsumer, fix *fixer.SaramaConsumer[dao.Interactive]) []mysarama.Consumer {
	return []mysarama.Consumer{intr, fix}
}

func InitSyncProducer(client sarama.Client) sarama.SyncProducer {
	p, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		panic(err)
	}
	return p
}
