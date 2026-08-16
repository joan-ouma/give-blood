package dto

type RegisterRequest struct {
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Role     string   `json:"role"`
	Name     string   `json:"name"`
	Lat      *float64 `json:"lat,omitempty"`
	Lng      *float64 `json:"lng,omitempty"`
	Long     *float64 `json:"long,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ErrorFieldResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}
