package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Location struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	AgencyID  primitive.ObjectID `bson:"agencyId" json:"agencyId"`
	Name      string             `bson:"name" json:"name"`
	Address   string             `bson:"address" json:"address"`
	City      string             `bson:"city" json:"city"`
	Lat       *float64           `bson:"lat,omitempty" json:"lat,omitempty"`
	Lng       *float64           `bson:"lng,omitempty" json:"lng,omitempty"`
	Hours     string             `bson:"hours" json:"hours"`
	Phone     string             `bson:"phone" json:"phone"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}
