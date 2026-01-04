"use client";

import { useEffect, useRef, useCallback } from "react";

interface ServiceWorkerHook {
  isSupported: boolean;
  isRegistered: boolean;
  sendNotification: (title: string, body: string, tag?: string) => void;
  requestPermission: () => Promise<NotificationPermission>;
}

export function useServiceWorker(): ServiceWorkerHook {
  const swRegistration = useRef<ServiceWorkerRegistration | null>(null);
  const isRegistered = useRef(false);

  useEffect(() => {
    if (typeof window === "undefined" || !("serviceWorker" in navigator)) {
      return;
    }

    navigator.serviceWorker
      .register("/sw.js")
      .then((registration) => {
        swRegistration.current = registration;
        isRegistered.current = true;
        console.log("Service Worker registered");
      })
      .catch((error) => {
        console.error("Service Worker registration failed:", error);
      });
  }, []);

  const requestPermission =
    useCallback(async (): Promise<NotificationPermission> => {
      if (!("Notification" in window)) {
        return "denied";
      }

      if (Notification.permission === "granted") {
        return "granted";
      }

      if (Notification.permission !== "denied") {
        const permission = await Notification.requestPermission();
        return permission;
      }

      return Notification.permission;
    }, []);

  const sendNotification = useCallback(
    (title: string, body: string, tag?: string) => {
      if (!swRegistration.current?.active) {
        console.warn("Service Worker not active");
        return;
      }

      if (Notification.permission !== "granted") {
        console.warn("Notification permission not granted");
        return;
      }

      if (document.visibilityState === "hidden") {
        swRegistration.current.active.postMessage({
          type: "SHOW_NOTIFICATION",
          payload: {
            title,
            body,
            tag: tag || "cityconnect-notification",
            data: { url: "/" },
          },
        });
      }
    },
    []
  );

  return {
    isSupported: typeof window !== "undefined" && "serviceWorker" in navigator,
    isRegistered: isRegistered.current,
    sendNotification,
    requestPermission,
  };
}
