package kafka

import (
	"context"
	"crypto/tls"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(bootstrapServer, apiKey, apiSecret, topic string) *Producer {
	mechanism := plain.Mechanism{
		Username: apiKey,
		Password: apiSecret,
	}

	writer := &kafka.Writer{
		Addr:  kafka.TCP(bootstrapServer),
		Topic: topic,
		Transport: &kafka.Transport{
			SASL: mechanism,
			TLS:  &tls.Config{},
		},
	}

	return &Producer{writer: writer}
}

func (p *Producer) Publish(ctx context.Context, query string) error {
	return p.writer.WriteMessages(ctx,
		kafka.Message{
			Value: []byte(query),
		},
	)
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
