# CityConnect - Sistem Pelaporan Warga Terdistribusi

PoC (Proof-of-Concept) untuk sistem pelaporan warga dengan arsitektur microservices.

## 🏗️ Arsitektur

```
┌─────────────┐      ┌──────────────┐      ┌───────────────┐
│   Client    │──────│ API Gateway  │──────│ Auth Service  │
│  (Browser)  │      │   (Nginx)    │      │    (Go)       │
└─────────────┘      └──────────────┘      └───────────────┘
                            │
                            │
                     ┌──────────────┐      ┌───────────────┐
                     │Report Service│──────│  PostgreSQL   │
                     │    (Go)      │      │   Database    │
                     └──────────────┘      └───────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- (Optional) Git for version control

### Jalankan Semua Services

```bash
# Build dan start semua container
docker-compose up --build

# Atau jalankan di background
docker-compose up -d --build
```

### Endpoints

| Service | Endpoint                              | Description       |
| ------- | ------------------------------------- | ----------------- |
| Gateway | http://localhost:8080                 | API Gateway       |
| Auth    | http://localhost:8080/api/v1/auth/    | Authentication    |
| Reports | http://localhost:8080/api/v1/reports/ | Report Management |

## 📋 API Documentation

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

Roles yang tersedia:

- `warga` - Pengguna umum
- `admin_kebersihan` - Admin Dinas Kebersihan
- `admin_kesehatan` - Admin Dinas Kesehatan
- `admin_infrastruktur` - Admin Dinas Infrastruktur

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

Privacy levels:

- `public` - Terlihat semua orang
- `private` - Hanya pelapor dan petugas
- `anonymous` - Identitas pelapor disembunyikan

#### Get Reports

```bash
curl http://localhost:8080/api/v1/reports/ \
  -H "Authorization: Bearer <TOKEN>"
```

## 🧪 Testing

### Menggunakan PowerShell (Windows)

```powershell
.\scripts\test-api.ps1
```

### Menggunakan Bash (Linux/Mac)

```bash
chmod +x scripts/test-api.sh
./scripts/test-api.sh
```

## 📁 Project Structure

```
Tubes-AAT/
├── auth-service/           # Authentication microservice
│   ├── internal/
│   │   ├── handler/        # HTTP handlers
│   │   ├── model/          # Data models
│   │   ├── repository/     # Database access
│   │   └── service/        # Business logic
│   ├── config/
│   ├── main.go
│   └── Dockerfile
├── report-service/         # Report management microservice
│   ├── internal/
│   │   ├── handler/
│   │   ├── model/
│   │   ├── repository/
│   │   └── service/
│   ├── config/
│   ├── main.go
│   └── Dockerfile
├── gateway/
│   └── nginx.conf          # API Gateway configuration
├── database/
│   └── init.sql            # Database schema & seed data
├── scripts/
│   ├── test-api.sh
│   └── test-api.ps1
├── docker-compose.yml
└── go.work
```

## 🔐 Security Features

- **JWT Authentication** - Token-based stateless auth
- **RBAC** - Role-Based Access Control
- **Anonymous Reporting** - SHA-256 hashed reporter ID (irreversible)
- **Department Isolation** - Admin hanya melihat data sesuai departemennya

## 👥 Demo Users (Seed Data)

| Email                        | Password    | Role                |
| ---------------------------- | ----------- | ------------------- |
| warga@test.com               | password123 | warga               |
| admin_kebersihan@test.com    | password123 | admin_kebersihan    |
| admin_kesehatan@test.com     | password123 | admin_kesehatan     |
| admin_infrastruktur@test.com | password123 | admin_infrastruktur |

## ⚠️ Known Limitations (PoC)

- Password di seed data menggunakan hash hardcoded
- JWT secret disimpan di config file (production harus pakai env/vault)
- Belum ada rate limiting
- Belum ada logging terpusat

## 📝 License

Educational Project - Tugas Kuliah AAT
