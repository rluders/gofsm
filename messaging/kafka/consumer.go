package kafka

import (
	"context"
	"log"
	"time"

	"github.com/rluders/gofsm/fsm"
	kafkago "github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader     *kafkago.Reader
	fsmEngine  *fsm.FSM
	eventCodec EventCodec
}

type Config struct {
	Brokers []string
	Topic   string
	GroupID string
}

func NewConsumer(cfg Config, engine *fsm.FSM, codec EventCodec) *Consumer {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       cfg.Topic,
		GroupID:     cfg.GroupID,
		StartOffset: kafkago.LastOffset,
		MaxWait:     1 * time.Second,
	})
	return &Consumer{
		reader:     reader,
		fsmEngine:  engine,
		eventCodec: codec,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			log.Printf("Error when reading Kafka message: %v", err)
			continue
		}

		entityID := string(m.Key)

		event, err := c.eventCodec.Decode(m.Value)
		if err != nil {
			log.Printf("Error when decoding Kafka message: %v", err)
			continue
		}

		ctxWithID := context.WithValue(ctx, "scanID", entityID)

		if err := c.fsmEngine.Trigger(ctxWithID, entityID, event); err != nil {
			log.Printf("Error when scanning Kafka message: %v", err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("Error when committing Kafka message: %v", err)
		}
	}
}
