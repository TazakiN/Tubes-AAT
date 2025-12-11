package service

import (
	"errors"
	"time"

	"auth-service/config"
	"auth-service/internal/model"
	"auth-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo  *repository.UserRepository
	jwtConfig config.JWTConfig
}

func NewAuthService(userRepo *repository.UserRepository, jwtConfig config.JWTConfig) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtConfig: jwtConfig,
	}
}

func (s *AuthService) Register(req *model.RegisterRequest) (*model.User, error) {
	// Check if email exists
	exists, err := s.userRepo.EmailExists(req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Map role to department
	var department *string
	switch req.Role {
	case model.RoleAdminKebersihan:
		dept := "kebersihan"
		department = &dept
	case model.RoleAdminKesehatan:
		dept := "kesehatan"
		department = &dept
	case model.RoleAdminInfrastruktur:
		dept := "infrastruktur"
		department = &dept
	}

	user := &model.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
		Role:         req.Role,
		Department:   department,
		CreatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(req *model.LoginRequest) (*model.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Generate JWT
	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*model.ValidateResponse, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(s.jwtConfig.Secret), nil
	})

	if err != nil || !token.Valid {
		return &model.ValidateResponse{Valid: false}, nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return &model.ValidateResponse{Valid: false}, nil
	}

	department := ""
	if dept, ok := claims["department"].(string); ok {
		department = dept
	}
	name := ""
	if n, ok := claims["name"].(string); ok {
		name = n
	}

	return &model.ValidateResponse{
		Valid:      true,
		UserID:     claims["user_id"].(string),
		Role:       claims["role"].(string),
		Department: department,
		Name:       name,
	}, nil
}

func (s *AuthService) GetUserByID(id uuid.UUID) (*model.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *AuthService) generateToken(user *model.User) (string, error) {
	department := ""
	if user.Department != nil {
		department = *user.Department
	}

	claims := jwt.MapClaims{
		"user_id":    user.ID.String(),
		"email":      user.Email,
		"name":       user.Name,
		"role":       string(user.Role),
		"department": department,
		"exp":        time.Now().Add(time.Hour * time.Duration(s.jwtConfig.ExpirationHours)).Unix(),
		"iat":        time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtConfig.Secret))
}
