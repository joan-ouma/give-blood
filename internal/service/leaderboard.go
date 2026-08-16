package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/joan-ouma/give-blood/internal/dto"
	"github.com/joan-ouma/give-blood/internal/entities"
)

type cacheEntry struct {
	entries   []dto.LeaderboardEntry
	expiresAt time.Time
}

type LeaderboardService struct {
	statsCol    *mongo.Collection
	donationCol *mongo.Collection
	userCol     *mongo.Collection
	cacheMu     sync.RWMutex
	cacheMap    map[string]cacheEntry
}

func NewLeaderboardService(db *mongo.Database) *LeaderboardService {
	return &LeaderboardService{
		statsCol:    db.Collection("donor_stats"),
		donationCol: db.Collection("donations"),
		userCol:     db.Collection("users"),
		cacheMap:    make(map[string]cacheEntry),
	}
}

func (s *LeaderboardService) EnsureIndexes(ctx context.Context) error {
	_, err := s.statsCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "points", Value: -1}},
	})
	return err
}

func (s *LeaderboardService) GetEligibility(ctx context.Context, donorIDStr string) (*dto.EligibilityResponse, error) {
	donorID, err := primitive.ObjectIDFromHex(donorIDStr)
	if err != nil {
		return nil, ErrForbidden
	}

	filter := bson.M{
		"donorId": donorID,
		"status":  entities.StatusVerified,
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "donatedAt", Value: -1}})

	var lastDonation entities.Donation
	err = s.donationCol.FindOne(ctx, filter, opts).Decode(&lastDonation)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return &dto.EligibilityResponse{
				LastDonationAt: nil,
				NextEligibleAt: nil,
				IsEligibleNow:  true,
				DaysRemaining:  0,
			}, nil
		}
		return nil, err
	}

	now := time.Now().UTC()
	var lastStr, nextStr *string
	lTime := lastDonation.DonatedAt.Format(time.RFC3339)
	lastStr = &lTime

	var nextEligible time.Time
	if lastDonation.NextEligibleAt != nil {
		nextEligible = *lastDonation.NextEligibleAt
	} else {
		nextEligible = lastDonation.DonatedAt.Add(56 * 24 * time.Hour).UTC()
	}

	nTime := nextEligible.Format(time.RFC3339)
	nextStr = &nTime

	isEligibleNow := !now.Before(nextEligible)
	daysRemaining := 0
	if !isEligibleNow {
		daysRemaining = int(nextEligible.Sub(now).Hours() / 24)
		if daysRemaining <= 0 {
			daysRemaining = 1
		}
	}

	return &dto.EligibilityResponse{
		LastDonationAt: lastStr,
		NextEligibleAt: nextStr,
		IsEligibleNow:  isEligibleNow,
		DaysRemaining:  daysRemaining,
	}, nil
}

type AggregatedLeaderboardEntry struct {
	ID             primitive.ObjectID `bson:"_id"`
	Points         int                `bson:"points"`
	TotalDonations int                `bson:"totalDonations"`
	TotalPints     int                `bson:"totalPints"`
	User           *struct {
		Name string `bson:"name"`
	} `bson:"user"`
}

func (s *LeaderboardService) getCachedPage(limit, offset int64) ([]dto.LeaderboardEntry, bool) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	key := fmt.Sprintf("%d_%d", limit, offset)
	if entry, ok := s.cacheMap[key]; ok {
		if time.Now().Before(entry.expiresAt) {
			return entry.entries, true
		}
	}
	return nil, false
}

func (s *LeaderboardService) setCachedPage(limit, offset int64, entries []dto.LeaderboardEntry) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	key := fmt.Sprintf("%d_%d", limit, offset)
	s.cacheMap[key] = cacheEntry{
		entries:   entries,
		expiresAt: time.Now().Add(30 * time.Second),
	}
}

func (s *LeaderboardService) getDonorRank(ctx context.Context, donorID primitive.ObjectID) (int, *entities.DonorStats, error) {
	var stats entities.DonorStats
	err := s.statsCol.FindOne(ctx, bson.M{"_id": donorID}).Decode(&stats)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil, nil
		}
		return 0, nil, err
	}

	filter := bson.M{
		"$or": []bson.M{
			{"points": bson.M{"$gt": stats.Points}},
			{
				"points": stats.Points,
				"_id":    bson.M{"$lt": stats.ID},
			},
		},
	}
	count, err := s.statsCol.CountDocuments(ctx, filter)
	if err != nil {
		return 0, nil, err
	}

	return int(count) + 1, &stats, nil
}

func (s *LeaderboardService) GetLeaderboard(ctx context.Context, currentDonorIDStr string, limit, offset int64) (*dto.LeaderboardResponse, error) {
	// 1. Check cache first
	entries, found := s.getCachedPage(limit, offset)
	if !found {
		// 2. Run $lookup aggregation
		pipeline := mongo.Pipeline{
			{{Key: "$sort", Value: bson.D{{Key: "points", Value: -1}, {Key: "_id", Value: 1}}}},
			{{Key: "$skip", Value: offset}},
			{{Key: "$limit", Value: limit}},
			{{Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "users"},
				{Key: "localField", Value: "_id"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "user"},
			}}},
			{{Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$user"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			}}},
		}

		cursor, err := s.statsCol.Aggregate(ctx, pipeline)
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)

		entries = []dto.LeaderboardEntry{}
		for cursor.Next(ctx) {
			var doc AggregatedLeaderboardEntry
			if err := cursor.Decode(&doc); err != nil {
				return nil, err
			}
			name := ""
			if doc.User != nil {
				name = doc.User.Name
			}
			entries = append(entries, dto.LeaderboardEntry{
				Rank:           int(offset) + len(entries) + 1,
				DonorID:        doc.ID.Hex(),
				Name:           name,
				Points:         doc.Points,
				TotalDonations: doc.TotalDonations,
				TotalPints:     doc.TotalPints,
			})
		}
		s.setCachedPage(limit, offset, entries)
	}

	// 3. Handle ?donorId matching calling donor
	var me *dto.LeaderboardEntry
	if currentDonorIDStr != "" {
		meID, err := primitive.ObjectIDFromHex(currentDonorIDStr)
		if err == nil {
			rank, stats, err := s.getDonorRank(ctx, meID)
			if err == nil && stats != nil {
				var u entities.User
				var name string
				if err := s.userCol.FindOne(ctx, bson.M{"_id": meID}).Decode(&u); err == nil {
					name = u.Name
				}
				me = &dto.LeaderboardEntry{
					Rank:           rank,
					DonorID:        stats.ID.Hex(),
					Name:           name,
					Points:         stats.Points,
					TotalDonations: stats.TotalDonations,
					TotalPints:     stats.TotalPints,
				}
			}
		}
	}

	return &dto.LeaderboardResponse{
		Entries: entries,
		Me:      me,
	}, nil
}
