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
	ErrDriveValidation = errors.New("validation failed")
)

type DriveService struct {
	col         *mongo.Collection
	locationCol *mongo.Collection
}

func NewDriveService(db *mongo.Database) *DriveService {
	return &DriveService{
		col:         db.Collection("drives"),
		locationCol: db.Collection("locations"),
	}
}

func (s *DriveService) EnsureIndexes(ctx context.Context) error {
	_, err := s.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "agencyId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "city", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "city", Value: 1},
				{Key: "endsAt", Value: 1},
			},
		},
	})
	return err
}

func (s *DriveService) Create(ctx context.Context, agencyIDStr string, req *dto.DriveCreateRequest) (*entities.Drive, error) {
	agencyID, err := primitive.ObjectIDFromHex(agencyIDStr)
	if err != nil {
		return nil, ErrForbidden
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.City) == "" {
		return nil, ErrDriveValidation
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return nil, ErrDriveValidation
	}
	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		return nil, ErrDriveValidation
	}

	if startsAt.After(endsAt) || startsAt.Equal(endsAt) {
		return nil, ErrDriveValidation
	}

	var locID *primitive.ObjectID
	var resolvedAddress string
	if req.LocationID != nil && *req.LocationID != "" {
		parsed, err := primitive.ObjectIDFromHex(*req.LocationID)
		if err != nil {
			return nil, ErrDriveValidation
		}
		locID = &parsed

		var loc entities.Location
		err = s.locationCol.FindOne(ctx, bson.M{"_id": *locID}).Decode(&loc)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrDriveValidation
			}
			return nil, err
		}
	} else {
		if strings.TrimSpace(req.Address) == "" {
			return nil, ErrDriveValidation
		}
		resolvedAddress = strings.TrimSpace(req.Address)
	}

	now := time.Now().UTC()
	drive := &entities.Drive{
		ID:         primitive.NewObjectID(),
		AgencyID:   agencyID,
		LocationID: locID,
		Title:      strings.TrimSpace(req.Title),
		Address:    resolvedAddress,
		City:       strings.TrimSpace(req.City),
		StartsAt:   startsAt.UTC(),
		EndsAt:     endsAt.UTC(),
		Notes:      strings.TrimSpace(req.Notes),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err = s.col.InsertOne(ctx, drive)
	if err != nil {
		return nil, err
	}

	return drive, nil
}

func (s *DriveService) Update(ctx context.Context, agencyIDStr string, driveIDStr string, req *dto.DriveUpdateRequest) (*entities.Drive, error) {
	agencyID, err := primitive.ObjectIDFromHex(agencyIDStr)
	if err != nil {
		return nil, ErrForbidden
	}

	driveID, err := primitive.ObjectIDFromHex(driveIDStr)
	if err != nil {
		return nil, ErrNotFound
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.City) == "" {
		return nil, ErrDriveValidation
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return nil, ErrDriveValidation
	}
	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		return nil, ErrDriveValidation
	}

	if startsAt.After(endsAt) || startsAt.Equal(endsAt) {
		return nil, ErrDriveValidation
	}

	var existing entities.Drive
	err = s.col.FindOne(ctx, bson.M{"_id": driveID}).Decode(&existing)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if existing.AgencyID != agencyID {
		return nil, ErrForbidden
	}

	var locID *primitive.ObjectID
	var resolvedAddress string
	if req.LocationID != nil && *req.LocationID != "" {
		parsed, err := primitive.ObjectIDFromHex(*req.LocationID)
		if err != nil {
			return nil, ErrDriveValidation
		}
		locID = &parsed

		var loc entities.Location
		err = s.locationCol.FindOne(ctx, bson.M{"_id": *locID}).Decode(&loc)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrDriveValidation
			}
			return nil, err
		}
	} else {
		if strings.TrimSpace(req.Address) == "" {
			return nil, ErrDriveValidation
		}
		resolvedAddress = strings.TrimSpace(req.Address)
	}

	update := bson.M{
		"$set": bson.M{
			"locationId": locID,
			"title":      strings.TrimSpace(req.Title),
			"address":    resolvedAddress,
			"city":       strings.TrimSpace(req.City),
			"startsAt":   startsAt.UTC(),
			"endsAt":     endsAt.UTC(),
			"notes":      strings.TrimSpace(req.Notes),
			"updatedAt":  time.Now().UTC(),
		},
	}

	var updated entities.Drive
	err = s.col.FindOneAndUpdate(ctx, bson.M{"_id": driveID}, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updated)
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

func (s *DriveService) Delete(ctx context.Context, agencyIDStr string, driveIDStr string) error {
	agencyID, err := primitive.ObjectIDFromHex(agencyIDStr)
	if err != nil {
		return ErrForbidden
	}

	driveID, err := primitive.ObjectIDFromHex(driveIDStr)
	if err != nil {
		return ErrNotFound
	}

	var existing entities.Drive
	err = s.col.FindOne(ctx, bson.M{"_id": driveID}).Decode(&existing)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrNotFound
		}
		return err
	}

	if existing.AgencyID != agencyID {
		return ErrForbidden
	}

	_, err = s.col.DeleteOne(ctx, bson.M{"_id": driveID})
	return err
}

func (s *DriveService) GetByID(ctx context.Context, idStr string) (*entities.Drive, error) {
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return nil, ErrNotFound
	}

	var drive entities.Drive
	err = s.col.FindOne(ctx, bson.M{"_id": id}).Decode(&drive)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &drive, nil
}

func (s *DriveService) List(ctx context.Context, city string, includePast bool, limit, offset int64) ([]entities.Drive, error) {
	filter := bson.M{}
	if city != "" {
		filter["city"] = city
	}
	if !includePast {
		filter["endsAt"] = bson.M{"$gte": time.Now().UTC()}
	}

	opts := options.Find().
		SetLimit(limit).
		SetSkip(offset).
		SetSort(bson.D{{Key: "startsAt", Value: 1}})

	cursor, err := s.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var drives []entities.Drive
	if err = cursor.All(ctx, &drives); err != nil {
		return nil, err
	}

	if drives == nil {
		drives = []entities.Drive{}
	}

	return drives, nil
}

func (s *DriveService) GetLocationsNamesMap(ctx context.Context, ids []primitive.ObjectID) (map[primitive.ObjectID]string, error) {
	if len(ids) == 0 {
		return map[primitive.ObjectID]string{}, nil
	}

	cursor, err := s.locationCol.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	names := make(map[primitive.ObjectID]string)
	for cursor.Next(ctx) {
		var loc entities.Location
		if err := cursor.Decode(&loc); err == nil {
			names[loc.ID] = loc.Name
		}
	}
	return names, nil
}
