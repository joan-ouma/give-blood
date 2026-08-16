package user

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/joan-ouma/give-blood/internal/auth"
)

var (
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrInvalidPassword = errors.New("password must be at least 8 characters")
	ErrInvalidRole     = errors.New("role must be agency or donor")
	ErrInvalidName     = errors.New("name is required")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*User, error) {
	email := strings.TrimSpace(req.Email)
	password := req.Password
	role := Role(strings.TrimSpace(req.Role))
	name := strings.TrimSpace(req.Name)

	if _, err := mail.ParseAddress(email); err != nil {
		return nil, ErrInvalidEmail
	}
	if len(password) < 8 {
		return nil, ErrInvalidPassword
	}
	if role != RoleAgency && role != RoleDonor {
		return nil, ErrInvalidRole
	}
	if name == "" {
		return nil, ErrInvalidName
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	u := &User{
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		Name:         name,
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) Authenticate(ctx context.Context, req *LoginRequest) (*User, error) {
	email := strings.TrimSpace(req.Email)
	if email == "" || req.Password == "" {
		return nil, errors.New("email and password required")
	}

	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if !auth.CheckPasswordHash(req.Password, u.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	return u, nil
}

func (s *Service) GetByID(ctx context.Context, idStr string) (*User, error) {
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return nil, errors.New("invalid id format")
	}
	return s.repo.GetByID(ctx, id)
}
