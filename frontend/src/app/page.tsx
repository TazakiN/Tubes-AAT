"use client";

import Link from "next/link";
import Navbar from "@/components/Navbar";
import { useAuth } from "@/lib/auth";

export default function HomePage() {
  const { user, isAdmin } = useAuth();

  return (
    <>
      <Navbar />
      <main className="container">
        <section style={{ textAlign: "center", padding: "6rem 2rem" }}>
          <h1
            style={{
              fontSize: "3rem",
              fontWeight: "700",
              marginBottom: "1.5rem",
              background:
                "linear-gradient(135deg, var(--accent-primary), var(--accent-secondary))",
              WebkitBackgroundClip: "text",
              WebkitTextFillColor: "transparent",
            }}
          >
            CityConnect
          </h1>
          <p
            style={{
              fontSize: "1.25rem",
              color: "var(--text-secondary)",
              maxWidth: "600px",
              margin: "0 auto 2rem",
            }}
          >
            Platform pelaporan masalah lingkungan untuk warga kota. Laporkan
            masalah, pantau progress, dan bantu memperbaiki lingkungan kita
            bersama.
          </p>

          <div
            style={{ display: "flex", gap: "1rem", justifyContent: "center" }}
          >
            <Link href="/reports" className="btn btn-primary">
              Lihat Laporan Publik
            </Link>
            {user ? (
              <Link
                href={isAdmin() ? "/admin" : "/dashboard"}
                className="btn btn-secondary"
              >
                {isAdmin() ? "Dashboard Admin" : "Dashboard Saya"}
              </Link>
            ) : (
              <Link href="/register" className="btn btn-secondary">
                Daftar Sekarang
              </Link>
            )}
          </div>
        </section>

        <section
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))",
            gap: "1.5rem",
            padding: "4rem 0",
          }}
        >
          <div className="card">
            <h3 className="card-title">📝 Buat Laporan</h3>
            <p className="card-description">
              Laporkan masalah di lingkungan Anda seperti jalan rusak, sampah
              menumpuk, atau masalah kesehatan.
            </p>
          </div>

          <div className="card">
            <h3 className="card-title">👀 Pantau Progress</h3>
            <p className="card-description">
              Lacak status penyelesaian laporan Anda secara real-time dengan
              notifikasi otomatis.
            </p>
          </div>

          <div className="card">
            <h3 className="card-title">👍 Vote Laporan</h3>
            <p className="card-description">
              Dukung laporan warga lain dengan upvote untuk memprioritaskan
              penanganan masalah.
            </p>
          </div>
        </section>
      </main>
    </>
  );
}
