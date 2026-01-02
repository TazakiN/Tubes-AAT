# CityConnect - Sistem Pelaporan Warga

Platform pelaporan masalah lingkungan berbasis microservices untuk warga kota.

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Git

### Setup & Run

```bash
# 1. Clone repository
git clone <repository-url>
cd Tubes-AAT

# 2. Copy environment file
cp .env.example .env

# 3. Build and start all services
docker-compose up -d --build

# 4. Wait for services to be healthy (~30 seconds)
docker-compose ps
```

### Access Points

| Service | URL | Credentials |
|---------|-----|-------------|
| **Frontend** | <http://localhost:8080> | — |
| **API Gateway** | <http://localhost:8080/api/v1> | — |
| **RabbitMQ Management** | <http://localhost:15672> | cityconnect / cityconnect_secret |
| **Grafana** | <http://localhost:3050> | admin / admin |

## Demo Accounts

| Email | Password | Role |
|-------|----------|------|
| <warga@test.com> | password123 | Citizen |
| <admin_kebersihan@test.com> | password123 | Admin Kebersihan |
| <admin_kesehatan@test.com> | password123 | Admin Kesehatan |
| <admin_infrastruktur@test.com> | password123 | Admin Infrastruktur |

## Features

### For Citizens (Warga)

- ✅ Register & Login with JWT authentication
- ✅ Create reports with public/private/anonymous privacy levels
- ✅ Search and filter reports by keyword and category
- ✅ Upvote/downvote public reports
- ✅ Real-time notifications via SSE when report status changes
- ✅ Create custom categories for reports

### For Government Admins

- ✅ View reports filtered by department
- ✅ Update report status (pending → accepted → in_progress → completed/rejected)
- ✅ Reports cannot be deleted (audit trail)
- ✅ Anonymous reporter identity hidden

## Architecture

```
┌─────────────┐     ┌─────────────────────────────────────────┐
│   Frontend  │────▶│           Nginx Gateway (:8080)         │
│  (Next.js)  │     │  - JWT validation via auth_request      │
└─────────────┘     │  - Route to backend services            │
                    └────────┬───────────────┬────────────────┘
                             │               │
                    ┌────────▼───────┐ ┌─────▼────────┐
                    │  Auth Service  │ │ Report Service│
                    │   (Go :3001)   │ │   (Go :3002)  │
                    │  - Register    │ │  - CRUD       │
                    │  - Login       │ │  - Voting     │
                    │  - JWT         │ │  - SSE Notif  │
                    └────────┬───────┘ └──────┬────────┘
                             │                │
                             │         ┌──────▼────────┐
                             │         │   RabbitMQ    │
                             │         │   (:5672)     │
                             │         │  - Pub/Sub    │
                             │         └──────┬────────┘
                             │                │
                             └───────┬────────┘
                                     ▼
                            ┌────────────────┐
                            │   PostgreSQL   │
                            │    (:5432)     │
                            └────────────────┘
```

### Message Flow

1. Admin updates report status via API
2. Report Service publishes message to RabbitMQ
3. Notification Consumer receives message from queue
4. Consumer saves notification to database
5. Consumer broadcasts to SSE Hub
6. SSE Hub pushes to connected frontend clients

## API Endpoints

### Authentication

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"pass123","name":"User","role":"warga"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"pass123"}'
```

### Reports

```bash
# Get public reports (with search)
curl "http://localhost:8080/api/v1/reports/public?search=jalan&category_id=7"

# Create report (requires token)
curl -X POST http://localhost:8080/api/v1/reports/ \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"title":"Title","description":"Desc","category_id":7,"privacy_level":"public"}'

# Vote on report
curl -X POST http://localhost:8080/api/v1/reports/<ID>/vote \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"vote_type":"upvote"}'

# Update status (admin only)
curl -X PATCH http://localhost:8080/api/v1/reports/<ID>/status \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"status":"in_progress"}'
```

### Notifications

```bash
# Get notifications
curl http://localhost:8080/api/v1/notifications \
  -H "Authorization: Bearer <TOKEN>"

# SSE stream
curl -N http://localhost:8080/api/v1/notifications/stream \
  -H "Authorization: Bearer <TOKEN>"
```

## Project Structure

```
Tubes-AAT/
├── auth-service/          # Authentication service (Go)
├── report-service/        # Report & notification service (Go)
├── frontend/              # Next.js frontend
├── gateway/               # Nginx configuration
├── database/              # SQL schema & seed data
├── observability/         # Loki/Grafana/Promtail config
├── scripts/               # Utility scripts
├── docker-compose.yml
├── .env.example
└── README.md
```

## Environment Variables

Copy `.env.example` to `.env` and configure:

```env
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=cityconnect
DB_PASSWORD=cityconnect_secret
DB_NAME=cityconnect

# JWT
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# Configuration
CONFIG_PATH=/app/config.json
```

## Development

### Run individual services

```bash
# Backend only
docker-compose up -d postgres auth-service report-service gateway

# Frontend development (with hot reload)
cd frontend && npm install && npm run dev
```

### Rebuild after changes

```bash
# Rebuild specific service
docker-compose build --no-cache report-service
docker-compose up -d report-service

# Rebuild all
docker-compose up -d --build
```

### View logs

```bash
docker-compose logs -f report-service
docker-compose logs -f auth-service
```

## Tech Stack

| Component | Technology |
|-----------|------------|
| Gateway | Nginx |
| Backend | Go (Gin) |
| Frontend | Next.js 14 |
| Database | PostgreSQL 15 |
| Message Broker | RabbitMQ 3.12 |
| Real-time | SSE |
| Auth | JWT |
| Logging | Loki + Promtail |
| Dashboard | Grafana |

## Known Limitations (PoC)

- JWT secret in config (use vault in production)
- No rate limiting
- Single instance per service (no horizontal scaling)
- Observability stack not yet configured for production

## License

Educational Project - Tugas Kuliah AAT 2024
