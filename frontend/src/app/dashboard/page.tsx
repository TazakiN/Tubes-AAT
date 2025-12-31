"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import Navbar from "@/components/Navbar";
import ReportCard from "@/components/ReportCard";
import SearchFilter from "@/components/SearchFilter";
import { useAuth } from "@/lib/auth";
import { api } from "@/lib/api";
import type { Report, Category, PrivacyLevel } from "@/types";

export default function DashboardPage() {
  const { user, isLoading: authLoading } = useAuth();
  const router = useRouter();

  const [reports, setReports] = useState<Report[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);

  // Form state
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId] = useState<number>(0);
  const [newCategoryName, setNewCategoryName] = useState("");
  const [newCategoryDept, setNewCategoryDept] = useState("");
  const [useNewCategory, setUseNewCategory] = useState(false);
  const [privacyLevel, setPrivacyLevel] = useState<PrivacyLevel>("public");
  const [formError, setFormError] = useState("");
  const [formSuccess, setFormSuccess] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (!authLoading && !user) {
      router.push("/login");
    }
  }, [authLoading, user, router]);

  useEffect(() => {
    if (user) {
      loadData();
    }
  }, [user]);

  const loadData = async () => {
    setIsLoading(true);
    try {
      const [reportsRes, categoriesRes] = await Promise.all([
        api.getMyReports(),
        api.getCategories(),
      ]);
      setReports(reportsRes.reports || []);
      setCategories(categoriesRes.categories || []);
      if (categoriesRes.categories?.length) {
        setCategoryId(categoriesRes.categories[0].id);
      }
    } catch (error) {
      console.error("Failed to load data:", error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSearch = useCallback(
    async (search: string, catId: number | null) => {
      setIsLoading(true);
      try {
        const response = await api.getMyReports(search, catId);
        setReports(response.reports || []);
      } catch (error) {
        console.error("Search failed:", error);
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  const handleCreateReport = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError("");
    setFormSuccess("");
    setIsSubmitting(true);

    try {
      const payload: any = {
        title,
        description,
        privacy_level: privacyLevel,
      };

      if (useNewCategory && newCategoryName && newCategoryDept) {
        payload.new_category_name = newCategoryName;
        payload.new_category_department = newCategoryDept;
      } else {
        payload.category_id = categoryId;
      }

      await api.createReport(payload);
      setFormSuccess("Laporan berhasil dibuat!");
      setTitle("");
      setDescription("");
      setPrivacyLevel("public");
      setNewCategoryName("");
      setNewCategoryDept("");
      setUseNewCategory(false);
      setTimeout(() => {
        setShowCreateModal(false);
        setFormSuccess("");
        loadData();
      }, 1500);
    } catch (error) {
      setFormError(
        error instanceof Error ? error.message : "Gagal membuat laporan"
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  if (authLoading || !user) {
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
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
            }}
          >
            <div>
              <h1 className="page-title">Dashboard Saya</h1>
              <p className="page-subtitle">Kelola laporan yang Anda buat</p>
            </div>
            <button
              className="btn btn-primary"
              onClick={() => setShowCreateModal(true)}
            >
              + Buat Laporan Baru
            </button>
          </div>
        </div>

        <div className="page-content">
          <SearchFilter
            categories={categories}
            onSearch={handleSearch}
            placeholder="Cari laporan saya..."
          />

          {isLoading ? (
            <div className="loading">
              <div className="spinner" />
            </div>
          ) : reports.length === 0 ? (
            <div className="empty-state">
              <div className="empty-state-icon">📝</div>
              <h3>Tidak ada laporan ditemukan</h3>
              <p>Buat laporan pertama Anda atau coba kata kunci lain</p>
            </div>
          ) : (
            <div className="reports-grid">
              {reports.map((report) => (
                <ReportCard
                  key={report.id}
                  report={report}
                  showVoting={false}
                />
              ))}
            </div>
          )}
        </div>
      </main>

      {showCreateModal && (
        <div
          className="modal-overlay"
          onClick={() => setShowCreateModal(false)}
        >
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2 className="modal-title">Buat Laporan Baru</h2>

            {formError && (
              <div className="message message-error">{formError}</div>
            )}
            {formSuccess && (
              <div className="message message-success">{formSuccess}</div>
            )}

            <form onSubmit={handleCreateReport}>
              <div className="form-group">
                <label className="form-label">Judul Laporan</label>
                <input
                  type="text"
                  className="form-input"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="Ringkasan masalah"
                  required
                />
              </div>

              <div className="form-group">
                <label className="form-label">Deskripsi</label>
                <textarea
                  className="form-textarea"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Jelaskan masalah secara detail..."
                  required
                />
              </div>

              <div className="form-group">
                <label
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "0.5rem",
                    cursor: "pointer",
                  }}
                >
                  <input
                    type="checkbox"
                    checked={useNewCategory}
                    onChange={(e) => setUseNewCategory(e.target.checked)}
                  />
                  Buat kategori baru
                </label>
              </div>

              {useNewCategory ? (
                <>
                  <div className="form-group">
                    <label className="form-label">Nama Kategori Baru</label>
                    <input
                      type="text"
                      className="form-input"
                      value={newCategoryName}
                      onChange={(e) => setNewCategoryName(e.target.value)}
                      placeholder="Contoh: Lampu Taman Rusak"
                      required={useNewCategory}
                    />
                  </div>
                  <div className="form-group">
                    <label className="form-label">Departemen</label>
                    <select
                      className="form-select"
                      value={newCategoryDept}
                      onChange={(e) => setNewCategoryDept(e.target.value)}
                      required={useNewCategory}
                    >
                      <option value="">Pilih Departemen</option>
                      <option value="kebersihan">Kebersihan</option>
                      <option value="kesehatan">Kesehatan</option>
                      <option value="infrastruktur">Infrastruktur</option>
                    </select>
                  </div>
                </>
              ) : (
                <div className="form-group">
                  <label className="form-label">Kategori</label>
                  <select
                    className="form-select"
                    value={categoryId}
                    onChange={(e) => setCategoryId(Number(e.target.value))}
                    required={!useNewCategory}
                  >
                    {categories.map((cat) => (
                      <option key={cat.id} value={cat.id}>
                        {cat.name} ({cat.department})
                      </option>
                    ))}
                  </select>
                </div>
              )}

              <div className="form-group">
                <label className="form-label">Tingkat Privasi</label>
                <select
                  className="form-select"
                  value={privacyLevel}
                  onChange={(e) =>
                    setPrivacyLevel(e.target.value as PrivacyLevel)
                  }
                >
                  <option value="public">
                    Publik - Semua orang bisa melihat
                  </option>
                  <option value="private">
                    Privat - Hanya Anda dan petugas
                  </option>
                  <option value="anonymous">
                    Anonim - Identitas disembunyikan
                  </option>
                </select>
              </div>

              <div className="modal-actions">
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => setShowCreateModal(false)}
                >
                  Batal
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={isSubmitting}
                >
                  {isSubmitting ? "Mengirim..." : "Kirim Laporan"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  );
}
