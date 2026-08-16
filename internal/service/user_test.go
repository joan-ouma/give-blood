package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/joan-ouma/give-blood/internal/dto"
)

func getTestDB(t *testing.T) (*mongo.Database, func()) {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		// Fallback to local docker container mapped port
		mongoURI = "mongodb://root:example@localhost:27017/blood_donation_test?authSource=admin"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Skipf("Skipping integration test; failed to connect to MongoDB: %v", err)
	}

	db := client.Database("blood_donation_test")

	// Cleanup function
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	}

	return db, cleanup
}

func TestUserService(t *testing.T) {
	db, cleanup := getTestDB(t)
	defer cleanup()

	svc := NewUserService(db)
	ctx := context.Background()

	err := svc.EnsureIndexes(ctx)
	if err != nil {
		t.Fatalf("failed to ensure indexes: %v", err)
	}

	t.Run("Valid Registration", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Email:    "newuser@example.com",
			Password: "securepassword123",
			Role:     "donor",
			Name:     "John Doe",
		}

		u, err := svc.Register(ctx, req)
		if err != nil {
			t.Fatalf("failed to register user: %v", err)
		}

		if u.Email != "newuser@example.com" {
			t.Errorf("expected email newuser@example.com, got %s", u.Email)
		}

		if u.Name != "John Doe" {
			t.Errorf("expected name John Doe, got %s", u.Name)
		}

		if u.PasswordHash == "" || u.PasswordHash == req.Password {
			t.Error("password was not hashed correctly")
		}
	})

	t.Run("Duplicate Email Registration", func(t *testing.T) {
		latVal := -1.2921
		lngVal := 36.8219
		req := &dto.RegisterRequest{
			Email:    "newuser@example.com", // Duplicate
			Password: "otherpassword123",
			Role:     "agency",
			Name:     "Duplicate User",
			Lat:      &latVal,
			Lng:      &lngVal,
		}

		_, err := svc.Register(ctx, req)
		if !errors.Is(err, ErrAlreadyExists) {
			t.Errorf("expected ErrAlreadyExists error, got %v", err)
		}
	})

	t.Run("Missing Agency Coordinates Registration", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Email:    "agencycoord@example.com",
			Password: "securepassword123",
			Role:     "agency",
			Name:     "No Coords Agency",
		}

		_, err := svc.Register(ctx, req)
		if !errors.Is(err, ErrInvalidCoords) {
			t.Errorf("expected ErrInvalidCoords error, got %v", err)
		}
	})

	t.Run("Invalid Email Format Registration", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Email:    "invalid-email",
			Password: "securepassword123",
			Role:     "donor",
			Name:     "Invalid Email User",
		}

		_, err := svc.Register(ctx, req)
		if !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("expected ErrInvalidEmail error, got %v", err)
		}
	})

	t.Run("Short Password Registration", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Email:    "short@example.com",
			Password: "short",
			Role:     "donor",
			Name:     "Short Password User",
		}

		_, err := svc.Register(ctx, req)
		if !errors.Is(err, ErrInvalidPass) {
			t.Errorf("expected ErrInvalidPass error, got %v", err)
		}
	})

	t.Run("Invalid Role Registration", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Email:    "badrole@example.com",
			Password: "securepassword123",
			Role:     "admin", // Invalid role
			Name:     "Bad Role User",
		}

		_, err := svc.Register(ctx, req)
		if !errors.Is(err, ErrInvalidRole) {
			t.Errorf("expected ErrInvalidRole error, got %v", err)
		}
	})

	t.Run("Authenticate Valid User", func(t *testing.T) {
		req := &dto.LoginRequest{
			Email:    "newuser@example.com",
			Password: "securepassword123",
		}

		u, err := svc.Authenticate(ctx, req)
		if err != nil {
			t.Fatalf("failed to authenticate valid user: %v", err)
		}

		if u.Email != "newuser@example.com" {
			t.Errorf("expected email newuser@example.com, got %s", u.Email)
		}
	})

	t.Run("Authenticate Invalid Password", func(t *testing.T) {
		req := &dto.LoginRequest{
			Email:    "newuser@example.com",
			Password: "wrongpassword",
		}

		_, err := svc.Authenticate(ctx, req)
		if err == nil {
			t.Error("expected authentication failure for wrong password")
		}
	})

	t.Run("Authenticate Non-existent User", func(t *testing.T) {
		req := &dto.LoginRequest{
			Email:    "notfound@example.com",
			Password: "password123",
		}

		_, err := svc.Authenticate(ctx, req)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("Get User By ID", func(t *testing.T) {
		// Find registered user first
		var existing bson.M
		err := db.Collection("users").FindOne(ctx, bson.M{"email": "newuser@example.com"}).Decode(&existing)
		if err != nil {
			t.Fatalf("failed to find user: %v", err)
		}

		idHex := existing["_id"].(primitive.ObjectID).Hex()
		u, err := svc.GetByID(ctx, idHex)
		if err != nil {
			t.Fatalf("failed to get user by ID: %v", err)
		}

		if u.Email != "newuser@example.com" {
			t.Errorf("expected email newuser@example.com, got %s", u.Email)
		}
	})
}
