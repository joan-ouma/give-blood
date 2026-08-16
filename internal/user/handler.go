package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/joan-ouma/give-blood/internal/auth"
)

type Handler struct {
	service      *Service
	tokenService *auth.TokenService
	limiter      *auth.RateLimiter
}

func NewHandler(service *Service, tokenService *auth.TokenService, limiter *auth.RateLimiter) *Handler {
	return &Handler{
		service:      service,
		tokenService: tokenService,
		limiter:      limiter,
	}
}

type errorFieldResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorFieldResponse{Error: msg})
}

func writeFieldsError(w http.ResponseWriter, status int, msg string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorFieldResponse{Error: msg, Fields: fields})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fields := make(map[string]string)
	if req.Email == "" {
		fields["email"] = "Email is required"
	}
	if len(req.Password) < 8 {
		fields["password"] = "Password must be at least 8 characters"
	}
	if req.Role != "agency" && req.Role != "donor" {
		fields["role"] = "Role must be agency or donor"
	}
	if req.Name == "" {
		fields["name"] = "Name is required"
	}

	if len(fields) > 0 {
		writeFieldsError(w, http.StatusBadRequest, "validation failed", fields)
		return
	}

	u, err := h.service.Register(r.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			writeError(w, http.StatusBadRequest, "email already registered")
			return
		}
		if errors.Is(err, ErrInvalidEmail) || errors.Is(err, ErrInvalidPassword) || errors.Is(err, ErrInvalidRole) || errors.Is(err, ErrInvalidName) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(u)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !h.limiter.Allow(r, req.Email) {
		writeError(w, http.StatusTooManyRequests, "too many attempts, try again later")
		return
	}

	u, err := h.service.Authenticate(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid email or password")
		return
	}

	accessToken, err := h.tokenService.GenerateAccessToken(u.ID.Hex(), string(u.Role))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	refreshToken, err := h.tokenService.GenerateRefreshToken(u.ID.Hex(), string(u.Role))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    refreshToken,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"accessToken": accessToken,
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	cookie, err := r.Cookie("refreshToken")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	claims, err := h.tokenService.ParseToken(cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	sub, okSub := claims["sub"].(string)
	role, okRole := claims["role"].(string)
	if !okSub || !okRole {
		writeError(w, http.StatusUnauthorized, "invalid token claims")
		return
	}

	accessToken, err := h.tokenService.GenerateAccessToken(sub, role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"accessToken": accessToken,
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, _, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	u, err := h.service.GetByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(u)
}
