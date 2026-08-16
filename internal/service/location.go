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
	ErrForbidden          = errors.New("forbidden")
	ErrLocationValidation = errors.New("validation failed")
)

type LocationService struct {
	col *mongo.Collection
}

func NewLocationService(db *mongo.Database) *LocationService {
	return &LocationService{
		col: db.Collection("locations"),
	}
}

func (s *LocationService) EnsureIndexes(ctx context.Context) error {
	_, err := s.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "agencyId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "city", Value: 1}},
		},
	})
	return err
}

func (s *LocationService) Create(ctx context.Context, agencyIDStr string, req *dto.LocationCreateRequest) (*entities.Location, error) {
	agencyID, err := primitive.ObjectIDFromHex(agencyIDStr)
	if err != nil {
		return nil, ErrForbidden
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.City) == "" {
		return nil, ErrLocationValidation
	}

	if req.Lat != nil && (*req.Lat < -90.0 || *req.Lat > 90.0) {
		return nil, ErrLocationValidation
	}
	if req.Lng != nil && (*req.Lng < -180.0 || *req.Lng > 180.0) {
		return nil, ErrLocationValidation
	}

	now := time.Now().UTC()
	loc := &entities.Location{
		ID:        primitive.NewObjectID(),
		AgencyID:  agencyID,
		Name:      strings.TrimSpace(req.Name),
		Address:   strings.TrimSpace(req.Address),
		City:      strings.TrimSpace(req.City),
		Lat:       req.Lat,
		Lng:       req.Lng,
		Hours:     strings.TrimSpace(req.Hours),
		Phone:     strings.TrimSpace(req.Phone),
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = s.col.InsertOne(ctx, loc)
	if err != nil {
		return nil, err
	}

	return loc, nil
}

func (s *LocationService) Update(ctx context.Context, agencyIDStr string, locationIDStr string, req *dto.LocationUpdateRequest) (*entities.Location, error) {
	agencyID, err := primitive.ObjectIDFromHex(agencyIDStr)
	if err != nil {
		return nil, ErrForbidden
	}

	locID, err := primitive.ObjectIDFromHex(locationIDStr)
	if err != nil {
		return nil, ErrNotFound
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.City) == "" {
		return nil, ErrLocationValidation
	}

	if req.Lat != nil && (*req.Lat < -90.0 || *req.Lat > 90.0) {
		return nil, ErrLocationValidation
	}
	if req.Lng != nil && (*req.Lng < -180.0 || *req.Lng > 180.0) {
		return nil, ErrLocationValidation
	}

	var existing entities.Location
	err = s.col.FindOne(ctx, bson.M{"_id": locID}).Decode(&existing)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if existing.AgencyID != agencyID {
		return nil, ErrForbidden
	}

	update := bson.M{
		"$set": bson.M{
			"name":      strings.TrimSpace(req.Name),
			"address":   strings.TrimSpace(req.Address),
			"city":      strings.TrimSpace(req.City),
			"lat":       req.Lat,
			"lng":       req.Lng,
			"hours":     strings.TrimSpace(req.Hours),
			"phone":     strings.TrimSpace(req.Phone),
			"updatedAt": time.Now().UTC(),
		},
	}

	var updated entities.Location
	err = s.col.FindOneAndUpdate(ctx, bson.M{"_id": locID}, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updated)
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

func (s *LocationService) Delete(ctx context.Context, agencyIDStr string, locationIDStr string) error {
	agencyID, err := primitive.ObjectIDFromHex(agencyIDStr)
	if err != nil {
		return ErrForbidden
	}

	locID, err := primitive.ObjectIDFromHex(locationIDStr)
	if err != nil {
		return ErrNotFound
	}

	var existing entities.Location
	err = s.col.FindOne(ctx, bson.M{"_id": locID}).Decode(&existing)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrNotFound
		}
		return err
	}

	if existing.AgencyID != agencyID {
		return ErrForbidden
	}

	_, err = s.col.DeleteOne(ctx, bson.M{"_id": locID})
	return err
}

func (s *LocationService) GetByID(ctx context.Context, idStr string) (*entities.Location, error) {
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return nil, ErrNotFound
	}

	var loc entities.Location
	err = s.col.FindOne(ctx, bson.M{"_id": id}).Decode(&loc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &loc, nil
}

func (s *LocationService) List(ctx context.Context, city string, limit, offset int64) ([]entities.Location, error) {
	filter := bson.M{}
	if city != "" {
		filter["city"] = city
	}

	opts := options.Find().
		SetLimit(limit).
		SetSkip(offset).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := s.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var locations []entities.Location
	if err = cursor.All(ctx, &locations); err != nil {
		return nil, err
	}

	if locations == nil {
		locations = []entities.Location{}
	}

	return locations, nil
}
