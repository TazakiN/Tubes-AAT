# CityConnect - Sistem Pelaporan Warga Terdistribusi

Proof-of-Concept (PoC) untuk sistem pelaporan warga dengan arsitektur microservices. Sistem ini memungkinkan warga kota (target: 2.5 juta penduduk) untuk melaporkan permasalahan lingkungan kepada pihak berwenang.

## Arsitektur Sistem

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                   FRONTEND                                       │
│                              Next.js (React)                                     │
│   ┌───────────────────┐  ┌───────────────────┐  ┌───────────────────┐           │
│   │  Citizen Dashboard │  │  Public Reports   │  │   Govt Dashboard  │           │
│   │  - My Reports      │  │  - Browse Reports │  │  - View Reports   │           │
│   │  - Create Report   │  │  - Vote Reports   │  │  - Change Status  │           │
│   │  - Edit Report     │  │  - View Details   │  │  - Dept Filtered  │           │
│   │  - Track Status    │  │                   │  │                   │           │
│   └───────────────────┘  └───────────────────┘  └───────────────────┘           │
└─────────────────────────────────────┬───────────────────────────────────────────┘
                                      │ HTTPS
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              API GATEWAY (Nginx)                                 │
│                              Port: 8080                                          │
│   - Reverse proxy to backend services                                            │
│   - Rate limiting (future)                                                       │
│   - SSL termination (production)                                                 │
└──────────┬──────────────────────┬──────────────────────┬────────────────────────┘
           │                      │                      │
           ▼                      ▼                      ▼
┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐
│   AUTH SERVICE   │   │  REPORT SERVICE  │   │ NOTIFICATION SVC │
│      (Go)        │   │      (Go)        │   │      (Go)        │
│   Port: 3001     │   │   Port: 3002     │   │   Port: 3003     │
├──────────────────┤   ├──────────────────┤   ├──────────────────┤
│ - Register       │   │ - CRUD Reports   │   │ - SSE Endpoint   │
│ - Login (JWT)    │   │ - Vote System    │   │ - Notify on      │
│ - Token Verify   │   │ - Status Update  │   │   status change  │
│ - RBAC           │   │ - Dept Filtering │   │ - Store in DB    │
└────────┬─────────┘   └────────┬─────────┘   └────────┬─────────┘
         │                      │                      │
         └──────────────────────┴──────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              PostgreSQL Database                                 │
│                              Port: 5432                                          │
├─────────────────────────────────────────────────────────────────────────────────┤
│ Tables:                                                                          │
│   - users           (id, email, password_hash, name, role, department)          │
│   - categories      (id, name, department)                                       │
│   - reports         (id, title, description, category_id, location, privacy,    │
│                      reporter_id, reporter_hash, status, vote_score, ...)       │
│   - report_votes    (id, report_id, user_id, vote_type, created_at)             │
│   - notifications   (id, user_id, report_id, message, is_read, created_at)      │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│                           OBSERVABILITY STACK                                    │
├─────────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                          │
│  │   Grafana   │◄───│    Loki     │◄───│  Promtail   │                          │
│  │  Port: 3000 │    │  Port: 3100 │    │  (sidecar)  │                          │
│  │             │    │             │    │             │                          │
│  │ Dashboards  │    │ Log Storage │    │ Log Shipper │                          │
│  └─────────────┘    └─────────────┘    └─────────────┘                          │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Fitur Fungsional

### Untuk Warga (Citizen)

| Fitur | Deskripsi |
|-------|-----------|
| **Registrasi & Login** | Autentikasi berbasis JWT |
| **Buat Laporan** | Laporan dengan judul, deskripsi, kategori, lokasi |
| **Privacy Levels** | Public (semua bisa lihat), Private (hanya pelapor & dinas), Anonymous (identitas disembunyikan) |
| **Kelola Laporan** | Lihat, edit laporan sendiri |
| **Lacak Status** | Lihat progress penyelesaian laporan |
| **Upvote/Downvote** | Vote laporan publik (termasuk laporan sendiri) |
| **Notifikasi** | Real-time notification via SSE saat status laporan berubah |

