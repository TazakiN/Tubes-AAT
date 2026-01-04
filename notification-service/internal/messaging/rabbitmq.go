package messaging

import (
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName    = "cityconnect.notifications"
	DLXExchangeName = "cityconnect.notifications.dlx"

	// Main queues
	QueueStatusUpdates = "queue.status_updates"
	QueueReportCreated = "queue.report_created"
	QueueVoteReceived  = "queue.vote_received"

	// Dead Letter Queues
	QueueStatusUpdatesDLQ = "queue.status_updates.dlq"
	QueueReportCreatedDLQ = "queue.report_created.dlq"
	QueueVoteReceivedDLQ  = "queue.vote_received.dlq"

	// Routing keys
	RoutingKeyStatusUpdate  = "report.status.updated"
	RoutingKeyReportCreated = "report.created"
	RoutingKeyVoteReceived  = "report.vote.received"

	reconnectDelay = 5 * time.Second
	prefetchCount  = 10 // QoS prefetch limit
)

type QueueConfig struct {
	QueueName     string
	RoutingKey    string
	DLQName       string
	DLQRoutingKey string
}

var QueueConfigs = []QueueConfig{
	{
		QueueName:     QueueStatusUpdates,
		RoutingKey:    RoutingKeyStatusUpdate,
		DLQName:       QueueStatusUpdatesDLQ,
		DLQRoutingKey: "dlq.status_updates",
	},
	{
		QueueName:     QueueReportCreated,
		RoutingKey:    RoutingKeyReportCreated,
		DLQName:       QueueReportCreatedDLQ,
		DLQRoutingKey: "dlq.report_created",
	},
	{
		QueueName:     QueueVoteReceived,
		RoutingKey:    RoutingKeyVoteReceived,
		DLQName:       QueueVoteReceivedDLQ,
		DLQRoutingKey: "dlq.vote_received",
	},
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

	// Set QoS prefetch limit
	if err := r.channel.Qos(prefetchCount, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	// Declare main exchange
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

	// Declare Dead Letter Exchange (DLX)
	err = r.channel.ExchangeDeclare(
		DLXExchangeName,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("dlx exchange declare: %w", err)
	}

	// Declare queues with DLQ configuration
	for _, qc := range QueueConfigs {
		// Declare DLQ first
		_, err = r.channel.QueueDeclare(
			qc.DLQName,
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			amqp.Table{
				"x-message-ttl": int64(86400000), // 24 hours TTL for DLQ messages
			},
		)
		if err != nil {
			return fmt.Errorf("dlq declare %s: %w", qc.DLQName, err)
		}

		// Bind DLQ to DLX
		err = r.channel.QueueBind(
			qc.DLQName,
			qc.DLQRoutingKey,
			DLXExchangeName,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("dlq bind %s: %w", qc.DLQName, err)
		}

		// Declare main queue with DLX configuration
		_, err = r.channel.QueueDeclare(
			qc.QueueName,
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			amqp.Table{
				"x-dead-letter-exchange":    DLXExchangeName,
				"x-dead-letter-routing-key": qc.DLQRoutingKey,
			},
		)
		if err != nil {
			return fmt.Errorf("queue declare %s: %w", qc.QueueName, err)
		}

		// Bind main queue to main exchange
		err = r.channel.QueueBind(
			qc.QueueName,
			qc.RoutingKey,
			ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("bind %s->%s: %w", qc.QueueName, qc.RoutingKey, err)
		}
	}

	log.Println("rabbitmq: connected with DLQ configuration")
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

func (r *RabbitMQ) ConsumeQueue(queueName string) (<-chan amqp.Delivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.channel == nil {
		return nil, fmt.Errorf("channel not available")
	}

	msgs, err := r.channel.Consume(
		queueName,
		"",    // consumer tag (auto-generated)
		false, // auto-ack (manual for retry support)
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

func (r *RabbitMQ) GetChannel() *amqp.Channel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.channel
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
