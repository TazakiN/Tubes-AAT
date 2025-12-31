package service

import (
	"log"
	"sync"

	"report-service/internal/model"
	"report-service/internal/repository"

	"github.com/google/uuid"
)

// Represents a connected SSE client for real-time notifications.
type SSEClient struct {
	UserID  uuid.UUID
	Channel chan *model.Notification
}

// Business logic layer for notification operations and SSE client management.
type NotificationService struct {
	notificationRepo *repository.NotificationRepository
	mu               sync.RWMutex
	clients          map[uuid.UUID][]*SSEClient // Map of userID to connected clients
}

// Constructor for NotificationService.
func NewNotificationService(notificationRepo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		clients:          make(map[uuid.UUID][]*SSEClient),
	}
}

// Returns all notifications for a user, sorted by creation time descending.
func (s *NotificationService) GetUserNotifications(userIDStr string) (*model.NotificationListResponse, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, err
	}

	notifications, err := s.notificationRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	if notifications == nil {
		notifications = []model.Notification{}
	}

	unreadCount, err := s.notificationRepo.GetUnreadCount(userID)
	if err != nil {
		return nil, err
	}

	return &model.NotificationListResponse{
		Notifications: notifications,
		UnreadCount:   unreadCount,
	}, nil
}

// Sets a single notification's read status to true.
func (s *NotificationService) MarkAsRead(notificationIDStr, userIDStr string) error {
	notificationID, err := uuid.Parse(notificationIDStr)
	if err != nil {
		return err
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return err
	}

	return s.notificationRepo.MarkAsRead(notificationID, userID)
}

// Sets all user notifications' read status to true.
func (s *NotificationService) MarkAllAsRead(userIDStr string) error {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return err
	}

	return s.notificationRepo.MarkAllAsRead(userID)
}

// Persists a status change notification and pushes to connected SSE clients.
func (s *NotificationService) CreateStatusNotification(reportID uuid.UUID, newStatus model.ReportStatus, reportTitle string, reporterID *uuid.UUID) error {
	// Create notification in database
	err := s.notificationRepo.CreateStatusNotification(reportID, newStatus, reportTitle)
	if err != nil {
		log.Printf("Failed to create notification: %v", err)
		return err
	}

	// If we have the reporter ID, push to connected SSE clients
	if reporterID != nil {
		notification := &model.Notification{
			ID:       uuid.New(),
			UserID:   *reporterID,
			ReportID: &reportID,
			Title:    "Status Laporan Diperbarui",
			Message:  "Laporan \"" + reportTitle + "\" telah diubah statusnya menjadi: " + string(newStatus),
			IsRead:   false,
		}
		s.pushToClients(*reporterID, notification)
	}

	return nil
}

// Creates and registers a new SSE client channel for real-time updates.
func (s *NotificationService) RegisterClient(userID uuid.UUID) *SSEClient {
	s.mu.Lock()
	defer s.mu.Unlock()

	client := &SSEClient{
		UserID:  userID,
		Channel: make(chan *model.Notification, 10), // Buffered channel
	}

	s.clients[userID] = append(s.clients[userID], client)
	log.Printf("SSE client registered for user %s (total: %d)", userID, len(s.clients[userID]))

	return client
}

// Removes an SSE client and closes its channel.
func (s *NotificationService) UnregisterClient(client *SSEClient) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userClients := s.clients[client.UserID]
	for i, c := range userClients {
		if c == client {
			// Remove client from slice
			s.clients[client.UserID] = append(userClients[:i], userClients[i+1:]...)
			break
		}
	}

	close(client.Channel)
	log.Printf("SSE client unregistered for user %s", client.UserID)
}

// Broadcasts a notification to all connected SSE clients for a user.
func (s *NotificationService) pushToClients(userID uuid.UUID, notification *model.Notification) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := s.clients[userID]
	for _, client := range clients {
		select {
		case client.Channel <- notification:
			log.Printf("Notification pushed to client for user %s", userID)
		default:
			log.Printf("Client channel full for user %s, skipping", userID)
		}
	}
}
