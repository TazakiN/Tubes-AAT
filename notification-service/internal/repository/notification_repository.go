package repository

import (
	"database/sql"
	"time"

	"notification-service/internal/model"

	"github.com/google/uuid"
)

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(notification *model.Notification) error {
	query := `
		INSERT INTO notifications (id, user_id, report_id, title, message, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(query,
		notification.ID,
		notification.UserID,
		notification.ReportID,
		notification.Title,
		notification.Message,
		notification.IsRead,
		notification.CreatedAt,
	)
	return err
}

func (r *NotificationRepository) GetByUserID(userID uuid.UUID) ([]model.Notification, error) {
	query := `
		SELECT id, user_id, report_id, title, message, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []model.Notification
	for rows.Next() {
		var n model.Notification
		var reportID sql.NullString
		err := rows.Scan(
			&n.ID,
			&n.UserID,
			&reportID,
			&n.Title,
			&n.Message,
			&n.IsRead,
			&n.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if reportID.Valid {
			uid, _ := uuid.Parse(reportID.String)
			n.ReportID = &uid
		}
		notifications = append(notifications, n)
	}

	return notifications, nil
}

func (r *NotificationRepository) GetUnreadCount(userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`
	var count int
	err := r.db.QueryRow(query, userID).Scan(&count)
	return count, err
}

func (r *NotificationRepository) MarkAsRead(notificationID, userID uuid.UUID) error {
	query := `UPDATE notifications SET is_read = TRUE WHERE id = $1 AND user_id = $2`
	result, err := r.db.Exec(query, notificationID, userID)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *NotificationRepository) MarkAllAsRead(userID uuid.UUID) error {
	query := `UPDATE notifications SET is_read = TRUE WHERE user_id = $1 AND is_read = FALSE`
	_, err := r.db.Exec(query, userID)
	return err
}

func (r *NotificationRepository) CreateStatusNotification(reportID uuid.UUID, newStatus model.ReportStatus, reportTitle string) error {
	var reporterID sql.NullString
	query := `SELECT reporter_id FROM reports WHERE id = $1`
	err := r.db.QueryRow(query, reportID).Scan(&reporterID)
	if err != nil {
		return err
	}

	if !reporterID.Valid {
		return nil
	}

	userID, err := uuid.Parse(reporterID.String)
	if err != nil {
		return err
	}

	notification := &model.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		ReportID:  &reportID,
		Title:     "Status Laporan Diperbarui",
		Message:   "Laporan \"" + reportTitle + "\" telah diubah statusnya menjadi: " + string(newStatus),
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	return r.Create(notification)
}

func (r *NotificationRepository) IsMessageProcessed(messageID string) (bool, error) {
	query := `SELECT 1 FROM processed_messages WHERE message_id = $1`
	var exists int
	err := r.db.QueryRow(query, messageID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *NotificationRepository) MarkMessageProcessed(messageID string) error {
	query := `INSERT INTO processed_messages (message_id, processed_at) VALUES ($1, $2) ON CONFLICT (message_id) DO NOTHING`
	_, err := r.db.Exec(query, messageID, time.Now())
	return err
}
