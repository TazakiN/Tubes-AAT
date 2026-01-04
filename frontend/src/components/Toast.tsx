"use client";

import {
  useState,
  useEffect,
  createContext,
  useContext,
  ReactNode,
} from "react";
import type { Notification } from "@/types";

interface Toast {
  id: string;
  title: string;
  message: string;
  type: "success" | "error" | "info";
}

interface ToastContextType {
  showToast: (
    title: string,
    message: string,
    type?: "success" | "error" | "info"
  ) => void;
  showNotificationToast: (notification: Notification) => void;
}

const ToastContext = createContext<ToastContextType | undefined>(undefined);

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return context;
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const showToast = (
    title: string,
    message: string,
    type: "success" | "error" | "info" = "info"
  ) => {
    const id = Date.now().toString();
    setToasts((prev) => [...prev, { id, title, message, type }]);
  };

  const showNotificationToast = (notification: Notification) => {
    showToast(notification.title, notification.message, "info");
  };

  const removeToast = (id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  };

  return (
    <ToastContext.Provider value={{ showToast, showNotificationToast }}>
      {children}
      <div
        style={{
          position: "fixed",
          top: "1rem",
          right: "1rem",
          zIndex: 9999,
          display: "flex",
          flexDirection: "column",
          gap: "0.5rem",
        }}
      >
        {toasts.map((toast) => (
          <ToastItem
            key={toast.id}
            toast={toast}
            onClose={() => removeToast(toast.id)}
          />
        ))}
      </div>
    </ToastContext.Provider>
  );
}

function ToastItem({ toast, onClose }: { toast: Toast; onClose: () => void }) {
  useEffect(() => {
    const timer = setTimeout(onClose, 4000);
    return () => clearTimeout(timer);
  }, [onClose]);

  const bgColor = {
    success: "rgba(34, 197, 94, 0.2)",
    error: "rgba(239, 68, 68, 0.2)",
    info: "rgba(99, 102, 241, 0.2)",
  }[toast.type];

  const borderColor = {
    success: "var(--success)",
    error: "var(--error)",
    info: "var(--accent-primary)",
  }[toast.type];

  const icon = {
    success: "✓",
    error: "✕",
    info: "🔔",
  }[toast.type];

  return (
    <div
      style={{
        background: bgColor,
        border: `1px solid ${borderColor}`,
        borderRadius: "8px",
        padding: "1rem",
        minWidth: "300px",
        maxWidth: "400px",
        animation: "slideIn 0.3s ease-out",
        display: "flex",
        gap: "0.75rem",
        boxShadow: "0 4px 12px rgba(0,0,0,0.3)",
      }}
    >
      <span style={{ fontSize: "1.25rem" }}>{icon}</span>
      <div style={{ flex: 1 }}>
        <div style={{ fontWeight: "600", marginBottom: "0.25rem" }}>
          {toast.title}
        </div>
        <div style={{ fontSize: "0.875rem", color: "var(--text-secondary)" }}>
          {toast.message}
        </div>
      </div>
      <button
        onClick={onClose}
        style={{
          background: "transparent",
          border: "none",
          color: "var(--text-secondary)",
          cursor: "pointer",
          fontSize: "1.25rem",
          padding: 0,
        }}
      >
        ×
      </button>
      <style jsx>{`
        @keyframes slideIn {
          from {
            opacity: 0;
            transform: translateX(100%);
          }
          to {
            opacity: 1;
            transform: translateX(0);
          }
        }
      `}</style>
    </div>
  );
}