### Untuk Pemerintah (Government Admin)

| Fitur | Deskripsi |
|-------|-----------|
| **Login dengan Role** | `admin_kebersihan`, `admin_kesehatan`, `admin_infrastruktur` |
| **Lihat Laporan Departemen** | Hanya laporan sesuai kewenangan department |
| **Ubah Status Laporan** | `pending` → `accepted` → `in_progress` → `completed` / `rejected` |
| **Privasi Pelapor** | Identitas pelapor disembunyikan untuk laporan anonymous |
| **Tidak Bisa Hapus** | Laporan tidak dapat dihapus untuk audit trail |

## Privacy Levels Explained

| Level | Visibility | Reporter Identity |
|-------|------------|-------------------|
| `public` | Semua warga & dinas terkait | Terlihat oleh semua |
| `private` | Hanya pelapor & dinas terkait | Terlihat oleh dinas |
| `anonymous` | Hanya pelapor & dinas terkait | **Disembunyikan** (hashed) |

## Tech Stack

| Komponen | Teknologi | Alasan Pemilihan |
|----------|-----------|------------------|
| **API Gateway** | Nginx | Lightweight, proven performance, native reverse proxy |
| **Backend Services** | Go (Gin) | High performance, low memory footprint, excellent concurrency |
| **Frontend** | Next.js (React) | SSR support, built-in routing, optimal for SEO & performance |
| **Database** | PostgreSQL | ACID compliance, JSONB support, mature & reliable |
| **Real-time** | SSE (Server-Sent Events) | Simpler than WebSocket for one-way server→client push |
| **Authentication** | JWT | Stateless, scalable, standard for microservices |
| **Logging** | Loki + Promtail | Lightweight log aggregation, Grafana-native |
| **Visualization** | Grafana | Unified dashboard for logs & metrics |

## Component Interactions

```
┌────────────────────────────────────────────────────────────────────────┐
│                         REQUEST FLOWS                                   │
└────────────────────────────────────────────────────────────────────────┘

1. USER REGISTRATION/LOGIN
   Client → Gateway → Auth Service → PostgreSQL
                   ← JWT Token ←

2. CREATE REPORT
   Client (+ JWT) → Gateway → [Validate JWT via Auth] 
                           → Report Service → PostgreSQL
                           → Notification Service (if applicable)

3. GOVERNMENT VIEW REPORTS
   Client (+ JWT) → Gateway → [Validate JWT + Check Role]
                           → Report Service (filter by department)
                           ← Reports (reporter identity stripped if anonymous)

4. VOTE ON REPORT
   Client (+ JWT) → Gateway → Report Service
                           → Check duplicate vote in report_votes
                           → Update vote_score in reports
                           ← Updated score

5. STATUS UPDATE (triggers notification)
   Govt Client → Gateway → Report Service
                        → Update status in PostgreSQL
                        → Insert notification in notifications table
                        → SSE push to connected clients

6. RECEIVE NOTIFICATIONS (SSE)
   Client (+ JWT) → Gateway → Notification Service
                           ← SSE stream (persistent connection)
                           ← Events pushed when status changes
```

## Database Schema Overview

```sql
-- Core tables
users           -- User accounts with roles
categories      -- Report categories mapped to departments  
reports         -- Citizen reports with privacy levels

-- New tables (to be implemented)
report_votes    -- Track votes per user to prevent duplicates
notifications   -- Persistent notification storage
```

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Node.js 18+ (for frontend development)
- Git

### Run All Services

```bash
# Build and start all containers
docker-compose up --build

# Or run in background
docker-compose up -d --build
```

### Endpoints

| Service | Endpoint | Description |
|---------|----------|-------------|
| Gateway | <http://localhost:8080> | API Gateway |
| Auth | <http://localhost:8080/api/v1/auth/> | Authentication |
| Reports | <http://localhost:8080/api/v1/reports/> | Report Management |
| Notifications | <http://localhost:8080/api/v1/notifications/> | SSE Notifications |
| Grafana | <http://localhost:3000> | Observability Dashboard |

