package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	donatedAt, errDate := time.Parse(time.RFC3339, req.DonatedAt)
	if errDate != nil {
		fields["donatedAt"] = "Invalid RFC3339 date format"
	} else if donatedAt.After(time.Now().UTC()) {
		fields["donatedAt"] = "Donation date cannot be in the future"
	}

	if req.Pints != nil && (*req.Pints < 1 || *req.Pints > 2) {
		fields["pints"] = "Pints must be between 1 and 2"
	}

	if len(fields) > 0 {
		writeFieldsError(w, http.StatusBadRequest, "validation failed", fields)
		return
	}

	donation, err := h.donationService.Create(r.Context(), userID, &req)
	if errors.Is(err, service.ErrDonationValidation) {
		writeError(w, http.StatusBadRequest, "validation failed")
		return
	}
	if errors.Is(err, service.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(mapDonationToResponse(donation))
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

	responseList := make([]dto.DonationResponse, len(list))
	for i, donation := range list {
		responseList[i] = mapDonationToResponse(&donation)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responseList)
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

	list, err := h.donationService.ListPending(r.Context(), userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responseList := make([]dto.DonationResponse, len(list))
	for i, donation := range list {
		responseList[i] = mapDonationToResponse(&donation)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responseList)
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
	if len(parts) < 3 || parts[2] == "" {
		writeError(w, http.StatusBadRequest, "missing donation id")
		return
	}
	donationIDStr := parts[2]

	donation, err := h.donationService.Verify(r.Context(), userID, donationIDStr)
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mapDonationToResponse(donation))
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
	if len(parts) < 3 || parts[2] == "" {
		writeError(w, http.StatusBadRequest, "missing donation id")
		return
	}
	donationIDStr := parts[2]

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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mapDonationToResponse(donation))
}

func mapDonationToResponse(d *entities.Donation) dto.DonationResponse {
	var driveIDStr *string
	if d.DriveID != nil {
		s := d.DriveID.Hex()
		driveIDStr = &s
	}
	var locIDStr *string
	if d.LocationID != nil {
		s := d.LocationID.Hex()
		locIDStr = &s
	}
	var verifiedAtStr *string
	if d.VerifiedAt != nil {
		s := d.VerifiedAt.Format(time.RFC3339)
		verifiedAtStr = &s
	}
	var verifiedByStr *string
	if d.VerifiedBy != nil {
		s := d.VerifiedBy.Hex()
		verifiedByStr = &s
	}
	var nextEligibleAtStr *string
	if d.NextEligibleAt != nil {
		s := d.NextEligibleAt.Format(time.RFC3339)
		nextEligibleAtStr = &s
	}

	return dto.DonationResponse{
		ID:              d.ID.Hex(),
		DonorID:         d.DonorID.Hex(),
		AgencyID:        d.AgencyID.Hex(),
		DriveID:         driveIDStr,
		LocationID:      locIDStr,
		Pints:           d.Pints,
		Status:          string(d.Status),
		DonatedAt:       d.DonatedAt.Format(time.RFC3339),
		VerifiedAt:      verifiedAtStr,
		VerifiedBy:      verifiedByStr,
		RejectionReason: d.RejectionReason,
		NextEligibleAt:  nextEligibleAtStr,
		CreatedAt:       d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       d.UpdatedAt.Format(time.RFC3339),
	}
}
