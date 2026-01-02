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
	ExchangeName = "cityconnect.notifications"

	QueueStatusUpdates = "queue.status_updates"
	QueueReportCreated = "queue.report_created"
	QueueVoteReceived  = "queue.vote_received"

	RoutingKeyStatusUpdate  = "report.status.updated"
	RoutingKeyReportCreated = "report.created"
	RoutingKeyVoteReceived  = "report.vote.received"

	reconnectDelay = 5 * time.Second
	publishTimeout = 5 * time.Second
)

type QueueBinding struct {
	QueueName  string
	RoutingKey string
}

var QueueBindings = []QueueBinding{
	{QueueStatusUpdates, RoutingKeyStatusUpdate},
	{QueueReportCreated, RoutingKeyReportCreated},
	{QueueVoteReceived, RoutingKeyVoteReceived},
}

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
		return fmt.Errorf("dial: %w", err)
	}

	r.channel, err = r.conn.Channel()
	if err != nil {
		r.conn.Close()
		return fmt.Errorf("channel: %w", err)
	}

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
		return fmt.Errorf("exchange declare: %w", err)
	}

	for _, qb := range QueueBindings {
		_, err = r.channel.QueueDeclare(
			qb.QueueName,
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,
		)
		if err != nil {
			return fmt.Errorf("queue declare %s: %w", qb.QueueName, err)
		}

		err = r.channel.QueueBind(
			qb.QueueName,
			qb.RoutingKey,
			ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("bind %s->%s: %w", qb.QueueName, qb.RoutingKey, err)
		}
	}

	log.Println("rabbitmq: connected")
	return nil
}

func (r *RabbitMQ) handleReconnect() {
	for {
		select {
		case <-r.done:
			return
		case err := <-r.conn.NotifyClose(make(chan *amqp.Error)):
			if err != nil {
				log.Printf("rabbitmq: disconnected: %v", err)
			}

			r.mu.Lock()
			for {
				if err := r.connect(); err != nil {
					log.Printf("rabbitmq: reconnect failed: %v", err)
					time.Sleep(reconnectDelay)
					continue
				}
				break
			}
			r.mu.Unlock()
		}
	}
}

func (r *RabbitMQ) publish(routingKey string, message interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.channel == nil {
		return fmt.Errorf("channel not available")
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()

	err = r.channel.PublishWithContext(
		ctx,
		ExchangeName,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

func (r *RabbitMQ) PublishStatusUpdate(msg StatusUpdateMessage) error {
	return r.publish(RoutingKeyStatusUpdate, msg)
}

func (r *RabbitMQ) PublishReportCreated(msg ReportCreatedMessage) error {
	return r.publish(RoutingKeyReportCreated, msg)
}

func (r *RabbitMQ) PublishVoteReceived(msg VoteReceivedMessage) error {
	return r.publish(RoutingKeyVoteReceived, msg)
}

func (r *RabbitMQ) ConsumeQueue(queueName string) (<-chan amqp.Delivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.channel == nil {
		return nil, fmt.Errorf("channel not available")
	}

	msgs, err := r.channel.Consume(
		queueName,
		"",    // consumer tag (auto-generated)
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("consume %s: %w", queueName, err)
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
}
