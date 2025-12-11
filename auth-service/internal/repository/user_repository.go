package repository

import (
	"database/sql"
	"fmt"

	"auth-service/internal/model"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, role, department, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Role,
		user.Department,
		user.CreatedAt,
	)
	return err
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, department, created_at
		FROM users WHERE email = $1
	`
	user := &model.User{}
	var dept sql.NullString

	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&dept,
		&user.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	if dept.Valid {
		user.Department = &dept.String
	}

	return user, nil
}

func (r *UserRepository) FindByID(id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, department, created_at
		FROM users WHERE id = $1
	`
	user := &model.User{}
	var dept sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&dept,
		&user.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	if dept.Valid {
		user.Department = &dept.String
	}

	return user, nil
}

func (r *UserRepository) EmailExists(email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	err := r.db.QueryRow(query, email).Scan(&exists)
	return exists, err
}
