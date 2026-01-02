package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName            = "cityconnect.notifications"
	QueueName               = "notification.events"
	RoutingKeyStatusUpdate  = "report.status.updated"
	RoutingKeyReportCreated = "report.created"
	RoutingKeyVoteReceived  = "report.vote.received"

	reconnectDelay = 5 * time.Second
	publishTimeout = 5 * time.Second
)

// Message types
const (
	MessageTypeStatusUpdate  = "status_update"
	MessageTypeReportCreated = "report_created"
	MessageTypeVoteReceived  = "vote_received"
)

type StatusUpdateMessage struct {
	ReportID    string `json:"report_id"`
	ReportTitle string `json:"report_title"`
	NewStatus   string `json:"new_status"`
	ReporterID  string `json:"reporter_id,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}

type ReportCreatedMessage struct {
	ReportID     string `json:"report_id"`
	ReportTitle  string `json:"report_title"`
	CategoryID   int    `json:"category_id"`
	CategoryName string `json:"category_name"`
	ReporterID   string `json:"reporter_id,omitempty"`
	ReporterName string `json:"reporter_name,omitempty"`
	PrivacyLevel string `json:"privacy_level"`
	Timestamp    int64  `json:"timestamp"`
}

type VoteReceivedMessage struct {
	ReportID    string `json:"report_id"`
	ReportTitle string `json:"report_title"`
	ReporterID  string `json:"reporter_id"`
	VoterID     string `json:"voter_id"`
	VoteType    string `json:"vote_type"`
	NewScore    int    `json:"new_score"`
	Timestamp   int64  `json:"timestamp"`
}

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	url     string
	mu      sync.RWMutex
	done    chan struct{}
}

func NewRabbitMQ(host, port, user, password string) (*RabbitMQ, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", user, password, host, port)

	rmq := &RabbitMQ{
		url:  url,
		done: make(chan struct{}),
	}

	if err := rmq.connect(); err != nil {
		return nil, err
	}

	go rmq.handleReconnect()

	return rmq, nil
}

func (r *RabbitMQ) connect() error {
	var err error

	r.conn, err = amqp.Dial(r.url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	r.channel, err = r.conn.Channel()
	if err != nil {
		r.conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
	err = r.channel.ExchangeDeclare(
		ExchangeName,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	_, err = r.channel.QueueDeclare(
		QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange with multiple routing keys
	routingKeys := []string{RoutingKeyStatusUpdate, RoutingKeyReportCreated, RoutingKeyVoteReceived}
	for _, key := range routingKeys {
		err = r.channel.QueueBind(
			QueueName,
			key,
			ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue with key %s: %w", key, err)
		}
	}

	log.Println("RabbitMQ connected and configured with all routing keys")
	return nil
}

func (r *RabbitMQ) handleReconnect() {
	for {
		select {
		case <-r.done:
			return
		case err := <-r.conn.NotifyClose(make(chan *amqp.Error)):
			if err != nil {
				log.Printf("RabbitMQ connection lost: %v. Reconnecting...", err)
			}

			r.mu.Lock()
			for {
				if err := r.connect(); err != nil {
					log.Printf("Failed to reconnect: %v. Retrying in %v...", err, reconnectDelay)
					time.Sleep(reconnectDelay)
					continue
				}
				break
			}
			r.mu.Unlock()
		}
	}
}

// Generic publish function
func (r *RabbitMQ) publish(routingKey string, message interface{}, logMsg string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.channel == nil {
		return fmt.Errorf("channel not available")
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()

	err = r.channel.PublishWithContext(
		ctx,
		ExchangeName,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("%s", logMsg)
	return nil
}

func (r *RabbitMQ) PublishStatusUpdate(msg StatusUpdateMessage) error {
	return r.publish(RoutingKeyStatusUpdate, msg, fmt.Sprintf("Published status update for report %s", msg.ReportID))
}

func (r *RabbitMQ) PublishReportCreated(msg ReportCreatedMessage) error {
	return r.publish(RoutingKeyReportCreated, msg, fmt.Sprintf("Published report created: %s", msg.ReportID))
}

func (r *RabbitMQ) PublishVoteReceived(msg VoteReceivedMessage) error {
	return r.publish(RoutingKeyVoteReceived, msg, fmt.Sprintf("Published vote received for report %s", msg.ReportID))
}

func (r *RabbitMQ) Consume() (<-chan amqp.Delivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.channel == nil {
		return nil, fmt.Errorf("channel not available")
	}

	msgs, err := r.channel.Consume(
		QueueName,
		"",    // consumer tag
		false, // auto-ack (false = manual ack for reliability)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register consumer: %w", err)
	}

	return msgs, nil
}

func (r *RabbitMQ) Close() {
	close(r.done)

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}

	log.Println("RabbitMQ connection closed")
}
