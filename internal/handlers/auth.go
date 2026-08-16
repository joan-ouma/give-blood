package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/joan-ouma/give-blood/internal/dto"
	"github.com/joan-ouma/give-blood/internal/service"
)

type AuthHandler struct {
	userService  *service.UserService
	tokenService *service.TokenService
	limiter      *service.RateLimiter
}

func NewAuthHandler(userService *service.UserService, tokenService *service.TokenService, limiter *service.RateLimiter) *AuthHandler {
	return &AuthHandler{
		userService:  userService,
		tokenService: tokenService,
		limiter:      limiter,
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(dto.ErrorFieldResponse{Error: msg})
}

func writeFieldsError(w http.ResponseWriter, status int, msg string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(dto.ErrorFieldResponse{Error: msg, Fields: fields})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req dto.RegisterRequest
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
	if req.Role == "agency" {
		if req.Lat == nil {
			fields["lat"] = "Latitude is required"
		}
		if req.Lng == nil && req.Long == nil {
			fields["long"] = "Longitude is required"
		}
	}

	if len(fields) > 0 {
		writeFieldsError(w, http.StatusBadRequest, "validation failed", fields)
		return
	}

	u, err := h.userService.Register(r.Context(), &req)
	if errors.Is(err, service.ErrAlreadyExists) {
		writeError(w, http.StatusBadRequest, "email already registered")
		return
	}
	if errors.Is(err, service.ErrInvalidEmail) || errors.Is(err, service.ErrInvalidPass) || errors.Is(err, service.ErrInvalidRole) || errors.Is(err, service.ErrInvalidName) || errors.Is(err, service.ErrInvalidCoords) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(u)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !h.limiter.Allow(r, req.Email) {
		writeError(w, http.StatusTooManyRequests, "too many attempts, try again later")
		return
	}

	u, err := h.userService.Authenticate(r.Context(), &req)
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

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
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

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, _, err := service.GetUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	u, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(u)
}
