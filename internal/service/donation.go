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
	userCol     *mongo.Collection
}

func NewDonationService(db *mongo.Database) *DonationService {
	return &DonationService{
		col:         db.Collection("donations"),
		statsCol:    db.Collection("donor_stats"),
		locationCol: db.Collection("locations"),
		driveCol:    db.Collection("drives"),
		userCol:     db.Collection("users"),
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

	// 1. Strict Eligibility Check (cooldown of 56 days)
	nowTime := time.Now().UTC()
	cooldownFilter := bson.M{
		"donorId":        donorID,
		"status":         entities.StatusVerified,
		"nextEligibleAt": bson.M{"$gt": nowTime},
	}
	cooldownCount, err := s.col.CountDocuments(ctx, cooldownFilter)
	if err == nil && cooldownCount > 0 {
		return nil, errors.New("donor is currently on a 56-day cooldown")
	}

	// 1b. Prevent multiple active RSVPs (pending or accepted)
	activeFilter := bson.M{
		"donorId": donorID,
		"status":  bson.M{"$in": []entities.DonationStatus{entities.StatusPending, entities.StatusAccepted}},
	}
	activeCount, err := s.col.CountDocuments(ctx, activeFilter)
	if err == nil && activeCount > 0 {
		return nil, errors.New("donor already has an active RSVP")
	}

	// 2. Parse/default donatedAt
	var donatedAt time.Time
	if req.DonatedAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.DonatedAt)
		if err != nil {
			return nil, ErrDonationValidation
		}
		if parsed.After(nowTime) {
			return nil, ErrDonationValidation
		}
		donatedAt = parsed.UTC()
	} else {
		donatedAt = nowTime
	}

	// 3. Pints default to 0 for RSVP
	pints := 0
	if req.Pints != nil {
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
		DonatedAt:  donatedAt,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err = s.col.InsertOne(ctx, donation)
	if err != nil {
		return nil, err
	}

	return donation, nil
}

func (s *DonationService) Accept(ctx context.Context, agencyIDStr string, donationIDStr string) (*entities.Donation, error) {
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
	filter := bson.M{
		"_id":    donationID,
		"status": entities.StatusPending,
	}

	update := bson.M{
		"$set": bson.M{
			"status":    entities.StatusAccepted,
			"updatedAt": now,
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

func (s *DonationService) Verify(ctx context.Context, agencyIDStr string, donationIDStr string, pints int) (*entities.Donation, error) {
	agencyID, err := primitive.ObjectIDFromHex(agencyIDStr)
	if err != nil {
		return nil, ErrForbidden
	}

	donationID, err := primitive.ObjectIDFromHex(donationIDStr)
	if err != nil {
		return nil, ErrNotFound
	}

	if pints < 1 || pints > 2 {
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

	if existing.Status != entities.StatusAccepted {
		return nil, ErrConflict
	}

	now := time.Now().UTC()
	nextEligible := now.Add(56 * 24 * time.Hour).UTC()

	filter := bson.M{
		"_id":    donationID,
		"status": entities.StatusAccepted,
	}

	update := bson.M{
		"$set": bson.M{
			"status":         entities.StatusVerified,
			"pints":          pints,
			"donatedAt":      now,
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

	// Update stats atomically
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

	if existing.Status != entities.StatusPending && existing.Status != entities.StatusAccepted {
		return nil, ErrConflict
	}

	now := time.Now().UTC()
	filter := bson.M{
		"_id":    donationID,
		"status": bson.M{"$in": []entities.DonationStatus{entities.StatusPending, entities.StatusAccepted}},
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
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

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

func (s *DonationService) ListPending(ctx context.Context, agencyIDStr string, status string, limit, offset int64) ([]entities.Donation, error) {
	agencyID, err := primitive.ObjectIDFromHex(agencyIDStr)
	if err != nil {
		return nil, ErrForbidden
	}

	opts := options.Find().
		SetLimit(limit).
		SetSkip(offset).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	if status == "" {
		status = "pending"
	}

	filter := bson.M{
		"agencyId": agencyID,
		"status":   status,
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

func (s *DonationService) EnrichList(ctx context.Context, list []entities.Donation) []dto.DonationResponse {
	responses := make([]dto.DonationResponse, len(list))

	userCache := make(map[primitive.ObjectID]entities.User)
	driveCache := make(map[primitive.ObjectID]entities.Drive)
	locCache := make(map[primitive.ObjectID]entities.Location)

	for i, d := range list {
		var driveIDStr *string
		if d.DriveID != nil {
			sID := d.DriveID.Hex()
			driveIDStr = &sID
		}
		var locIDStr *string
		if d.LocationID != nil {
			sID := d.LocationID.Hex()
			locIDStr = &sID
		}
		var verifiedAtStr *string
		if d.VerifiedAt != nil {
			sID := d.VerifiedAt.Format(time.RFC3339)
			verifiedAtStr = &sID
		}
		var verifiedByStr *string
		if d.VerifiedBy != nil {
			sID := d.VerifiedBy.Hex()
			verifiedByStr = &sID
		}
		var nextEligibleAtStr *string
		if d.NextEligibleAt != nil {
			sID := d.NextEligibleAt.Format(time.RFC3339)
			nextEligibleAtStr = &sID
		}

		res := dto.DonationResponse{
			ID:              d.ID.Hex(),
			DonorID:         d.DonorID.Hex(),
			AgencyID:        d.AgencyID.Hex(),
			DriveID:         driveIDStr,
			LocationID:      locIDStr,
			Pints:           d.Pints,
			Status:          string(d.Status),
			DonatedAt:       d.DonatedAt.Format(time.RFC3339),
			VerifiedAt:      verifiedAtStr,
			VerifiedBy:      verifiedByStr,
			RejectionReason: d.RejectionReason,
			NextEligibleAt:  nextEligibleAtStr,
			CreatedAt:       d.CreatedAt.Format(time.RFC3339),
			UpdatedAt:       d.UpdatedAt.Format(time.RFC3339),
		}

		// Enrich User
		var user entities.User
		if cached, ok := userCache[d.DonorID]; ok {
			user = cached
		} else {
			_ = s.userCol.FindOne(ctx, bson.M{"_id": d.DonorID}).Decode(&user)
			userCache[d.DonorID] = user
		}
		res.DonorName = user.Name
		res.DonorEmail = user.Email

		// Enrich Drive
		if d.DriveID != nil {
			var drive entities.Drive
			if cached, ok := driveCache[*d.DriveID]; ok {
				drive = cached
			} else {
				_ = s.driveCol.FindOne(ctx, bson.M{"_id": *d.DriveID}).Decode(&drive)
				driveCache[*d.DriveID] = drive
			}
			res.DriveTitle = drive.Title
		}

		// Enrich Location
		if d.LocationID != nil {
			var loc entities.Location
			if cached, ok := locCache[*d.LocationID]; ok {
				loc = cached
			} else {
				_ = s.locationCol.FindOne(ctx, bson.M{"_id": *d.LocationID}).Decode(&loc)
				locCache[*d.LocationID] = loc
			}
			res.LocationName = loc.Name
		}

		responses[i] = res
	}
	return responses
}
