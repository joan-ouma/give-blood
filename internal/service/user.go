package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/joan-ouma/give-blood/internal/auth"
	"github.com/joan-ouma/give-blood/internal/dto"
	"github.com/joan-ouma/give-blood/internal/entities"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrAlreadyExists = errors.New("user already exists")
	ErrInvalidEmail  = errors.New("invalid email format")
	ErrInvalidPass   = errors.New("password must be at least 8 characters")
	ErrInvalidRole   = errors.New("role must be agency or donor")
	ErrInvalidName   = errors.New("name is required")
)

type UserService struct {
	col *mongo.Collection
}

func NewUserService(db *mongo.Database) *UserService {
	return &UserService{
		col: db.Collection("users"),
	}
}

func (s *UserService) EnsureIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := s.col.Indexes().CreateOne(ctx, indexModel)
	return err
}

func (s *UserService) Register(ctx context.Context, req *dto.RegisterRequest) (*entities.User, error) {
	email := strings.TrimSpace(req.Email)
	password := req.Password
	role := entities.Role(strings.TrimSpace(req.Role))
	name := strings.TrimSpace(req.Name)

	if _, err := mail.ParseAddress(email); err != nil {
		return nil, ErrInvalidEmail
	}
	if len(password) < 8 {
		return nil, ErrInvalidPass
	}
	if role != entities.RoleAgency && role != entities.RoleDonor {
		return nil, ErrInvalidRole
	}
	if name == "" {
		return nil, ErrInvalidName
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	u := &entities.User{
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		Name:         name,
	}

	u.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	_, err = s.col.InsertOne(ctx, u)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}

	return u, nil
}

func (s *UserService) Authenticate(ctx context.Context, req *dto.LoginRequest) (*entities.User, error) {
	email := strings.TrimSpace(req.Email)
	if email == "" || req.Password == "" {
		return nil, errors.New("email and password required")
	}

	var u entities.User
	err := s.col.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if !auth.CheckPasswordHash(req.Password, u.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	return &u, nil
}

func (s *UserService) GetByID(ctx context.Context, idStr string) (*entities.User, error) {
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return nil, errors.New("invalid id format")
	}

	var u entities.User
	err = s.col.FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &u, nil
}
