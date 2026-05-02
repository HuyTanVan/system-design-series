package event

import (
	"context"
	"log"

	"autocomplete/internal/kafka"
)

type AsyncEventQueue struct {
	queue    chan string
	producer *kafka.Producer
}

func NewAsyncEventQueue(p *kafka.Producer, bufferSize int) *AsyncEventQueue {
	return &AsyncEventQueue{
		queue:    make(chan string, bufferSize),
		producer: p,
	}
}

// Start background worker to consume events and publish to Kafka
func (b *AsyncEventQueue) Start() {
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
func (b *AsyncEventQueue) Emit(msg string) {
	select {
	case b.queue <- msg:
	default:
		// backpressure strategy (drop, log or retry)
		log.Println("event queue full, dropping message")
	}
}
