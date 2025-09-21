package ioc

import (
	"github.com/IBM/sarama"
	"github.com/spf13/viper"
	"learning_go/webook/interactive/events"
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
	client, err := sarama.NewClient(address, saramaConfig)
	if err != nil {
		panic(err)
	}
	return client
}

func InitConsumers(consumer *events.InteractiveReadEventBatchConsumer) []mysarama.Consumer {
	return []mysarama.Consumer{consumer}
}
