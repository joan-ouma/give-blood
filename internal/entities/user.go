package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Role string

const (
	RoleAgency Role = "agency"
	RoleDonor  Role = "donor"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	Role         Role               `bson:"role" json:"role"`
	Name         string             `bson:"name" json:"name"`
	Lat          *float64           `bson:"lat,omitempty" json:"lat,omitempty"`
	Lng          *float64           `bson:"lng,omitempty" json:"lng,omitempty"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
}
