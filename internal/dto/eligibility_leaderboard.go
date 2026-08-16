package dto

type EligibilityResponse struct {
	LastDonationAt *string `json:"lastDonationAt"`
	NextEligibleAt *string `json:"nextEligibleAt"`
	IsEligibleNow  bool    `json:"isEligibleNow"`
	DaysRemaining  int     `json:"daysRemaining"`
}

type LeaderboardEntry struct {
	Rank           int    `json:"rank"`
	DonorID        string `json:"donorId"`
	Name           string `json:"name"`
	Points         int    `json:"points"`
	TotalDonations int    `json:"totalDonations"`
	TotalPints     int    `json:"totalPints"`
}

type LeaderboardResponse struct {
	Entries []LeaderboardEntry `json:"entries"`
	Me      *LeaderboardEntry  `json:"me,omitempty"`
}
