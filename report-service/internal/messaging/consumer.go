package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"report-service/internal/model"
	"report-service/internal/repository"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
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
			h.clients[client.UserID] = append(h.clients[client.UserID], client)

		case client := <-h.unregister:
			userClients := h.clients[client.UserID]
			for i, c := range userClients {
				if c == client {
					h.clients[client.UserID] = append(userClients[:i], userClients[i+1:]...)
					break
				}
			}
			close(client.Channel)

		case notification := <-h.broadcast:
			clients := h.clients[notification.UserID]
			for _, client := range clients {
				select {
				case client.Channel <- notification:
				default:
					// channel full, skip
				}
			}
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

type NotificationConsumer struct {
	rmq              *RabbitMQ
	notificationRepo *repository.NotificationRepository
	sseHub           *SSEHub
	done             chan struct{}
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
	go c.consume()
}

func (c *NotificationConsumer) consume() {
	for {
		select {
		case <-c.done:
			return
		default:
			msgs, err := c.rmq.Consume()
			if err != nil {
				log.Printf("consume error: %v, retrying...", err)
				time.Sleep(5 * time.Second)
				continue
			}

			c.processMessages(msgs)
		}
	}
}

func (c *NotificationConsumer) processMessages(msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-c.done:
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Println("channel closed, reconnecting...")
				return
			}

			c.handleMessage(msg)
		}
	}
}

func (c *NotificationConsumer) handleMessage(msg amqp.Delivery) {
	routingKey := msg.RoutingKey

	switch routingKey {
	case RoutingKeyStatusUpdate:
		c.handleStatusUpdate(msg)
	case RoutingKeyReportCreated:
		c.handleReportCreated(msg)
	case RoutingKeyVoteReceived:
		c.handleVoteReceived(msg)
	default:
		msg.Nack(false, false)
	}
}

func (c *NotificationConsumer) handleStatusUpdate(msg amqp.Delivery) {
	var statusUpdate StatusUpdateMessage
	if err := json.Unmarshal(msg.Body, &statusUpdate); err != nil {
		log.Printf("unmarshal error: %v", err)
		msg.Nack(false, false)
		return
	}

	reportID, err := uuid.Parse(statusUpdate.ReportID)
	if err != nil {
		msg.Nack(false, false)
		return
	}
	err = c.notificationRepo.CreateStatusNotification(
		reportID,
		model.ReportStatus(statusUpdate.NewStatus),
		statusUpdate.ReportTitle,
	)
	if err != nil {
		log.Printf("db error: %v", err)
		msg.Nack(false, true)
		return
	}

	// kirim ke reporter via SSE
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

	msg.Ack(false)
}

func (c *NotificationConsumer) handleReportCreated(msg amqp.Delivery) {
	var reportCreated ReportCreatedMessage
	if err := json.Unmarshal(msg.Body, &reportCreated); err != nil {
		log.Printf("unmarshal error: %v", err)
		msg.Nack(false, false)
		return
	}

	// TODO: notify admins, update dashboards, etc.
	msg.Ack(false)
}

func (c *NotificationConsumer) handleVoteReceived(msg amqp.Delivery) {
	var voteReceived VoteReceivedMessage
	if err := json.Unmarshal(msg.Body, &voteReceived); err != nil {
		log.Printf("unmarshal error: %v", err)
		msg.Nack(false, false)
		return
	}

	reportID, err := uuid.Parse(voteReceived.ReportID)
	if err != nil {
		msg.Nack(false, false)
		return
	}

	// kirim notifikasi ke pemilik report (skip jika vote sendiri)
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

			// Save to database
			if err := c.notificationRepo.Create(notification); err != nil {
				log.Printf("db error: %v", err)
			}

			c.sseHub.SendToUser(notification)
		}
	}

	msg.Ack(false)
}

func (c *NotificationConsumer) Stop() {
	close(c.done)
}
