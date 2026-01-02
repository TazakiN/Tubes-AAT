package model

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleWarga           Role = "warga"
	RoleAdminKebersihan Role = "admin_kebersihan"
	RoleAdminKesehatan  Role = "admin_kesehatan"
	RoleAdminInfrastruktur Role = "admin_infrastruktur"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Name         string     `json:"name"`
	Role         Role       `json:"role"`
	Department   *string    `json:"department,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Request/Response
type RegisterRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=6"`
	Name       string `json:"name" binding:"required"`
	Role       Role   `json:"role" binding:"required"`
	Department string `json:"department"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type ValidateResponse struct {
	Valid      bool   `json:"valid"`
	UserID     string `json:"user_id"`
	Role       string `json:"role"`
	Department string `json:"department"`
	Name       string `json:"name"`
}
