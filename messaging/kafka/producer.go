package kafka

import (
	"context"
	"encoding/json"
	kafkago "github.com/segmentio/kafka-go"
)

type EventPublisher struct {
	writer *kafkago.Writer
	topic  string
}

func NewPublisher(brokers []string, topic string) *EventPublisher {
	return &EventPublisher{
		writer: &kafkago.Writer{
			Addr:     kafkago.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafkago.Hash{},
		},
		topic: topic,
	}
}

func (p *EventPublisher) Publish(ctx context.Context, key string, eventName string, payload interface{}) error {
	value, err := json.Marshal(map[string]interface{}{
		"name":    eventName,
		"payload": payload,
	})
	if err != nil {
		return err
	}

	msg := kafkago.Message{
		Key:   []byte(key),
		Value: value,
	}

	return p.writer.WriteMessages(ctx, msg)
}
