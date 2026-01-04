package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"report-service/internal/repository"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	workerInterval     = 1 * time.Second
	batchSize          = 50
	cleanupInterval    = 1 * time.Hour
	publishedRetention = 24 * time.Hour
)

// OutboxWorker publishes messages from outbox table to RabbitMQ
type OutboxWorker struct {
	outboxRepo *repository.OutboxRepository
	rmq        *RabbitMQ
	done       chan struct{}
	wg         sync.WaitGroup
}

// NewOutboxWorker creates a new OutboxWorker
func NewOutboxWorker(outboxRepo *repository.OutboxRepository, rmq *RabbitMQ) *OutboxWorker {
	return &OutboxWorker{
		outboxRepo: outboxRepo,
		rmq:        rmq,
		done:       make(chan struct{}),
	}
}

// Start begins the outbox worker
func (w *OutboxWorker) Start() {
	w.wg.Add(2)
	go w.processLoop()
	go w.cleanupLoop()
	log.Println("outbox worker: started")
}

// processLoop continuously processes pending outbox messages
func (w *OutboxWorker) processLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(workerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			log.Println("outbox worker: process loop stopping")
			return
		case <-ticker.C:
			w.processPendingMessages()
		}
	}
}

// processPendingMessages fetches and publishes pending messages
func (w *OutboxWorker) processPendingMessages() {
	messages, err := w.outboxRepo.GetPendingMessages(batchSize)
	if err != nil {
		log.Printf("outbox worker: failed to get pending messages: %v", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	log.Printf("outbox worker: processing %d pending messages", len(messages))

	for _, msg := range messages {
		if err := w.publishWithConfirm(msg.RoutingKey, msg.Payload); err != nil {
			log.Printf("outbox worker: failed to publish message %s: %v", msg.ID, err)
			w.outboxRepo.MarkAsFailed(msg.ID, err.Error())
			continue
		}

		if err := w.outboxRepo.MarkAsPublished(msg.ID); err != nil {
			log.Printf("outbox worker: failed to mark message %s as published: %v", msg.ID, err)
		} else {
			log.Printf("outbox worker: message %s published successfully", msg.ID)
		}
	}
}

// publishWithConfirm publishes a message with publisher confirms
func (w *OutboxWorker) publishWithConfirm(routingKey string, payload json.RawMessage) error {
	w.rmq.mu.RLock()
	defer w.rmq.mu.RUnlock()

	if w.rmq.channel == nil {
		return fmt.Errorf("channel not available")
	}

	// Enable publisher confirms
	if err := w.rmq.channel.Confirm(false); err != nil {
		return fmt.Errorf("confirm mode: %w", err)
	}

	confirms := w.rmq.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()

	err := w.rmq.channel.PublishWithContext(
		ctx,
		ExchangeName,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         payload,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	// Wait for confirmation
	select {
	case confirm := <-confirms:
		if !confirm.Ack {
			return fmt.Errorf("message not acknowledged by RabbitMQ")
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("publish confirmation timeout")
	}
}

// cleanupLoop periodically cleans up old published messages
func (w *OutboxWorker) cleanupLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			log.Println("outbox worker: cleanup loop stopping")
			return
		case <-ticker.C:
			deleted, err := w.outboxRepo.DeletePublished(publishedRetention)
			if err != nil {
				log.Printf("outbox worker: cleanup failed: %v", err)
			} else if deleted > 0 {
				log.Printf("outbox worker: cleaned up %d old published messages", deleted)
			}
		}
	}
}

// Stop gracefully stops the outbox worker
func (w *OutboxWorker) Stop() {
	log.Println("outbox worker: stopping...")
	close(w.done)
	w.wg.Wait()
	log.Println("outbox worker: stopped")
}

// GetStats returns outbox statistics
func (w *OutboxWorker) GetStats() (map[string]int, error) {
	return w.outboxRepo.GetStats()
}
