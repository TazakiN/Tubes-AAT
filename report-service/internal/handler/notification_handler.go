package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"report-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HTTP handler for notification-related endpoints.
type NotificationHandler struct {
	notificationService *service.NotificationService
}

// Constructor for NotificationHandler.
func NewNotificationHandler(notificationService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

// Handles GET /notifications - returns all notifications for the authenticated user.
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	response, err := h.notificationService.GetUserNotifications(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Handles GET /notifications/stream - establishes an SSE connection for real-time notifications.
func (h *NotificationHandler) StreamNotifications(c *gin.Context) {
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Register SSE client
	client := h.notificationService.RegisterClient(userID)
	defer h.notificationService.UnregisterClient(client)

	// Send initial connection event
	c.SSEvent("connected", gin.H{"message": "SSE connection established"})
	c.Writer.Flush()

	// Create a channel to detect client disconnect
	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			// Client disconnected
			return
		case notification, ok := <-client.Channel:
			if !ok {
				// Channel closed
				return
			}
			// Send notification as SSE event
			data, _ := json.Marshal(notification)
			c.SSEvent("notification", string(data))
			c.Writer.Flush()
		}
	}
}

// Handles PATCH /notifications/:id/read - marks a single notification as read.
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification ID required"})
		return
	}

	err := h.notificationService.MarkAsRead(notificationID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification marked as read"})
}

// Handles PATCH /notifications/read-all - marks all notifications for the user as read.
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.notificationService.MarkAllAsRead(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all notifications marked as read"})
}

// Helper for writing SSE event messages to an io.Writer.
func SSEWrite(w io.Writer, event string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(jsonData))
	return err
}
