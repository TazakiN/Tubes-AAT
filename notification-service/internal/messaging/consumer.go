package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"notification-service/internal/model"
	"notification-service/internal/repository"

	"github.com/avast/retry-go"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	maxRetryAttempts = 3
	initialDelay     = 1 * time.Second
	maxDelay         = 30 * time.Second
)

type SSEClient struct {
	UserID  uuid.UUID
	Channel chan *model.Notification
}

type SSEHub struct {
	clients    map[uuid.UUID][]*SSEClient
	register   chan *SSEClient
	unregister chan *SSEClient
	broadcast  chan *model.Notification
	mu         sync.RWMutex
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients:    make(map[uuid.UUID][]*SSEClient),
		register:   make(chan *SSEClient),
		unregister: make(chan *SSEClient),
		broadcast:  make(chan *model.Notification, 100),
	}
}

func (h *SSEHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = append(h.clients[client.UserID], client)
			h.mu.Unlock()
			log.Printf("sse: client registered for user %s", client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			userClients := h.clients[client.UserID]
			for i, c := range userClients {
				if c == client {
					h.clients[client.UserID] = append(userClients[:i], userClients[i+1:]...)
					break
				}
			}
			h.mu.Unlock()
			close(client.Channel)
			log.Printf("sse: client unregistered for user %s", client.UserID)

		case notification := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[notification.UserID]
			for _, client := range clients {
				select {
				case client.Channel <- notification:
				default:
					// channel full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *SSEHub) RegisterClient(userID uuid.UUID) *SSEClient {
	client := &SSEClient{
		UserID:  userID,
		Channel: make(chan *model.Notification, 10),
	}
	h.register <- client
	return client
}

func (h *SSEHub) UnregisterClient(client *SSEClient) {
	h.unregister <- client
}

func (h *SSEHub) SendToUser(notification *model.Notification) {
	h.broadcast <- notification
}

// NotificationConsumer consumes messages from RabbitMQ queues
type NotificationConsumer struct {
	rmq              *RabbitMQ
	notificationRepo *repository.NotificationRepository
	sseHub           *SSEHub
	done             chan struct{}
	wg               sync.WaitGroup
}

func NewNotificationConsumer(rmq *RabbitMQ, notificationRepo *repository.NotificationRepository, sseHub *SSEHub) *NotificationConsumer {
	return &NotificationConsumer{
		rmq:              rmq,
		notificationRepo: notificationRepo,
		sseHub:           sseHub,
		done:             make(chan struct{}),
	}
}

func (c *NotificationConsumer) Start() {
	c.wg.Add(3)
	go c.consumeQueue(QueueStatusUpdates, c.handleStatusUpdate)
	go c.consumeQueue(QueueReportCreated, c.handleReportCreated)
	go c.consumeQueue(QueueVoteReceived, c.handleVoteReceived)
	log.Println("consumers started")
}

func (c *NotificationConsumer) consumeQueue(queueName string, handler func(amqp.Delivery) error) {
	defer c.wg.Done()

	for {
		select {
		case <-c.done:
			log.Printf("consumer %s: stopping", queueName)
			return
		default:
			msgs, err := c.rmq.ConsumeQueue(queueName)
			if err != nil {
				log.Printf("consumer %s: error %v, retrying in 5s...", queueName, err)
				time.Sleep(5 * time.Second)
				continue
			}

			log.Printf("consumer %s: listening for messages", queueName)
			c.processQueue(queueName, msgs, handler)
		}
	}
}

func (c *NotificationConsumer) processQueue(queueName string, msgs <-chan amqp.Delivery, handler func(amqp.Delivery) error) {
	for {
		select {
		case <-c.done:
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Printf("consumer %s: channel closed, reconnecting...", queueName)
				return
			}
			c.processMessageWithRetry(queueName, msg, handler)
		}
	}
}

// processMessageWithRetry handles retry with backoff
func (c *NotificationConsumer) processMessageWithRetry(queueName string, msg amqp.Delivery, handler func(amqp.Delivery) error) {
	messageID := msg.MessageId
	if messageID == "" {
		messageID = fmt.Sprintf("%x", msg.Body[:min(32, len(msg.Body))])
	}

	processed, err := c.notificationRepo.IsMessageProcessed(messageID)
	if err != nil {
		log.Printf("%s: idempotency check failed: %v", queueName, err)
	}
	if processed {
		log.Printf("%s: %s already processed", queueName, messageID)
		msg.Ack(false)
		return
	}

	err = retry.Do(
		func() error {
			return handler(msg)
		},
		retry.Attempts(maxRetryAttempts),
		retry.Delay(initialDelay),
		retry.MaxDelay(maxDelay),
		retry.DelayType(retry.BackOffDelay),
		retry.OnRetry(func(n uint, err error) {
			log.Printf("%s: retry %d: %v", queueName, n+1, err)
		}),
	)

	if err != nil {
		log.Printf("%s: failed, sending to DLQ: %v", queueName, err)
		msg.Nack(false, false)
		return
	}

	if err := c.notificationRepo.MarkMessageProcessed(messageID); err != nil {
		log.Printf("%s: mark processed failed: %v", queueName, err)
	}

	msg.Ack(false)
}

func (c *NotificationConsumer) handleStatusUpdate(msg amqp.Delivery) error {
	var statusUpdate model.StatusUpdateMessage
	if err := json.Unmarshal(msg.Body, &statusUpdate); err != nil {
		log.Printf("status_update: bad json: %v", err)
		return nil
	}

	reportID, err := uuid.Parse(statusUpdate.ReportID)
	if err != nil {
		log.Printf("status_update: bad report_id: %v", err)
		return nil
	}

	err = c.notificationRepo.CreateStatusNotification(
		reportID,
		model.ReportStatus(statusUpdate.NewStatus),
		statusUpdate.ReportTitle,
	)
	if err != nil {
		return err
	}

	// Send SSE notification
	if statusUpdate.ReporterID != "" {
		reporterID, err := uuid.Parse(statusUpdate.ReporterID)
		if err == nil {
			notification := &model.Notification{
				ID:        uuid.New(),
				UserID:    reporterID,
				ReportID:  &reportID,
				Title:     "Status Laporan Diperbarui",
				Message:   "Laporan \"" + statusUpdate.ReportTitle + "\" telah diubah statusnya menjadi: " + statusUpdate.NewStatus,
				IsRead:    false,
				CreatedAt: time.Now(),
			}
			c.sseHub.SendToUser(notification)
		}
	}

	return nil
}

func (c *NotificationConsumer) handleReportCreated(msg amqp.Delivery) error {
	var reportCreated model.ReportCreatedMessage
	if err := json.Unmarshal(msg.Body, &reportCreated); err != nil {
		log.Printf("report_created: bad json: %v", err)
		return nil
	}

	log.Printf("report_created: %s by %s", reportCreated.ReportID, reportCreated.ReporterName)
	// TODO: notify admins
	return nil
}

func (c *NotificationConsumer) handleVoteReceived(msg amqp.Delivery) error {
	var voteReceived model.VoteReceivedMessage
	if err := json.Unmarshal(msg.Body, &voteReceived); err != nil {
		log.Printf("vote: bad json: %v", err)
		return nil
	}

	reportID, err := uuid.Parse(voteReceived.ReportID)
	if err != nil {
		log.Printf("vote: bad report_id: %v", err)
		return nil
	}

	// skip if voter is reporter
	if voteReceived.ReporterID != "" && voteReceived.ReporterID != voteReceived.VoterID {
		reporterID, err := uuid.Parse(voteReceived.ReporterID)
		if err == nil {
			voteText := "upvote"
			if voteReceived.VoteType == "downvote" {
				voteText = "downvote"
			}

			notification := &model.Notification{
				ID:        uuid.New(),
				UserID:    reporterID,
				ReportID:  &reportID,
				Title:     "Laporan Mendapat " + voteText,
				Message:   "Laporan \"" + voteReceived.ReportTitle + "\" mendapat " + voteText + ". Skor: " + fmt.Sprintf("%d", voteReceived.NewScore),
				IsRead:    false,
				CreatedAt: time.Now(),
			}

			if err := c.notificationRepo.Create(notification); err != nil {
				return err
			}

			// Send SSE notification
			c.sseHub.SendToUser(notification)
		}
	}

	return nil
}

func (c *NotificationConsumer) Stop() {
	close(c.done)
	c.wg.Wait()
	log.Println("consumers stopped")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
