"use client";

import { useState, useEffect } from "react";
import { Bell } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { api } from "@/lib/api";
import { useServiceWorker } from "@/hooks/useServiceWorker";
import type { Notification } from "@/types";

interface NotificationBellProps {
  onNewNotification?: (notification: Notification) => void;
}

export default function NotificationBell({
  onNewNotification,
}: NotificationBellProps) {
  const { user, token } = useAuth();
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const { sendNotification, requestPermission } = useServiceWorker();

  useEffect(() => {
    if (user) {
      requestPermission();

      loadNotifications();
      // Setup SSE for real-time notifications
      const eventSource = new EventSource(
        api.getNotificationStreamUrl(token || "")
      );

      eventSource.onmessage = (event) => {
        try {
          const notification = JSON.parse(event.data) as Notification;
          setNotifications((prev) => [notification, ...prev]);
          setUnreadCount((prev) => prev + 1);

          sendNotification(
            notification.title,
            notification.message,
            notification.id
          );

          if (onNewNotification) {
            onNewNotification(notification);
          }
        } catch (e) {
          console.error("Failed to parse notification:", e);
        }
      };

      eventSource.onerror = () => {
        console.error("SSE connection error");
        eventSource.close();
      };

      return () => eventSource.close();
    }
  }, [user, token, sendNotification, requestPermission]);

  const loadNotifications = async () => {
    setIsLoading(true);
    try {
      const response = await api.getNotifications();
      setNotifications(response.notifications || []);
      setUnreadCount(response.unread_count || 0);
    } catch (error) {
      console.error("Failed to load notifications:", error);
    } finally {
      setIsLoading(false);
    }
  };

  const markAsRead = async (id: string) => {
    try {
      await api.markNotificationRead(id);
      setNotifications((prev) =>
        prev.map((n) => (n.id === id ? { ...n, is_read: true } : n))
      );
      setUnreadCount((prev) => Math.max(0, prev - 1));
    } catch (error) {
      console.error("Failed to mark as read:", error);
    }
  };

  const markAllRead = async () => {
    try {
      await api.markAllNotificationsRead();
      setNotifications((prev) => prev.map((n) => ({ ...n, is_read: true })));
      setUnreadCount(0);
    } catch (error) {
      console.error("Failed to mark all as read:", error);
    }
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return "Baru saja";
    if (diffMins < 60) return `${diffMins} menit lalu`;
    if (diffMins < 1440) return `${Math.floor(diffMins / 60)} jam lalu`;
    return date.toLocaleDateString("id-ID");
  };

  if (!user) return null;

  return (
    <div className="dropdown notification-bell">
      <button
        onClick={() => setIsOpen(!isOpen)}
        style={{
          background: "transparent",
          border: "none",
          cursor: "pointer",
          fontSize: "1.25rem",
          position: "relative",
          padding: "0.5rem",
        }}
      >
        <Bell size={20} color="var(--text-primary)" />
        {unreadCount > 0 && (
          <span className="notification-count">
            {unreadCount > 9 ? "9+" : unreadCount}
          </span>
        )}
      </button>

      {isOpen && (
        <div className="dropdown-menu">
          <div
            style={{
              padding: "1rem",
              borderBottom: "1px solid var(--border)",
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
            }}
          >
            <strong>Notifikasi</strong>
            {unreadCount > 0 && (
              <button
                onClick={markAllRead}
                style={{
                  background: "transparent",
                  border: "none",
                  color: "var(--accent-primary)",
                  cursor: "pointer",
                  fontSize: "0.75rem",
                }}
              >
                Tandai semua dibaca
              </button>
            )}
          </div>

          {isLoading ? (
            <div style={{ padding: "2rem", textAlign: "center" }}>
              <div
                className="spinner"
                style={{ width: "24px", height: "24px" }}
              />
            </div>
          ) : notifications.length === 0 ? (
            <div
              style={{
                padding: "2rem",
                textAlign: "center",
                color: "var(--text-secondary)",
              }}
            >
              Tidak ada notifikasi
            </div>
          ) : (
            notifications.slice(0, 10).map((notification) => (
              <div
                key={notification.id}
                className={`dropdown-item notification-item ${
                  !notification.is_read ? "unread" : ""
                }`}
                onClick={() =>
                  !notification.is_read && markAsRead(notification.id)
                }
              >
                <div
                  style={{
                    fontWeight: notification.is_read ? "normal" : "600",
                    marginBottom: "0.25rem",
                  }}
                >
                  {notification.title}
                </div>
                <div
                  style={{
                    fontSize: "0.75rem",
                    color: "var(--text-secondary)",
                    marginBottom: "0.25rem",
                  }}
                >
                  {notification.message}
                </div>
                <div
                  style={{
                    fontSize: "0.625rem",
                    color: "var(--text-secondary)",
                  }}
                >
                  {formatDate(notification.created_at)}
                </div>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
