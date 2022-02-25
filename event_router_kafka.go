package dynproxy

import (
	"context"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"strings"
)

const (
	KafkaBrokersProp = "event_kafka_brokers"
	KafkaTopicProp   = "event_kafka_topic"
)

type KafkaEventRouter struct {
	ctx      context.Context
	producer *kafka.Writer
}

func InitEventRouter(ctx context.Context, conf map[string]interface{}) {
	kafkaEventRouter := &KafkaEventRouter{
		ctx: ctx,
	}
	err := kafkaEventRouter.configure(conf)
	if err != nil {

	}
	eventRouter = kafkaEventRouter

}

func (kef *KafkaEventRouter) Process(key string, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	message := kafka.Message{
		Key:   []byte(key),
		Value: data,
	}
	return kef.producer.WriteMessages(kef.ctx, message)
}

func (kef *KafkaEventRouter) configure(conf map[string]interface{}) error {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(getBrokers(conf)...),
		Topic:        getTopic(conf),
		RequiredAcks: kafka.RequireOne,
		Async:        true,
		Balancer:     &kafka.RoundRobin{},
		//Compression:  kafka.Lz4,
	}
	kef.producer = writer
	return nil
}

func getTopic(conf map[string]interface{}) string {
	if topicValue, ok := conf[KafkaTopicProp]; ok {
		if topic, ok := topicValue.(string); ok {
			return topic
		}
	}
	log.Fatal().Msgf("Incorrect topic name for event kafka router: %+v", conf)
	return ""
}

func getBrokers(conf map[string]interface{}) []string {
	if brokersValue, ok := conf[KafkaBrokersProp]; ok {
		if brokers, ok := brokersValue.(string); ok {
			return strings.Split(brokers, ",")
		}
	}
	log.Fatal().Msgf("Incorrect brokers url for event kafka router: %+v", conf)
	return nil
}