## API Documentation

### Auth Service

#### Register User

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "warga@example.com",
    "password": "password123",
    "name": "Budi Warga",
    "role": "warga"
  }'
```

Available roles:

- `warga` - Citizen (default)
- `admin_kebersihan` - Sanitation Department
- `admin_kesehatan` - Health Department
- `admin_infrastruktur` - Infrastructure Department

#### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "warga@example.com",
    "password": "password123"
  }'
```

### Report Service

#### Create Report

```bash
curl -X POST http://localhost:8080/api/v1/reports/ \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Sampah menumpuk",
    "description": "Di Jalan X sudah 3 hari",
    "category_id": 2,
    "location_lat": -6.2088,
    "location_lng": 106.8456,
    "privacy_level": "public"
  }'
```

#### Get Public Reports (with vote scores)

```bash
curl http://localhost:8080/api/v1/reports/public
```

#### Vote on Report

```bash
curl -X POST http://localhost:8080/api/v1/reports/<report_id>/vote \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"vote_type": "upvote"}'  # or "downvote"
```

#### Update Report Status (Government only)

```bash
curl -X PATCH http://localhost:8080/api/v1/reports/<report_id>/status \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"status": "in_progress"}'
```

### Notification Service

#### Connect to SSE Stream

```bash
curl -N http://localhost:8080/api/v1/notifications/stream \
  -H "Authorization: Bearer <TOKEN>"
```

#### Get Notification History

```bash
curl http://localhost:8080/api/v1/notifications/ \
  -H "Authorization: Bearer <TOKEN>"
```

## Security Features

- **JWT Authentication** - Stateless token-based auth
- **RBAC** - Role-Based Access Control per department
- **Anonymous Reporting** - SHA-256 hashed reporter ID (irreversible)
- **Department Isolation** - Admin only sees relevant department data
- **Audit Trail** - Reports cannot be deleted

## Demo Users (Seed Data)

| Email | Password | Role |
|-------|----------|------|
| <warga@test.com> | password123 | warga |
| <admin_kebersihan@test.com> | password123 | admin_kebersihan |
| <admin_kesehatan@test.com> | password123 | admin_kesehatan |
| <admin_infrastruktur@test.com> | password123 | admin_infrastruktur |

## Project Structure

```
Tubes-AAT/
├── auth-service/           # Authentication microservice (Go)
│   ├── internal/
│   │   ├── handler/        # HTTP handlers
│   │   ├── model/          # Data models
│   │   ├── repository/     # Database access
│   │   └── service/        # Business logic
│   ├── config/
│   ├── main.go
│   └── Dockerfile
├── report-service/         # Report management microservice (Go)
│   ├── internal/
│   │   ├── handler/
│   │   ├── model/
│   │   ├── repository/
│   │   └── service/
│   ├── config/
│   ├── main.go
│   └── Dockerfile
├── notification-service/   # SSE notification service (Go) [TO BE IMPLEMENTED]
├── frontend/               # Next.js application [TO BE IMPLEMENTED]
├── gateway/
│   └── nginx.conf          # API Gateway configuration
├── observability/          # Loki + Grafana config [TO BE IMPLEMENTED]
│   ├── loki/
│   ├── promtail/
│   └── grafana/
├── database/
│   └── init.sql            # Database schema & seed data
├── scripts/
│   ├── test-api.sh
│   └── test-api.ps1
├── docs/
│   └── Spesifikasi.pdf     # Original specification
├── docker-compose.yml
└── go.work
```

## Known Limitations (PoC)

- Password in seed data uses hardcoded hash
- JWT secret stored in config file (production: use env/vault)
- No rate limiting implemented yet
- No horizontal scaling configuration (single instance per service)

## License

Educational Project - Tugas Kuliah AAT
