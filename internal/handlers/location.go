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

type LocationHandler struct {
	locService *service.LocationService
}

func NewLocationHandler(locService *service.LocationService) *LocationHandler {
	return &LocationHandler{locService: locService}
}

func (h *LocationHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, role, err := service.GetUserFromContext(r.Context())
	if err != nil || role != string(entities.RoleAgency) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req dto.LocationCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fields := make(map[string]string)
	if strings.TrimSpace(req.Name) == "" {
		fields["name"] = "Name is required"
	}
	if strings.TrimSpace(req.City) == "" {
		fields["city"] = "City is required"
	}
	if req.Lat != nil && (*req.Lat < -90.0 || *req.Lat > 90.0) {
		fields["lat"] = "Latitude must be between -90 and 90"
	}
	if req.Lng != nil && (*req.Lng < -180.0 || *req.Lng > 180.0) {
		fields["lng"] = "Longitude must be between -180 and 180"
	}

	if len(fields) > 0 {
		writeFieldsError(w, http.StatusBadRequest, "validation failed", fields)
		return
	}

	loc, err := h.locService.Create(r.Context(), userID, &req)
	if errors.Is(err, service.ErrLocationValidation) {
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
	_ = json.NewEncoder(w).Encode(mapLocationToResponse(loc))
}

func (h *LocationHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
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
		writeError(w, http.StatusBadRequest, "missing location id")
		return
	}
	locationIDStr := parts[3]

	var req dto.LocationUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fields := make(map[string]string)
	if strings.TrimSpace(req.Name) == "" {
		fields["name"] = "Name is required"
	}
	if strings.TrimSpace(req.City) == "" {
		fields["city"] = "City is required"
	}
	if req.Lat != nil && (*req.Lat < -90.0 || *req.Lat > 90.0) {
		fields["lat"] = "Latitude must be between -90 and 90"
	}
	if req.Lng != nil && (*req.Lng < -180.0 || *req.Lng > 180.0) {
		fields["lng"] = "Longitude must be between -180 and 180"
	}

	if len(fields) > 0 {
		writeFieldsError(w, http.StatusBadRequest, "validation failed", fields)
		return
	}

	loc, err := h.locService.Update(r.Context(), userID, locationIDStr, &req)
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}
	if errors.Is(err, service.ErrForbidden) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if errors.Is(err, service.ErrLocationValidation) {
		writeError(w, http.StatusBadRequest, "validation failed")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mapLocationToResponse(loc))
}

func (h *LocationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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
		writeError(w, http.StatusBadRequest, "missing location id")
		return
	}
	locationIDStr := parts[3]

	err = h.locService.Delete(r.Context(), userID, locationIDStr)
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "location not found")
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

	w.WriteHeader(http.StatusNoContent)
}

func (h *LocationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		writeError(w, http.StatusBadRequest, "missing location id")
		return
	}
	locationIDStr := parts[3]

	loc, err := h.locService.GetByID(r.Context(), locationIDStr)
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mapLocationToResponse(loc))
}

func (h *LocationHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	city := r.URL.Query().Get("city")
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

	locations, err := h.locService.List(r.Context(), city, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responseList := make([]dto.LocationResponse, len(locations))
	for i, loc := range locations {
		responseList[i] = mapLocationToResponse(&loc)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responseList)
}

func mapLocationToResponse(loc *entities.Location) dto.LocationResponse {
	return dto.LocationResponse{
		ID:        loc.ID.Hex(),
		AgencyID:  loc.AgencyID.Hex(),
		Name:      loc.Name,
		Address:   loc.Address,
		City:      loc.City,
		Lat:       loc.Lat,
		Lng:       loc.Lng,
		Hours:     loc.Hours,
		Phone:     loc.Phone,
		CreatedAt: loc.CreatedAt.Format(time.RFC3339),
		UpdatedAt: loc.UpdatedAt.Format(time.RFC3339),
	}
}
