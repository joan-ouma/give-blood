package service

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/joan-ouma/give-blood/internal/dto"
)

func TestLocationService(t *testing.T) {
	db, cleanup := getTestDB(t)
	defer cleanup()

	svc := NewLocationService(db)
	ctx := context.Background()

	err := svc.EnsureIndexes(ctx)
	if err != nil {
		t.Fatalf("failed to ensure indexes: %v", err)
	}

	agencyID := primitive.NewObjectID().Hex()
	otherAgencyID := primitive.NewObjectID().Hex()
	var createdLocID string

	t.Run("Create Valid Location", func(t *testing.T) {
		lat := -1.2921
		lng := 36.8219
		req := &dto.LocationCreateRequest{
			Name:    "Central Clinic",
			Address: "123 Main St",
			City:    "Nairobi",
			Lat:     &lat,
			Lng:     &lng,
			Hours:   "08:00 - 17:00",
			Phone:   "+254 700 000 000",
		}

		loc, err := svc.Create(ctx, agencyID, req)
		if err != nil {
			t.Fatalf("failed to create location: %v", err)
		}

		if loc.Name != "Central Clinic" {
			t.Errorf("expected name Central Clinic, got %s", loc.Name)
		}

		createdLocID = loc.ID.Hex()
	})

	t.Run("Create Invalid Lat/Lng Location", func(t *testing.T) {
		badLat := 120.0 // Invalid latitude
		req := &dto.LocationCreateRequest{
			Name: "Bad Clinic",
			City: "Nairobi",
			Lat:  &badLat,
		}

		_, err := svc.Create(ctx, agencyID, req)
		if !errors.Is(err, ErrLocationValidation) {
			t.Errorf("expected ErrLocationValidation, got %v", err)
		}
	})

	t.Run("Create Empty Name Location", func(t *testing.T) {
		req := &dto.LocationCreateRequest{
			Name: "", // Invalid
			City: "Nairobi",
		}

		_, err := svc.Create(ctx, agencyID, req)
		if !errors.Is(err, ErrLocationValidation) {
			t.Errorf("expected ErrLocationValidation, got %v", err)
		}
	})

	t.Run("Update Location Valid", func(t *testing.T) {
		lat := -1.2900
		req := &dto.LocationUpdateRequest{
			Name:    "Central Clinic Updated",
			Address: "124 Main St",
			City:    "Nairobi",
			Lat:     &lat,
		}

		loc, err := svc.Update(ctx, agencyID, createdLocID, req)
		if err != nil {
			t.Fatalf("failed to update location: %v", err)
		}

		if loc.Name != "Central Clinic Updated" {
			t.Errorf("expected updated name, got %s", loc.Name)
		}
	})

	t.Run("Update Location Unauthorized", func(t *testing.T) {
		req := &dto.LocationUpdateRequest{
			Name: "Hacked Clinic",
			City: "Nairobi",
		}

		_, err := svc.Update(ctx, otherAgencyID, createdLocID, req)
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("Get Location By ID", func(t *testing.T) {
		loc, err := svc.GetByID(ctx, createdLocID)
		if err != nil {
			t.Fatalf("failed to get location: %v", err)
		}

		if loc.Name != "Central Clinic Updated" {
			t.Errorf("expected name Central Clinic Updated, got %s", loc.Name)
		}
	})

	t.Run("List Locations", func(t *testing.T) {
		locs, err := svc.List(ctx, "Nairobi", 10, 0)
		if err != nil {
			t.Fatalf("failed to list locations: %v", err)
		}

		if len(locs) != 1 {
			t.Errorf("expected 1 location, got %d", len(locs))
		}
	})

	t.Run("Delete Location Unauthorized", func(t *testing.T) {
		err := svc.Delete(ctx, otherAgencyID, createdLocID)
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("Delete Location Valid", func(t *testing.T) {
		err := svc.Delete(ctx, agencyID, createdLocID)
		if err != nil {
			t.Fatalf("failed to delete location: %v", err)
		}

		_, err = svc.GetByID(ctx, createdLocID)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}
