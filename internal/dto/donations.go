package dto

type DonationCreateRequest struct {
	DriveID    *string `json:"driveId,omitempty"`
	LocationID *string `json:"locationId,omitempty"`
	Pints      *int    `json:"pints,omitempty"`
	DonatedAt  string  `json:"donatedAt"`
}

type DonationRejectRequest struct {
	RejectionReason string `json:"rejectionReason"`
}

type DonationResponse struct {
	ID              string  `json:"id"`
	DonorID         string  `json:"donorId"`
	AgencyID        string  `json:"agencyId"`
	DriveID         *string `json:"driveId,omitempty"`
	LocationID      *string `json:"locationId,omitempty"`
	Pints           int     `json:"pints"`
	Status          string  `json:"status"`
	DonatedAt       string  `json:"donatedAt"`
	VerifiedAt      *string `json:"verifiedAt,omitempty"`
	VerifiedBy      *string `json:"verifiedBy,omitempty"`
	RejectionReason *string `json:"rejectionReason,omitempty"`
	NextEligibleAt  *string `json:"nextEligibleAt,omitempty"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}
