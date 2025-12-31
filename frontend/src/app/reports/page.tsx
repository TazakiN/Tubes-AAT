"use client";

import { useState, useEffect, useCallback } from "react";
import Navbar from "@/components/Navbar";
import ReportCard from "@/components/ReportCard";
import SearchFilter from "@/components/SearchFilter";
import { api } from "@/lib/api";
import type { Report, Category } from "@/types";

export default function ReportsPage() {
  const [reports, setReports] = useState<Report[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setIsLoading(true);
    try {
      const [reportsRes, categoriesRes] = await Promise.all([
        api.getPublicReports(),
        api.getCategories(),
      ]);
      setReports(reportsRes.reports || []);
      setCategories(categoriesRes.categories || []);
    } catch (error) {
      console.error("Failed to load data:", error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSearch = useCallback(
    async (search: string, categoryId: number | null) => {
      setIsLoading(true);
      try {
        const response = await api.getPublicReports(search, categoryId);
        setReports(response.reports || []);
      } catch (error) {
        console.error("Search failed:", error);
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  return (
    <>
      <Navbar />
      <main className="container">
        <div className="page-header">
          <h1 className="page-title">Laporan Publik</h1>
          <p className="page-subtitle">
            Laporan dari warga yang dibagikan secara publik. Vote untuk
            mendukung.
          </p>
        </div>

        <div className="page-content">
          <SearchFilter
            categories={categories}
            onSearch={handleSearch}
            placeholder="Cari laporan publik..."
          />

          {isLoading ? (
            <div className="loading">
              <div className="spinner" />
            </div>
          ) : reports.length === 0 ? (
            <div className="empty-state">
              <div className="empty-state-icon">📋</div>
              <h3>Tidak ada laporan ditemukan</h3>
              <p>Coba kata kunci atau kategori lain</p>
            </div>
          ) : (
            <div className="reports-grid">
              {reports.map((report) => (
                <ReportCard key={report.id} report={report} showVoting={true} />
              ))}
            </div>
          )}
        </div>
      </main>
    </>
  );
}
