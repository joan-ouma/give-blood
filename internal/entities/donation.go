package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DonationStatus string

const (
	StatusPending  DonationStatus = "pending"
	StatusAccepted DonationStatus = "accepted"
	StatusVerified DonationStatus = "verified"
	StatusRejected DonationStatus = "rejected"
)

type Donation struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	DonorID         primitive.ObjectID  `bson:"donorId" json:"donorId"`
	AgencyID        primitive.ObjectID  `bson:"agencyId" json:"agencyId"`
	DriveID         *primitive.ObjectID `bson:"driveId,omitempty" json:"driveId,omitempty"`
	LocationID      *primitive.ObjectID `bson:"locationId,omitempty" json:"locationId,omitempty"`
	Pints           int                 `bson:"pints" json:"pints"`
	Status          DonationStatus      `bson:"status" json:"status"`
	DonatedAt       time.Time           `bson:"donatedAt" json:"donatedAt"`
	VerifiedAt      *time.Time          `bson:"verifiedAt,omitempty" json:"verifiedAt,omitempty"`
	VerifiedBy      *primitive.ObjectID `bson:"verifiedBy,omitempty" json:"verifiedBy,omitempty"`
	RejectionReason *string             `bson:"rejectionReason,omitempty" json:"rejectionReason,omitempty"`
	NextEligibleAt  *time.Time          `bson:"nextEligibleAt,omitempty" json:"nextEligibleAt,omitempty"`
	CreatedAt       time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time           `bson:"updatedAt" json:"updatedAt"`
}

type DonorStats struct {
	ID             primitive.ObjectID `bson:"_id" json:"donorId"`
	TotalPints     int                `bson:"totalPints" json:"totalPints"`
	TotalDonations int                `bson:"totalDonations" json:"totalDonations"`
	Points         int                `bson:"points" json:"points"`
	UpdatedAt      time.Time          `bson:"updatedAt" json:"updatedAt"`
}
