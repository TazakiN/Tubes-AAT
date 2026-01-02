package model

import (
	"time"

	"github.com/google/uuid"
)

type PrivacyLevel string

const (
	PrivacyPublic    PrivacyLevel = "public"
	PrivacyPrivate   PrivacyLevel = "private"
	PrivacyAnonymous PrivacyLevel = "anonymous"
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

type Category struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Department string `json:"department"`
}

type Report struct {
	ID           uuid.UUID    `json:"id"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	CategoryID   int          `json:"category_id"`
	Category     *Category    `json:"category,omitempty"`
	LocationLat  *float64     `json:"location_lat,omitempty"`
	LocationLng  *float64     `json:"location_lng,omitempty"`
	PhotoURL     *string      `json:"photo_url,omitempty"`
	PrivacyLevel PrivacyLevel `json:"privacy_level"`
	ReporterID   *uuid.UUID   `json:"reporter_id,omitempty"`   // Hidden for anonymous
	ReporterName *string      `json:"reporter_name,omitempty"` // Hidden for anonymous
	ReporterHash *string      `json:"-"` 
	Status       ReportStatus `json:"status"`
	VoteScore    int          `json:"vote_score"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

type ReportVote struct {
	ID        uuid.UUID `json:"id"`
	ReportID  uuid.UUID `json:"report_id"`
	UserID    uuid.UUID `json:"user_id"`
	VoteType  VoteType  `json:"vote_type"`
	CreatedAt time.Time `json:"created_at"`
}

type Notification struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	ReportID  *uuid.UUID `json:"report_id,omitempty"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	IsRead    bool       `json:"is_read"`
	CreatedAt time.Time  `json:"created_at"`
}

// Request/Response DTOs
type CreateReportRequest struct {
	Title                 string       `json:"title" binding:"required"`
	Description           string       `json:"description" binding:"required"`
	CategoryID            int          `json:"category_id"`                       // Use existing category
	NewCategoryName       *string      `json:"new_category_name,omitempty"`       // Create new category
	NewCategoryDepartment *string      `json:"new_category_department,omitempty"` // Required if new category
	LocationLat           *float64     `json:"location_lat"`
	LocationLng           *float64     `json:"location_lng"`
	PhotoURL              *string      `json:"photo_url"`
	PrivacyLevel          PrivacyLevel `json:"privacy_level" binding:"required"`
}

type CreateCategoryRequest struct {
	Name       string `json:"name" binding:"required"`
	Department string `json:"department" binding:"required"`
}

type UpdateReportRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

type UpdateStatusRequest struct {
	Status ReportStatus `json:"status" binding:"required"`
}

type VoteRequest struct {
	VoteType VoteType `json:"vote_type" binding:"required"`
}

type VoteResponse struct {
	VoteScore    int       `json:"vote_score"`
	UserVoteType *VoteType `json:"user_vote_type,omitempty"`
}

type ReportListResponse struct {
	Reports []Report `json:"reports"`
	Total   int      `json:"total"`
}

type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unread_count"`
}
