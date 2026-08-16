package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Drive struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	AgencyID   primitive.ObjectID  `bson:"agencyId" json:"agencyId"`
	LocationID *primitive.ObjectID `bson:"locationId,omitempty" json:"locationId,omitempty"`
	Title      string              `bson:"title" json:"title"`
	Address    string              `bson:"address,omitempty" json:"address,omitempty"`
	City       string              `bson:"city" json:"city"`
	StartsAt   time.Time           `bson:"startsAt" json:"startsAt"`
	EndsAt     time.Time           `bson:"endsAt" json:"endsAt"`
	Notes      string              `bson:"notes" json:"notes"`
	CreatedAt  time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time           `bson:"updatedAt" json:"updatedAt"`
}
