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
	ReporterHash *string      `json:"-"`                       // Never expose, used internally
	Status       ReportStatus `json:"status"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Request/Response DTOs
type CreateReportRequest struct {
	Title        string       `json:"title" binding:"required"`
	Description  string       `json:"description" binding:"required"`
	CategoryID   int          `json:"category_id" binding:"required"`
	LocationLat  *float64     `json:"location_lat"`
	LocationLng  *float64     `json:"location_lng"`
	PhotoURL     *string      `json:"photo_url"`
	PrivacyLevel PrivacyLevel `json:"privacy_level" binding:"required"`
}

type UpdateStatusRequest struct {
	Status ReportStatus `json:"status" binding:"required"`
}

type ReportListResponse struct {
	Reports []Report `json:"reports"`
	Total   int      `json:"total"`
}
