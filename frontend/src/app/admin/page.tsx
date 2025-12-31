"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import Navbar from "@/components/Navbar";
import { useAuth } from "@/lib/auth";
import { api } from "@/lib/api";
import type { Report, ReportStatus } from "@/types";

const STATUS_OPTIONS: { value: ReportStatus; label: string }[] = [
  { value: "pending", label: "Pending" },
  { value: "accepted", label: "Diterima" },
  { value: "in_progress", label: "Dalam Proses" },
  { value: "completed", label: "Selesai" },
  { value: "rejected", label: "Ditolak" },
];

export default function AdminPage() {
  const { user, isLoading: authLoading, isAdmin } = useAuth();
  const router = useRouter();

  const [reports, setReports] = useState<Report[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [selectedReport, setSelectedReport] = useState<Report | null>(null);
  const [newStatus, setNewStatus] = useState<ReportStatus>("pending");
  const [isUpdating, setIsUpdating] = useState(false);
  const [message, setMessage] = useState({ type: "", text: "" });

  useEffect(() => {
    if (!authLoading) {
      if (!user) {
        router.push("/login");
      } else if (!isAdmin()) {
        router.push("/dashboard");
      }
    }
  }, [authLoading, user, isAdmin, router]);

  useEffect(() => {
    if (user && isAdmin()) {
      loadReports();
    }
  }, [user]);

  const loadReports = async () => {
    setIsLoading(true);
    try {
      const response = await api.getReports();
      setReports(response.reports || []);
    } catch (error) {
      console.error("Failed to load reports:", error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleStatusUpdate = async () => {
    if (!selectedReport) return;

    setIsUpdating(true);
    setMessage({ type: "", text: "" });

    try {
      await api.updateReportStatus(selectedReport.id, newStatus);
      setMessage({ type: "success", text: "Status berhasil diperbarui" });
      setSelectedReport(null);
      loadReports();
    } catch (error) {
      setMessage({
        type: "error",
        text:
          error instanceof Error ? error.message : "Gagal memperbarui status",
      });
    } finally {
      setIsUpdating(false);
    }
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString("id-ID", {
      day: "numeric",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const getDepartmentLabel = () => {
    switch (user?.department) {
      case "kebersihan":
        return "Dinas Kebersihan";
      case "kesehatan":
        return "Dinas Kesehatan";
      case "infrastruktur":
        return "Dinas Infrastruktur";
      default:
        return "Unknown";
    }
  };

  if (authLoading || !user || !isAdmin()) {
    return (
      <>
        <Navbar />
        <div className="loading">
          <div className="spinner" />
        </div>
      </>
    );
  }

  return (
    <>
      <Navbar />
      <main className="container">
        <div className="page-header">
          <h1 className="page-title">Dashboard Admin</h1>
          <p className="page-subtitle">{getDepartmentLabel()}</p>
        </div>

        {message.text && (
          <div className={`message message-${message.type}`}>
            {message.text}
          </div>
        )}

        <div className="page-content">
          {isLoading ? (
            <div className="loading">
              <div className="spinner" />
            </div>
          ) : reports.length === 0 ? (
            <div className="empty-state">
              <div className="empty-state-icon">📋</div>
              <h3>Tidak ada laporan</h3>
              <p>Belum ada laporan untuk departemen Anda</p>
            </div>
          ) : (
            <div style={{ overflowX: "auto" }}>
              <table
                style={{
                  width: "100%",
                  borderCollapse: "collapse",
                  background: "var(--bg-secondary)",
                  borderRadius: "8px",
                  overflow: "hidden",
                }}
              >
                <thead>
                  <tr style={{ borderBottom: "1px solid var(--border)" }}>
                    <th style={{ padding: "1rem", textAlign: "left" }}>
                      Judul
                    </th>
                    <th style={{ padding: "1rem", textAlign: "left" }}>
                      Kategori
                    </th>
                    <th style={{ padding: "1rem", textAlign: "left" }}>
                      Status
                    </th>
                    <th style={{ padding: "1rem", textAlign: "left" }}>
                      Tanggal
                    </th>
                    <th style={{ padding: "1rem", textAlign: "left" }}>Aksi</th>
                  </tr>
                </thead>
                <tbody>
                  {reports.map((report) => (
                    <tr
                      key={report.id}
                      style={{ borderBottom: "1px solid var(--border)" }}
                    >
                      <td style={{ padding: "1rem" }}>
                        <div style={{ fontWeight: "500" }}>{report.title}</div>
                        <div
                          style={{
                            fontSize: "0.75rem",
                            color: "var(--text-secondary)",
                            marginTop: "0.25rem",
                          }}
                        >
                          {report.description.substring(0, 60)}...
                        </div>
                      </td>
                      <td style={{ padding: "1rem" }}>
                        <span className="category-badge">
                          {report.category?.name || "N/A"}
                        </span>
                      </td>
                      <td style={{ padding: "1rem" }}>
                        <span
                          className={`status-badge status-${report.status}`}
                        >
                          {report.status.replace("_", " ")}
                        </span>
                      </td>
                      <td
                        style={{
                          padding: "1rem",
                          fontSize: "0.875rem",
                          color: "var(--text-secondary)",
                        }}
                      >
                        {formatDate(report.created_at)}
                      </td>
                      <td style={{ padding: "1rem" }}>
                        <button
                          className="btn btn-secondary btn-sm"
                          onClick={() => {
                            setSelectedReport(report);
                            setNewStatus(report.status);
                          }}
                        >
                          Ubah Status
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </main>

      {selectedReport && (
        <div className="modal-overlay" onClick={() => setSelectedReport(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2 className="modal-title">Ubah Status Laporan</h2>

            <div style={{ marginBottom: "1.5rem" }}>
              <strong>{selectedReport.title}</strong>
              <p
                style={{ color: "var(--text-secondary)", marginTop: "0.5rem" }}
              >
                {selectedReport.description}
              </p>
            </div>

            <div className="form-group">
              <label className="form-label">Status Baru</label>
              <select
                className="form-select"
                value={newStatus}
                onChange={(e) => setNewStatus(e.target.value as ReportStatus)}
              >
                {STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            <div className="modal-actions">
              <button
                className="btn btn-secondary"
                onClick={() => setSelectedReport(null)}
              >
                Batal
              </button>
              <button
                className="btn btn-primary"
                onClick={handleStatusUpdate}
                disabled={isUpdating}
              >
                {isUpdating ? "Menyimpan..." : "Simpan"}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
