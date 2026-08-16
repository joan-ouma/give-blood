package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/joan-ouma/give-blood/internal/dto"
)

func TestDriveService(t *testing.T) {
	db, cleanup := getTestDB(t)
	defer cleanup()

	svc := NewDriveService(db)
	locSvc := NewLocationService(db)
	ctx := context.Background()

	err := svc.EnsureIndexes(ctx)
	if err != nil {
		t.Fatalf("failed to ensure indexes: %v", err)
	}

	agencyID := primitive.NewObjectID().Hex()
	otherAgencyID := primitive.NewObjectID().Hex()

	// Setup a Location to link
	lat := -1.2921
	lng := 36.8219
	loc, err := locSvc.Create(ctx, agencyID, &dto.LocationCreateRequest{
		Name:    "Central Clinic for Drive",
		Address: "123 Main St",
		City:    "Nairobi",
		Lat:     &lat,
		Lng:     &lng,
	})
	if err != nil {
		t.Fatalf("failed to create pre-requisite location: %v", err)
	}
	locIDStr := loc.ID.Hex()

	var createdDriveID string

	t.Run("Create Valid Drive with Location", func(t *testing.T) {
		req := &dto.DriveCreateRequest{
			Title:      "Annual Blood Drive",
			City:       "Nairobi",
			StartsAt:   time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			EndsAt:     time.Now().Add(5 * time.Hour).Format(time.RFC3339),
			LocationID: &locIDStr,
		}

		drive, err := svc.Create(ctx, agencyID, req)
		if err != nil {
			t.Fatalf("failed to create drive: %v", err)
		}

		if drive.Title != "Annual Blood Drive" {
			t.Errorf("expected drive title Annual Blood Drive, got %s", drive.Title)
		}

		createdDriveID = drive.ID.Hex()
	})

	t.Run("Create Valid Drive with custom Address", func(t *testing.T) {
		req := &dto.DriveCreateRequest{
			Title:    "Community Center Drive",
			City:     "Nairobi",
			StartsAt: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			EndsAt:   time.Now().Add(5 * time.Hour).Format(time.RFC3339),
			Address:  "Community Hall Room 2",
		}

		drive, err := svc.Create(ctx, agencyID, req)
		if err != nil {
			t.Fatalf("failed to create drive: %v", err)
		}

		if drive.Address != "Community Hall Room 2" {
			t.Errorf("expected community address, got %s", drive.Address)
		}
	})

	t.Run("Create Drive Date Mismatch", func(t *testing.T) {
		// StartsAt after EndsAt
		req := &dto.DriveCreateRequest{
			Title:    "Mismatch Drive",
			City:     "Nairobi",
			StartsAt: time.Now().Add(5 * time.Hour).Format(time.RFC3339),
			EndsAt:   time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			Address:  "Nairobi",
		}

		_, err := svc.Create(ctx, agencyID, req)
		if !errors.Is(err, ErrDriveValidation) {
			t.Errorf("expected ErrDriveValidation, got %v", err)
		}
	})

	t.Run("Update Drive Valid", func(t *testing.T) {
		req := &dto.DriveUpdateRequest{
			Title:    "Annual Blood Drive Updated",
			City:     "Nairobi",
			StartsAt: time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			EndsAt:   time.Now().Add(6 * time.Hour).Format(time.RFC3339),
			Address:  "Central Plaza",
		}

		drive, err := svc.Update(ctx, agencyID, createdDriveID, req)
		if err != nil {
			t.Fatalf("failed to update drive: %v", err)
		}

		if drive.Title != "Annual Blood Drive Updated" {
			t.Errorf("expected updated title, got %s", drive.Title)
		}
	})

	t.Run("Update Drive Unauthorized", func(t *testing.T) {
		req := &dto.DriveUpdateRequest{
			Title:    "Hacked Drive",
			City:     "Nairobi",
			StartsAt: time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			EndsAt:   time.Now().Add(6 * time.Hour).Format(time.RFC3339),
			Address:  "Central Plaza",
		}

		_, err := svc.Update(ctx, otherAgencyID, createdDriveID, req)
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("Get Drive By ID", func(t *testing.T) {
		drive, err := svc.GetByID(ctx, createdDriveID)
		if err != nil {
			t.Fatalf("failed to get drive: %v", err)
		}

		if drive.Title != "Annual Blood Drive Updated" {
			t.Errorf("expected title Annual Blood Drive Updated, got %s", drive.Title)
		}
	})

	t.Run("List Drives", func(t *testing.T) {
		drives, err := svc.List(ctx, "Nairobi", true, 10, 0)
		if err != nil {
			t.Fatalf("failed to list drives: %v", err)
		}

		if len(drives) != 2 {
			t.Errorf("expected 2 drives, got %d", len(drives))
		}
	})

	t.Run("Delete Drive Valid", func(t *testing.T) {
		err := svc.Delete(ctx, agencyID, createdDriveID)
		if err != nil {
			t.Fatalf("failed to delete drive: %v", err)
		}

		_, err = svc.GetByID(ctx, createdDriveID)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}
