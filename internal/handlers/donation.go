package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/joan-ouma/give-blood/internal/dto"
	"github.com/joan-ouma/give-blood/internal/entities"
	"github.com/joan-ouma/give-blood/internal/service"
)

type DonationHandler struct {
	donationService *service.DonationService
	limiter         *service.RateLimiter
}

func NewDonationHandler(donationService *service.DonationService, limiter *service.RateLimiter) *DonationHandler {
	return &DonationHandler{
		donationService: donationService,
		limiter:         limiter,
	}
}

func (h *DonationHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, role, err := service.GetUserFromContext(r.Context())
	if err != nil || role != string(entities.RoleDonor) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req dto.DonationCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !h.limiter.Allow(r, userID) {
		writeError(w, http.StatusTooManyRequests, "too many attempts, try again later")
		return
	}

	fields := make(map[string]string)
	if req.DriveID == nil && req.LocationID == nil {
		fields["driveId"] = "Either DriveID or LocationID is required"
	}

	if req.Pints != nil && (*req.Pints < 0 || *req.Pints > 2) {
		fields["pints"] = "Pints must be between 0 and 2"
	}

	if len(fields) > 0 {
		writeFieldsError(w, http.StatusBadRequest, "validation failed", fields)
		return
	}

	donation, err := h.donationService.Create(r.Context(), userID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "cooldown") {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, service.ErrDonationValidation) {
			writeError(w, http.StatusBadRequest, "validation failed")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	enriched := h.donationService.EnrichList(r.Context(), []entities.Donation{*donation})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(enriched[0])
}

func (h *DonationHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, role, err := service.GetUserFromContext(r.Context())
	if err != nil || role != string(entities.RoleDonor) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

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

	list, err := h.donationService.ListMine(r.Context(), userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	enriched := h.donationService.EnrichList(r.Context(), list)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(enriched)
}

func (h *DonationHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, role, err := service.GetUserFromContext(r.Context())
	if err != nil || role != string(entities.RoleAgency) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	status := r.URL.Query().Get("status")

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

	list, err := h.donationService.ListPending(r.Context(), userID, status, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	enriched := h.donationService.EnrichList(r.Context(), list)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(enriched)
}

func (h *DonationHandler) Accept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, role, err := service.GetUserFromContext(r.Context())
	if err != nil || role != string(entities.RoleAgency) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		writeError(w, http.StatusBadRequest, "missing donation id")
		return
	}
	donationIDStr := parts[3]

	donation, err := h.donationService.Accept(r.Context(), userID, donationIDStr)
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "donation not found")
		return
	}
	if errors.Is(err, service.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if errors.Is(err, service.ErrConflict) {
		writeError(w, http.StatusConflict, "donation already processed")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	enriched := h.donationService.EnrichList(r.Context(), []entities.Donation{*donation})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(enriched[0])
}

func (h *DonationHandler) Verify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, role, err := service.GetUserFromContext(r.Context())
	if err != nil || role != string(entities.RoleAgency) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		writeError(w, http.StatusBadRequest, "missing donation id")
		return
	}
	donationIDStr := parts[3]

	var req dto.DonationVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Pints < 1 || req.Pints > 2 {
		writeError(w, http.StatusBadRequest, "pints must be between 1 and 2")
		return
	}

	donation, err := h.donationService.Verify(r.Context(), userID, donationIDStr, req.Pints)
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "donation not found")
		return
	}
	if errors.Is(err, service.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if errors.Is(err, service.ErrConflict) {
		writeError(w, http.StatusConflict, "donation already processed")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	enriched := h.donationService.EnrichList(r.Context(), []entities.Donation{*donation})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(enriched[0])
}

func (h *DonationHandler) Reject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, role, err := service.GetUserFromContext(r.Context())
	if err != nil || role != string(entities.RoleAgency) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		writeError(w, http.StatusBadRequest, "missing donation id")
		return
	}
	donationIDStr := parts[3]

	var req dto.DonationRejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.RejectionReason) == "" {
		writeFieldsError(w, http.StatusBadRequest, "validation failed", map[string]string{
			"rejectionReason": "Rejection reason must be provided",
		})
		return
	}

	donation, err := h.donationService.Reject(r.Context(), userID, donationIDStr, &req)
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "donation not found")
		return
	}
	if errors.Is(err, service.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if errors.Is(err, service.ErrConflict) {
		writeError(w, http.StatusConflict, "donation already processed")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	enriched := h.donationService.EnrichList(r.Context(), []entities.Donation{*donation})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(enriched[0])
}
