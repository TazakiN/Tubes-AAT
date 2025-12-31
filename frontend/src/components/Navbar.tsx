"use client";

import Link from "next/link";
import { useAuth } from "@/lib/auth";
import NotificationBell from "./NotificationBell";
import { useToast } from "./Toast";

export default function Navbar() {
  const { user, logout, isAdmin } = useAuth();
  const { showNotificationToast } = useToast();

  return (
    <nav className="navbar">
      <div className="container navbar-content">
        <Link href="/" className="navbar-brand">
          CityConnect
        </Link>

        <div className="navbar-links">
          <Link href="/reports" className="navbar-link">
            Laporan Publik
          </Link>

          {user && (
            <>
              {isAdmin() ? (
                <Link href="/admin" className="navbar-link">
                  Dashboard Admin
                </Link>
              ) : (
                <Link href="/dashboard" className="navbar-link">
                  Dashboard Saya
                </Link>
              )}
            </>
          )}
        </div>

        <div className="navbar-user">
          {user ? (
            <>
              <NotificationBell onNewNotification={showNotificationToast} />
              <span
                style={{ color: "var(--text-secondary)", fontSize: "0.875rem" }}
              >
                {user.name}
              </span>
              <button onClick={logout} className="btn btn-secondary btn-sm">
                Logout
              </button>
            </>
          ) : (
            <>
              <Link href="/login" className="btn btn-secondary btn-sm">
                Login
              </Link>
              <Link href="/register" className="btn btn-primary btn-sm">
                Register
              </Link>
            </>
          )}
        </div>
      </div>
    </nav>
  );
}
