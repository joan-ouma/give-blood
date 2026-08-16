package dto

type DonationCreateRequest struct {
	DriveID    *string `json:"driveId,omitempty"`
	LocationID *string `json:"locationId,omitempty"`
	Pints      *int    `json:"pints,omitempty"`
	DonatedAt  string  `json:"donatedAt,omitempty"`
}

type DonationVerifyRequest struct {
	Pints int `json:"pints"`
}

type DonationRejectRequest struct {
	RejectionReason string `json:"rejectionReason"`
}

type DonationResponse struct {
	ID              string  `json:"id"`
	DonorID         string  `json:"donorId"`
	DonorName       string  `json:"donorName,omitempty"`
	DonorEmail      string  `json:"donorEmail,omitempty"`
	AgencyID        string  `json:"agencyId"`
	DriveID         *string `json:"driveId,omitempty"`
	DriveTitle      string  `json:"driveTitle,omitempty"`
	LocationID      *string `json:"locationId,omitempty"`
	LocationName    string  `json:"locationName,omitempty"`
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
