-- CityConnect Database Schema
-- PostgreSQL initialization script

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =====================
-- USERS TABLE
-- =====================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (
        role IN (
            'warga',
            'admin_kebersihan',
            'admin_kesehatan',
            'admin_infrastruktur'
        )
    ),
    department VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for faster email lookup
CREATE INDEX idx_users_email ON users (email);

CREATE INDEX idx_users_role ON users (role);

-- =====================
-- CATEGORIES TABLE
-- =====================
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department VARCHAR(100) NOT NULL
);

-- Index for department filtering
CREATE INDEX idx_categories_department ON categories (department);

-- =====================
-- REPORTS TABLE
-- =====================
CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    category_id INTEGER REFERENCES categories (id),
    location_lat DECIMAL(10, 8),
    location_lng DECIMAL(11, 8),
    photo_url TEXT,
    privacy_level VARCHAR(20) NOT NULL CHECK (
        privacy_level IN (
            'public',
            'private',
            'anonymous'
        )
    ),
    reporter_id UUID REFERENCES users (id), -- NULL for anonymous reports
    reporter_hash VARCHAR(64), -- SHA-256 hash for anonymous reports (abuse detection)
    status VARCHAR(50) DEFAULT 'pending' CHECK (
        status IN (
            'pending',
            'accepted',
            'in_progress',
            'completed',
            'rejected'
        )
    ),
    vote_score INTEGER DEFAULT 0, -- Net score (upvotes - downvotes)
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX idx_reports_category ON reports (category_id);

CREATE INDEX idx_reports_status ON reports (status);

CREATE INDEX idx_reports_reporter ON reports (reporter_id);

CREATE INDEX idx_reports_created ON reports (created_at DESC);

CREATE INDEX idx_reports_vote_score ON reports (vote_score DESC);

-- =====================
-- REPORT VOTES TABLE
-- =====================
-- Tracks individual votes to prevent duplicates and enable vote changes
CREATE TABLE report_votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    report_id UUID NOT NULL REFERENCES reports (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    vote_type VARCHAR(10) NOT NULL CHECK (vote_type IN ('upvote', 'downvote')),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (report_id, user_id) -- One vote per user per report
);

CREATE INDEX idx_report_votes_report ON report_votes (report_id);

CREATE INDEX idx_report_votes_user ON report_votes (user_id);

-- =====================
-- NOTIFICATIONS TABLE
-- =====================
-- Persistent notifications for status updates
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    report_id UUID REFERENCES reports (id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications (user_id);

CREATE INDEX idx_notifications_unread ON notifications (user_id, is_read)
WHERE
    is_read = FALSE;

-- =====================
-- OUTBOX TABLE (Transactional Outbox Pattern)
-- =====================
-- Stores messages to be published to RabbitMQ
CREATE TABLE outbox_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    routing_key VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    published_at TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    last_error TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'published', 'failed'))
);

CREATE INDEX idx_outbox_pending ON outbox_messages (status, created_at)
WHERE status = 'pending';

CREATE INDEX idx_outbox_status ON outbox_messages (status);

-- =====================
-- PROCESSED MESSAGES TABLE (Idempotency)
-- =====================
-- Tracks processed message IDs to prevent duplicate processing
CREATE TABLE processed_messages (
    message_id VARCHAR(255) PRIMARY KEY,
    processed_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_processed_messages_time ON processed_messages (processed_at);

-- Clean up old processed messages (older than 7 days) - run periodically
-- DELETE FROM processed_messages WHERE processed_at < NOW() - INTERVAL '7 days';

-- =====================
-- SEED DATA - Categories
-- =====================
INSERT INTO
    categories (name, department)
VALUES (
        'Kebersihan Jalan',
        'kebersihan'
    ),
    (
        'Sampah Menumpuk',
        'kebersihan'
    ),
    (
        'Saluran Air Tersumbat',
        'kebersihan'
    ),
    ('Wabah Penyakit', 'kesehatan'),
    (
        'Klinik Kurang Fasilitas',
        'kesehatan'
    ),
    ('Vaksinasi', 'kesehatan'),
    (
        'Jalan Rusak',
        'infrastruktur'
    ),
    (
        'Lampu Jalan Mati',
        'infrastruktur'
    ),
    (
        'Jembatan Rusak',
        'infrastruktur'
    );

-- =====================
-- SEED DATA - Demo Users
-- Password: 'password123' (bcrypt hash)
-- =====================
INSERT INTO
    users (
        id,
        email,
        password_hash,
        name,
        role,
        department
    )
VALUES (
        '11111111-1111-1111-1111-111111111111',
        'warga@test.com',
        '$2a$12$pWAZ3QeIFtafoCTTZ4hkQezmUYPy5NSndf3XDSMKAJWd7ol9uQtEq',
        'Budi Warga',
        'warga',
        NULL
    ),
    (
        '22222222-2222-2222-2222-222222222222',
        'admin_kebersihan@test.com',
        '$2a$12$pWAZ3QeIFtafoCTTZ4hkQezmUYPy5NSndf3XDSMKAJWd7ol9uQtEq',
        'Admin Kebersihan',
        'admin_kebersihan',
        'kebersihan'
    ),
    (
        '33333333-3333-3333-3333-333333333333',
        'admin_kesehatan@test.com',
        '$2a$12$pWAZ3QeIFtafoCTTZ4hkQezmUYPy5NSndf3XDSMKAJWd7ol9uQtEq',
        'Admin Kesehatan',
        'admin_kesehatan',
        'kesehatan'
    ),
    (
        '44444444-4444-4444-4444-444444444444',
        'admin_infrastruktur@test.com',
        '$2a$12$pWAZ3QeIFtafoCTTZ4hkQezmUYPy5NSndf3XDSMKAJWd7ol9uQtEq',
        'Admin Infrastruktur',
        'admin_infrastruktur',
        'infrastruktur'
    );

-- =====================
-- SEED DATA - Sample Reports
-- =====================
INSERT INTO
    reports (
        id,
        title,
        description,
        category_id,
        location_lat,
        location_lng,
        privacy_level,
        reporter_id,
        status
    )
VALUES (
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        'Sampah menumpuk di Jalan Merdeka',
        'Sudah 3 hari tidak diangkut',
        2,
        -6.2088,
        106.8456,
        'public',
        '11111111-1111-1111-1111-111111111111',
        'pending'
    ),
    (
        'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
        'Jalan berlubang berbahaya',
        'Lubang besar di depan pasar',
        7,
        -6.2100,
        106.8500,
        'public',
        '11111111-1111-1111-1111-111111111111',
        'in_progress'
    );

COMMENT ON TABLE users IS 'User accounts for CityConnect - warga and admin dinas';

COMMENT ON TABLE categories IS 'Report categories mapped to departments';

COMMENT ON TABLE reports IS 'Citizen reports with privacy levels';