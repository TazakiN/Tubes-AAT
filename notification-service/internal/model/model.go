package model

import (
	"time"

	"github.com/google/uuid"
)

type ReportStatus string

const (
	StatusPending    ReportStatus = "pending"
	StatusAccepted   ReportStatus = "accepted"
	StatusInProgress ReportStatus = "in_progress"
	StatusCompleted  ReportStatus = "completed"
	StatusRejected   ReportStatus = "rejected"
)

type VoteType string

const (
	VoteUpvote   VoteType = "upvote"
	VoteDownvote VoteType = "downvote"
)

type Notification struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	ReportID  *uuid.UUID `json:"report_id,omitempty"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	IsRead    bool       `json:"is_read"`
	CreatedAt time.Time  `json:"created_at"`
}

type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unread_count"`
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

type ProcessedMessage struct {
	MessageID   string    `json:"message_id"`
	ProcessedAt time.Time `json:"processed_at"`
}
