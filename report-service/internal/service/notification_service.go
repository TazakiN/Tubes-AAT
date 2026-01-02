package service

import (
	"report-service/internal/messaging"
	"report-service/internal/model"
	"report-service/internal/repository"

	"github.com/google/uuid"
)

type NotificationService struct {
	notificationRepo *repository.NotificationRepository
	sseHub           *messaging.SSEHub
}

func NewNotificationService(notificationRepo *repository.NotificationRepository, sseHub *messaging.SSEHub) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		sseHub:           sseHub,
	}
}

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

func (s *NotificationService) MarkAllAsRead(userIDStr string) error {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return err
	}

	return s.notificationRepo.MarkAllAsRead(userID)
}

func (s *NotificationService) RegisterClient(userID uuid.UUID) *messaging.SSEClient {
	return s.sseHub.RegisterClient(userID)
}

func (s *NotificationService) UnregisterClient(client *messaging.SSEClient) {
	s.sseHub.UnregisterClient(client)
}
