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

type NotificationHandler struct {
	notificationService *service.NotificationService
}

func NewNotificationHandler(notificationService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

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

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	client := h.notificationService.RegisterClient(userID)
	defer h.notificationService.UnregisterClient(client)

	c.SSEvent("connected", gin.H{"message": "SSE connection established"})
	c.Writer.Flush()

	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			return
		case notification, ok := <-client.Channel:
			if !ok {
				return
			}
			data, _ := json.Marshal(notification)
			c.SSEvent("notification", string(data))
			c.Writer.Flush()
		}
	}
}

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

func SSEWrite(w io.Writer, event string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(jsonData))
	return err
}
