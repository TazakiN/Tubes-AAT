"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuth } from "@/lib/auth";
import Navbar from "@/components/Navbar";

export default function RegisterPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [role, setRole] = useState("warga");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const { register } = useAuth();
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setSuccess("");
    setIsLoading(true);

    try {
      await register(email, password, name, role);
      setSuccess("Registrasi berhasil! Silakan login.");
      setTimeout(() => router.push("/login"), 2000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registrasi gagal");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <Navbar />
      <main className="container">
        <div className="form-card card">
          <h1 style={{ marginBottom: "0.5rem" }}>Daftar</h1>
          <p style={{ color: "var(--text-secondary)", marginBottom: "2rem" }}>
            Buat akun CityConnect baru
          </p>

          {error && <div className="message message-error">{error}</div>}
          {success && <div className="message message-success">{success}</div>}

          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label className="form-label">Nama Lengkap</label>
              <input
                type="text"
                className="form-input"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Nama Anda"
                required
              />
            </div>

            <div className="form-group">
              <label className="form-label">Email</label>
              <input
                type="email"
                className="form-input"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="email@example.com"
                required
              />
            </div>

            <div className="form-group">
              <label className="form-label">Password</label>
              <input
                type="password"
                className="form-input"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Minimal 6 karakter"
                minLength={6}
                required
              />
            </div>

            <div className="form-group">
              <label className="form-label">Tipe Akun</label>
              <select
                className="form-select"
                value={role}
                onChange={(e) => setRole(e.target.value)}
              >
                <option value="warga">Warga</option>
                <option value="admin_kebersihan">Admin Dinas Kebersihan</option>
                <option value="admin_kesehatan">Admin Dinas Kesehatan</option>
                <option value="admin_infrastruktur">
                  Admin Dinas Infrastruktur
                </option>
              </select>
            </div>

            <button
              type="submit"
              className="btn btn-primary"
              style={{ width: "100%" }}
              disabled={isLoading}
            >
              {isLoading ? "Loading..." : "Daftar"}
            </button>
          </form>

          <p
            style={{
              marginTop: "1.5rem",
              textAlign: "center",
              color: "var(--text-secondary)",
            }}
          >
            Sudah punya akun? <Link href="/login">Login</Link>
          </p>
        </div>
      </main>
    </>
  );
}
