package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/joan-ouma/give-blood/internal/dto"
	"github.com/joan-ouma/give-blood/internal/entities"
)

var (
	ErrDonationValidation = errors.New("validation failed")
	ErrConflict           = errors.New("conflict status change")
)

type DonationService struct {
	col         *mongo.Collection
	statsCol    *mongo.Collection
	locationCol *mongo.Collection
	driveCol    *mongo.Collection
}

func NewDonationService(db *mongo.Database) *DonationService {
	return &DonationService{
		col:         db.Collection("donations"),
		statsCol:    db.Collection("donor_stats"),
		locationCol: db.Collection("locations"),
		driveCol:    db.Collection("drives"),
	}
}

func (s *DonationService) EnsureIndexes(ctx context.Context) error {
	_, err := s.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "agencyId", Value: 1},
				{Key: "status", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "donorId", Value: 1},
				{Key: "status", Value: 1},
			},
		},
	})
	return err
}

func (s *DonationService) Create(ctx context.Context, donorIDStr string, req *dto.DonationCreateRequest) (*entities.Donation, error) {
	donorID, err := primitive.ObjectIDFromHex(donorIDStr)
	if err != nil {
		return nil, ErrForbidden
	}

	donatedAt, err := time.Parse(time.RFC3339, req.DonatedAt)
	if err != nil {
		return nil, ErrDonationValidation
	}
	if donatedAt.After(time.Now().UTC()) {
		return nil, ErrDonationValidation
	}

	pints := 1
	if req.Pints != nil {
		if *req.Pints < 1 || *req.Pints > 2 {
			return nil, ErrDonationValidation
		}
		pints = *req.Pints
	}

	var driveID *primitive.ObjectID
	var locationID *primitive.ObjectID
	var agencyID primitive.ObjectID

	if req.DriveID != nil && *req.DriveID != "" {
		parsed, err := primitive.ObjectIDFromHex(*req.DriveID)
		if err != nil {
			return nil, ErrDonationValidation
		}
		driveID = &parsed

		var drive entities.Drive
		err = s.driveCol.FindOne(ctx, bson.M{"_id": *driveID}).Decode(&drive)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrDonationValidation
			}
			return nil, err
		}
		agencyID = drive.AgencyID

		if drive.LocationID != nil {
			locationID = drive.LocationID
		} else if req.LocationID != nil && *req.LocationID != "" {
			locParsed, err := primitive.ObjectIDFromHex(*req.LocationID)
			if err != nil {
				return nil, ErrDonationValidation
			}
			locationID = &locParsed
		}
	} else if req.LocationID != nil && *req.LocationID != "" {
		parsed, err := primitive.ObjectIDFromHex(*req.LocationID)
		if err != nil {
			return nil, ErrDonationValidation
		}
		locationID = &parsed

		var loc entities.Location
		err = s.locationCol.FindOne(ctx, bson.M{"_id": *locationID}).Decode(&loc)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrDonationValidation
			}
			return nil, err
		}
		agencyID = loc.AgencyID
	} else {
		return nil, ErrDonationValidation
	}

	now := time.Now().UTC()
	donation := &entities.Donation{
		ID:         primitive.NewObjectID(),
		DonorID:    donorID,
		AgencyID:   agencyID,
		DriveID:    driveID,
		LocationID: locationID,
		Pints:      pints,
		Status:     entities.StatusPending,
		DonatedAt:  donatedAt.UTC(),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err = s.col.InsertOne(ctx, donation)
	if err != nil {
		return nil, err
	}

	return donation, nil
}

