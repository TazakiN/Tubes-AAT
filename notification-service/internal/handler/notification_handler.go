package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"notification-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	notificationService *service.NotificationService
	jwtSecret           string
}

func NewNotificationHandler(notificationService *service.NotificationService, jwtSecret string) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		jwtSecret:           jwtSecret,
	}
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
	userID := c.GetHeader("X-User-ID")

	if userID == "" {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		claims, err := h.validateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		userID = claims["user_id"].(string)
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	client := h.notificationService.RegisterClient(uid)
	defer h.notificationService.UnregisterClient(client)

	c.SSEvent("connected", gin.H{"status": "connected", "user_id": userID})
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

func (h *NotificationHandler) validateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(h.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return claims, nil
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

	if err := h.notificationService.MarkAsRead(notificationID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "marked as read"})
}

func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.notificationService.MarkAllAsRead(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all marked as read"})
}

func (h *NotificationHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "notification-service",
	})
}

func (h *NotificationHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "notification-service",
		"version": "1.0.0",
	})
}

func (h *NotificationHandler) GetDLQStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"note": "query RabbitMQ management API directly",
	})
}

func formatDuration(d int64) string {
	return fmt.Sprintf("%ds", d)
}
