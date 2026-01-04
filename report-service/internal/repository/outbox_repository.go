package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OutboxMessage struct {
	ID          uuid.UUID       `json:"id"`
	RoutingKey  string          `json:"routing_key"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   time.Time       `json:"created_at"`
	PublishedAt *time.Time      `json:"published_at,omitempty"`
	RetryCount  int             `json:"retry_count"`
	LastError   *string         `json:"last_error,omitempty"`
	Status      string          `json:"status"`
}

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) CreateInTransaction(tx *sql.Tx, routingKey string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO outbox_messages (id, routing_key, payload, status)
		VALUES ($1, $2, $3, 'pending')
	`
	_, err = tx.Exec(query, uuid.New(), routingKey, payloadBytes)
	return err
}

func (r *OutboxRepository) Create(routingKey string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO outbox_messages (id, routing_key, payload, status)
		VALUES ($1, $2, $3, 'pending')
	`
	_, err = r.db.Exec(query, uuid.New(), routingKey, payloadBytes)
	return err
}

func (r *OutboxRepository) GetPendingMessages(limit int) ([]OutboxMessage, error) {
	query := `
		SELECT id, routing_key, payload, created_at, retry_count, last_error, status
		FROM outbox_messages
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []OutboxMessage
	for rows.Next() {
		var m OutboxMessage
		var lastError sql.NullString
		err := rows.Scan(
			&m.ID,
			&m.RoutingKey,
			&m.Payload,
			&m.CreatedAt,
			&m.RetryCount,
			&lastError,
			&m.Status,
		)
		if err != nil {
			return nil, err
		}
		if lastError.Valid {
			m.LastError = &lastError.String
		}
		messages = append(messages, m)
	}

	return messages, nil
}

func (r *OutboxRepository) MarkAsPublished(id uuid.UUID) error {
	query := `
		UPDATE outbox_messages
		SET status = 'published', published_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *OutboxRepository) MarkAsFailed(id uuid.UUID, errMsg string) error {
	query := `
		UPDATE outbox_messages
		SET retry_count = retry_count + 1, last_error = $2,
		    status = CASE WHEN retry_count >= 5 THEN 'failed' ELSE 'pending' END
		WHERE id = $1
	`
	_, err := r.db.Exec(query, id, errMsg)
	return err
}

func (r *OutboxRepository) DeletePublished(olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM outbox_messages
		WHERE status = 'published' AND published_at < $1
	`
	result, err := r.db.Exec(query, time.Now().Add(-olderThan))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *OutboxRepository) GetStats() (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM outbox_messages
		GROUP BY status
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}

	return stats, nil
}

func (r *OutboxRepository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}