func (s *DonationService) Verify(ctx context.Context, agencyIDStr string, donationIDStr string) (*entities.Donation, error) {
	agencyID, err := primitive.ObjectIDFromHex(agencyIDStr)
	if err != nil {
		return nil, ErrForbidden
	}

	donationID, err := primitive.ObjectIDFromHex(donationIDStr)
	if err != nil {
		return nil, ErrNotFound
	}

	var existing entities.Donation
	err = s.col.FindOne(ctx, bson.M{"_id": donationID}).Decode(&existing)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if existing.AgencyID != agencyID {
		return nil, ErrForbidden
	}

	if existing.Status != entities.StatusPending {
		return nil, ErrConflict
	}

	now := time.Now().UTC()
	nextEligible := existing.DonatedAt.Add(56 * 24 * time.Hour).UTC()

	filter := bson.M{
		"_id":    donationID,
		"status": entities.StatusPending,
	}

	update := bson.M{
		"$set": bson.M{
			"status":         entities.StatusVerified,
			"verifiedAt":     now,
			"verifiedBy":     agencyID,
			"nextEligibleAt": nextEligible,
			"updatedAt":      now,
		},
	}

	var updated entities.Donation
	err = s.col.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updated)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrConflict
		}
		return nil, err
	}

	// Update stats atomically via $inc: totalDonations*10 + totalPints*5 => increment donations by 1 (10 pts), pints by pints (pints*5 pts)
	statsFilter := bson.M{"_id": updated.DonorID}
	statsUpdate := bson.M{
		"$inc": bson.M{
			"totalDonations": 1,
			"totalPints":     updated.Pints,
			"points":         10 + (updated.Pints * 5),
		},
		"$set": bson.M{
			"updatedAt": now,
		},
	}
	_, err = s.statsCol.UpdateOne(ctx, statsFilter, statsUpdate, options.Update().SetUpsert(true))
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

func (s *DonationService) Reject(ctx context.Context, agencyIDStr string, donationIDStr string, req *dto.DonationRejectRequest) (*entities.Donation, error) {
	agencyID, err := primitive.ObjectIDFromHex(agencyIDStr)
	if err != nil {
		return nil, ErrForbidden
	}

	donationID, err := primitive.ObjectIDFromHex(donationIDStr)
	if err != nil {
		return nil, ErrNotFound
	}

	if strings.TrimSpace(req.RejectionReason) == "" {
		return nil, ErrDonationValidation
	}

	var existing entities.Donation
	err = s.col.FindOne(ctx, bson.M{"_id": donationID}).Decode(&existing)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if existing.AgencyID != agencyID {
		return nil, ErrForbidden
	}

	if existing.Status != entities.StatusPending {
		return nil, ErrConflict
	}

	now := time.Now().UTC()
	filter := bson.M{
		"_id":    donationID,
		"status": entities.StatusPending,
	}

	reason := strings.TrimSpace(req.RejectionReason)
	update := bson.M{
		"$set": bson.M{
			"status":          entities.StatusRejected,
			"rejectionReason": reason,
			"updatedAt":       now,
		},
	}

	var updated entities.Donation
	err = s.col.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updated)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrConflict
		}
		return nil, err
	}

	return &updated, nil
}

func (s *DonationService) ListMine(ctx context.Context, donorIDStr string, limit, offset int64) ([]entities.Donation, error) {
	donorID, err := primitive.ObjectIDFromHex(donorIDStr)
	if err != nil {
		return nil, ErrForbidden
	}

	opts := options.Find().
		SetLimit(limit).
		SetSkip(offset).
		SetSort(bson.D{{Key: "donatedAt", Value: -1}})

	cursor, err := s.col.Find(ctx, bson.M{"donorId": donorID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []entities.Donation
	if err = cursor.All(ctx, &list); err != nil {
		return nil, err
	}

	if list == nil {
		list = []entities.Donation{}
	}
	return list, nil
}

func (s *DonationService) ListPending(ctx context.Context, agencyIDStr string, limit, offset int64) ([]entities.Donation, error) {
	agencyID, err := primitive.ObjectIDFromHex(agencyIDStr)
	if err != nil {
		return nil, ErrForbidden
	}

	opts := options.Find().
		SetLimit(limit).
		SetSkip(offset).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	filter := bson.M{
		"agencyId": agencyID,
		"status":   entities.StatusPending,
	}

	cursor, err := s.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []entities.Donation
	if err = cursor.All(ctx, &list); err != nil {
		return nil, err
	}

	if list == nil {
		list = []entities.Donation{}
	}
	return list, nil
}
