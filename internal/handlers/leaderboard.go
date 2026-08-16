package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/joan-ouma/give-blood/internal/entities"
	"github.com/joan-ouma/give-blood/internal/service"
)

type LeaderboardHandler struct {
	leaderboardService *service.LeaderboardService
	tokenService       *service.TokenService
}

func NewLeaderboardHandler(leaderboardService *service.LeaderboardService, tokenService *service.TokenService) *LeaderboardHandler {
	return &LeaderboardHandler{
		leaderboardService: leaderboardService,
		tokenService:       tokenService,
	}
}

func (h *LeaderboardHandler) GetEligibility(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, role, err := service.GetUserFromContext(r.Context())
	if err != nil || role != string(entities.RoleDonor) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	res, err := h.leaderboardService.GetEligibility(r.Context(), userID)
	if errors.Is(err, service.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (h *LeaderboardHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	donorIDParam := r.URL.Query().Get("donorId")

	var limit int64 = 20
	if parsed, err := strconv.ParseInt(limitStr, 10, 64); err == nil && parsed > 0 {
		limit = parsed
	}
	if limit > 100 {
		limit = 100
	}

	var offset int64 = 0
	if parsed, err := strconv.ParseInt(offsetStr, 10, 64); err == nil && parsed >= 0 {
		offset = parsed
	}

	var callingDonorID string
	authHeader := r.Header.Get("Authorization")
	var tokenStr string
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	}
	claims, err := h.tokenService.ParseToken(tokenStr)
	if err != nil || claims == nil {
		claims = make(map[string]interface{})
	}
	sub, okSub := claims["sub"].(string)
	role, okRole := claims["role"].(string)
	if okSub && okRole && role == string(entities.RoleDonor) {
		callingDonorID = sub
	}

	var targetDonorID string
	if donorIDParam != "" && donorIDParam == callingDonorID {
		targetDonorID = callingDonorID
	}

	res, err := h.leaderboardService.GetLeaderboard(r.Context(), targetDonorID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}
