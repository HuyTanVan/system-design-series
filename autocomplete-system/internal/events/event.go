package events

import (
	"context"
	"log"

	"autocomplete/internal/kafka"
)

type EventBus struct {
	queue    chan string
	producer *kafka.Producer
}

func NewEventBus(p *kafka.Producer, bufferSize int) *EventBus {
	return &EventBus{
		queue:    make(chan string, bufferSize),
		producer: p,
	}
}

// Start background worker
func (b *EventBus) Start() {
	go func() {
		for msg := range b.queue {
			err := b.producer.Publish(context.Background(), msg)
			if err != nil {
				log.Printf("Kafka publish failed: %v", err)
			}
		}
	}()
}

// Non-blocking send
func (b *EventBus) Emit(msg string) {
	select {
	case b.queue <- msg:
	default:
		// backpressure strategy (drop or log)
		log.Println("event queue full, dropping message")
	}
}
