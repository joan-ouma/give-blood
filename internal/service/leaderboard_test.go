package service

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/joan-ouma/give-blood/internal/entities"
)

func TestLeaderboardService(t *testing.T) {
	db, cleanup := getTestDB(t)
	defer cleanup()

	svc := NewLeaderboardService(db)
	ctx := context.Background()

	err := svc.EnsureIndexes(ctx)
	if err != nil {
		t.Fatalf("failed to ensure indexes: %v", err)
	}

	donorID1 := primitive.NewObjectID()
	donorID2 := primitive.NewObjectID()

	// Seed users for names
	_, _ = db.Collection("users").InsertMany(ctx, []interface{}{
		entities.User{ID: donorID1, Name: "Donor One", Email: "donor1@example.com"},
		entities.User{ID: donorID2, Name: "Donor Two", Email: "donor2@example.com"},
	})

	// Seed donor stats for leaderboard rankings
	_, _ = db.Collection("donor_stats").InsertMany(ctx, []interface{}{
		entities.DonorStats{
			ID:             donorID1,
			Points:         100,
			TotalDonations: 5,
			TotalPints:     8,
			UpdatedAt:      time.Now().UTC(),
		},
		entities.DonorStats{
			ID:             donorID2,
			Points:         50,
			TotalDonations: 2,
			TotalPints:     3,
			UpdatedAt:      time.Now().UTC(),
		},
	})

	t.Run("Get Eligibility for Never Donated", func(t *testing.T) {
		newDonor := primitive.NewObjectID().Hex()
		res, err := svc.GetEligibility(ctx, newDonor)
		if err != nil {
			t.Fatalf("failed to get eligibility: %v", err)
		}

		if !res.IsEligibleNow {
			t.Error("new donor should be eligible immediately")
		}

		if res.LastDonationAt != nil {
			t.Error("LastDonationAt should be nil for new donor")
		}
	})

	t.Run("Get Leaderboard Rankings", func(t *testing.T) {
		res, err := svc.GetLeaderboard(ctx, donorID1.Hex(), 10, 0)
		if err != nil {
			t.Fatalf("failed to get leaderboard: %v", err)
		}

		if len(res.Entries) != 2 {
			t.Fatalf("expected 2 leaderboard entries, got %d", len(res.Entries))
		}

		// Rank 1 should be donorID1 (100 points)
		if res.Entries[0].DonorID != donorID1.Hex() {
			t.Errorf("expected rank 1 to be %s, got %s", donorID1.Hex(), res.Entries[0].DonorID)
		}
		if res.Entries[0].Rank != 1 {
			t.Errorf("expected rank 1, got %d", res.Entries[0].Rank)
		}

		// Rank 2 should be donorID2 (50 points)
		if res.Entries[1].DonorID != donorID2.Hex() {
			t.Errorf("expected rank 2 to be %s, got %s", donorID2.Hex(), res.Entries[1].DonorID)
		}
		if res.Entries[1].Rank != 2 {
			t.Errorf("expected rank 2, got %d", res.Entries[1].Rank)
		}
	})

	t.Run("Leaderboard Caching Check", func(t *testing.T) {
		// Fetch once to populate cache
		_, _ = svc.GetLeaderboard(ctx, donorID1.Hex(), 10, 0)

		// Modify values directly in database
		_, _ = db.Collection("donor_stats").UpdateOne(
			ctx,
			bson.M{"_id": donorID2},
			bson.M{"$set": bson.M{"points": 200}},
		)

		// Fetch again; should return cached result (Donor One still rank 1, since cache expires in 1 min)
		res, err := svc.GetLeaderboard(ctx, donorID1.Hex(), 10, 0)
		if err != nil {
			t.Fatalf("failed to get leaderboard: %v", err)
		}

		if res.Entries[0].DonorID != donorID1.Hex() {
			t.Errorf("expected cached rank 1 to still be %s, got %s", donorID1.Hex(), res.Entries[0].DonorID)
		}
	})
}
