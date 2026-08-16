package dto

type LocationCreateRequest struct {
	Name    string   `json:"name"`
	Address string   `json:"address"`
	City    string   `json:"city"`
	Lat     *float64 `json:"lat,omitempty"`
	Lng     *float64 `json:"lng,omitempty"`
	Hours   string   `json:"hours"`
	Phone   string   `json:"phone"`
}

type LocationUpdateRequest struct {
	Name    string   `json:"name"`
	Address string   `json:"address"`
	City    string   `json:"city"`
	Lat     *float64 `json:"lat,omitempty"`
	Lng     *float64 `json:"lng,omitempty"`
	Hours   string   `json:"hours"`
	Phone   string   `json:"phone"`
}

type DriveCreateRequest struct {
	LocationID *string `json:"locationId,omitempty"`
	Title      string  `json:"title"`
	Address    string  `json:"address,omitempty"`
	City       string  `json:"city"`
	StartsAt   string  `json:"startsAt"`
	EndsAt     string  `json:"endsAt"`
	Notes      string  `json:"notes"`
}

type DriveUpdateRequest struct {
	LocationID *string `json:"locationId,omitempty"`
	Title      string  `json:"title"`
	Address    string  `json:"address,omitempty"`
	City       string  `json:"city"`
	StartsAt   string  `json:"startsAt"`
	EndsAt     string  `json:"endsAt"`
	Notes      string  `json:"notes"`
}

type LocationResponse struct {
	ID        string   `json:"id"`
	AgencyID  string   `json:"agencyId"`
	Name      string   `json:"name"`
	Address   string   `json:"address"`
	City      string   `json:"city"`
	Lat       *float64 `json:"lat,omitempty"`
	Lng       *float64 `json:"lng,omitempty"`
	Hours     string   `json:"hours"`
	Phone     string   `json:"phone"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

type DriveResponse struct {
	ID           string  `json:"id"`
	AgencyID     string  `json:"agencyId"`
	LocationID   *string `json:"locationId,omitempty"`
	Title        string  `json:"title"`
	Address      string  `json:"address,omitempty"`
	City         string  `json:"city"`
	StartsAt     string  `json:"startsAt"`
	EndsAt       string  `json:"endsAt"`
	Notes        string  `json:"notes"`
	LocationName string  `json:"locationName,omitempty"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}
