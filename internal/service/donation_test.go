package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/joan-ouma/give-blood/internal/dto"
	"github.com/joan-ouma/give-blood/internal/entities"
)

func TestDonationService(t *testing.T) {
	db, cleanup := getTestDB(t)
	defer cleanup()

	svc := NewDonationService(db)
	driveSvc := NewDriveService(db)
	locSvc := NewLocationService(db)
	leaderboardSvc := NewLeaderboardService(db)
	ctx := context.Background()

	err := svc.EnsureIndexes(ctx)
	if err != nil {
		t.Fatalf("failed to ensure indexes: %v", err)
	}

	agencyID := primitive.NewObjectID().Hex()
	donorID := primitive.NewObjectID().Hex()

	// 1. Setup Location
	lat := -1.2921
	lng := 36.8219
	loc, err := locSvc.Create(ctx, agencyID, &dto.LocationCreateRequest{
		Name:    "Nairobi HQ Clinic",
		Address: "123 Red Cross Rd",
		City:    "Nairobi",
		Lat:     &lat,
		Lng:     &lng,
	})
	if err != nil {
		t.Fatalf("failed to create location: %v", err)
	}
	locIDStr := loc.ID.Hex()

	// 2. Setup Drive
	drive, err := driveSvc.Create(ctx, agencyID, &dto.DriveCreateRequest{
		Title:      "Drive 1",
		City:       "Nairobi",
		StartsAt:   time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		EndsAt:     time.Now().Add(5 * time.Hour).Format(time.RFC3339),
		LocationID: &locIDStr,
	})
	if err != nil {
		t.Fatalf("failed to create drive: %v", err)
	}
	driveIDStr := drive.ID.Hex()

	var createdDonationID string

	t.Run("Create Valid Donation", func(t *testing.T) {
		req := &dto.DonationCreateRequest{
			DriveID: &driveIDStr,
		}

		donation, err := svc.Create(ctx, donorID, req)
		if err != nil {
			t.Fatalf("failed to create donation: %v", err)
		}

		if donation.Status != entities.StatusPending {
			t.Errorf("expected status %s, got %s", entities.StatusPending, donation.Status)
		}

		createdDonationID = donation.ID.Hex()
	})

	t.Run("Accept RSVP", func(t *testing.T) {
		donation, err := svc.Accept(ctx, agencyID, createdDonationID)
		if err != nil {
			t.Fatalf("failed to accept RSVP: %v", err)
		}

		if donation.Status != entities.StatusAccepted {
			t.Errorf("expected accepted status, got %s", donation.Status)
		}
	})

	t.Run("Verify Donation Valid", func(t *testing.T) {
		donation, err := svc.Verify(ctx, agencyID, createdDonationID, 1)
		if err != nil {
			t.Fatalf("failed to verify donation: %v", err)
		}

		if donation.Status != entities.StatusVerified {
			t.Errorf("expected verified status, got %s", donation.Status)
		}

		if donation.NextEligibleAt == nil {
			t.Error("NextEligibleAt should be computed and populated upon verification")
		}
	})

	t.Run("Verify Already Verified Donation Conflict", func(t *testing.T) {
		_, err := svc.Verify(ctx, agencyID, createdDonationID, 1)
		if !errors.Is(err, ErrConflict) {
			t.Errorf("expected ErrConflict, got %v", err)
		}
	})

	t.Run("Create Second Donation and Reject It", func(t *testing.T) {
		secondDonorID := primitive.NewObjectID().Hex()
		req := &dto.DonationCreateRequest{
			DriveID: &driveIDStr,
		}

		donation, err := svc.Create(ctx, secondDonorID, req)
		if err != nil {
			t.Fatalf("failed to create second donation: %v", err)
		}

		rejectReq := &dto.DonationRejectRequest{
			RejectionReason: "Hemoglobin level below threshold",
		}

		rejected, err := svc.Reject(ctx, agencyID, donation.ID.Hex(), rejectReq)
		if err != nil {
			t.Fatalf("failed to reject donation: %v", err)
		}

		if rejected.Status != entities.StatusRejected {
			t.Errorf("expected rejected status, got %s", rejected.Status)
		}
	})

	t.Run("Donor Eligibility Checks", func(t *testing.T) {
		eligibility, err := leaderboardSvc.GetEligibility(ctx, donorID)
		if err != nil {
			t.Fatalf("failed to get eligibility: %v", err)
		}

		if eligibility.IsEligibleNow {
			t.Error("donor should not be eligible immediately after verification of a valid donation")
		}
	})
}
