package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/joan-ouma/give-blood/internal/dto"
	"github.com/joan-ouma/give-blood/internal/entities"
	"github.com/joan-ouma/give-blood/internal/service"
)

type DriveHandler struct {
	driveService *service.DriveService
}

func NewDriveHandler(driveService *service.DriveService) *DriveHandler {
	return &DriveHandler{driveService: driveService}
}

func (h *DriveHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, role, err := service.GetUserFromContext(r.Context())
	if err != nil || role != string(entities.RoleAgency) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req dto.DriveCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fields := make(map[string]string)
	if strings.TrimSpace(req.Title) == "" {
		fields["title"] = "Title is required"
	}
	if strings.TrimSpace(req.City) == "" {
		fields["city"] = "City is required"
	}

	startsAt, errStart := time.Parse(time.RFC3339, req.StartsAt)
	if errStart != nil {
		fields["startsAt"] = "Invalid RFC3339 date format"
	}
	endsAt, errEnd := time.Parse(time.RFC3339, req.EndsAt)
	if errEnd != nil {
		fields["endsAt"] = "Invalid RFC3339 date format"
	}

	if errStart == nil && errEnd == nil {
		if startsAt.After(endsAt) || startsAt.Equal(endsAt) {
			fields["startsAt"] = "StartsAt must be before EndsAt"
		}
	}

	if req.LocationID == nil || *req.LocationID == "" {
		if strings.TrimSpace(req.Address) == "" {
			fields["address"] = "Address is required when LocationID is empty"
		}
	}

	if len(fields) > 0 {
		writeFieldsError(w, http.StatusBadRequest, "validation failed", fields)
		return
	}

	drive, err := h.driveService.Create(r.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrDriveValidation) {
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

	response := mapDriveToResponse(drive)
	if drive.LocationID != nil {
		namesMap, err := h.driveService.GetLocationsNamesMap(r.Context(), []primitive.ObjectID{*drive.LocationID})
		if err == nil {
			response.LocationName = namesMap[*drive.LocationID]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *DriveHandler) Update(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "missing drive id")
		return
	}
	driveIDStr := parts[3]

	var req dto.DriveUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fields := make(map[string]string)
	if strings.TrimSpace(req.Title) == "" {
		fields["title"] = "Title is required"
	}
	if strings.TrimSpace(req.City) == "" {
		fields["city"] = "City is required"
	}

	startsAt, errStart := time.Parse(time.RFC3339, req.StartsAt)
	if errStart != nil {
		fields["startsAt"] = "Invalid RFC3339 date format"
	}
	endsAt, errEnd := time.Parse(time.RFC3339, req.EndsAt)
	if errEnd != nil {
		fields["endsAt"] = "Invalid RFC3339 date format"
	}

	if errStart == nil && errEnd == nil {
		if startsAt.After(endsAt) || startsAt.Equal(endsAt) {
			fields["startsAt"] = "StartsAt must be before EndsAt"
		}
	}

	if req.LocationID == nil || *req.LocationID == "" {
		if strings.TrimSpace(req.Address) == "" {
			fields["address"] = "Address is required when LocationID is empty"
		}
	}

	if len(fields) > 0 {
		writeFieldsError(w, http.StatusBadRequest, "validation failed", fields)
		return
	}

	drive, err := h.driveService.Update(r.Context(), userID, driveIDStr, &req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "drive not found")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		if errors.Is(err, service.ErrDriveValidation) {
			writeError(w, http.StatusBadRequest, "validation failed")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := mapDriveToResponse(drive)
	if drive.LocationID != nil {
		namesMap, err := h.driveService.GetLocationsNamesMap(r.Context(), []primitive.ObjectID{*drive.LocationID})
		if err == nil {
			response.LocationName = namesMap[*drive.LocationID]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *DriveHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "missing drive id")
		return
	}
	driveIDStr := parts[3]

	err = h.driveService.Delete(r.Context(), userID, driveIDStr)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "drive not found")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DriveHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		writeError(w, http.StatusBadRequest, "missing drive id")
		return
	}
	driveIDStr := parts[2]

	drive, err := h.driveService.GetByID(r.Context(), driveIDStr)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "drive not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := mapDriveToResponse(drive)
	if drive.LocationID != nil {
		namesMap, err := h.driveService.GetLocationsNamesMap(r.Context(), []primitive.ObjectID{*drive.LocationID})
		if err == nil {
			response.LocationName = namesMap[*drive.LocationID]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *DriveHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	city := r.URL.Query().Get("city")
	includePast := r.URL.Query().Get("includePast") == "true"
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	var limit int64 = 20
	if limitStr != "" {
		if parsed, err := strconv.ParseInt(limitStr, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}

	var offset int64 = 0
	if offsetStr != "" {
		if parsed, err := strconv.ParseInt(offsetStr, 10, 64); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	drives, err := h.driveService.List(r.Context(), city, includePast, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	var locationIDs []primitive.ObjectID
	for _, d := range drives {
		if d.LocationID != nil {
			locationIDs = append(locationIDs, *d.LocationID)
		}
	}

	namesMap, err := h.driveService.GetLocationsNamesMap(r.Context(), locationIDs)
	if err != nil {
		namesMap = make(map[primitive.ObjectID]string)
	}

	responseList := make([]dto.DriveResponse, len(drives))
	for i, d := range drives {
		res := mapDriveToResponse(&d)
		if d.LocationID != nil {
			res.LocationName = namesMap[*d.LocationID]
		}
		responseList[i] = res
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responseList)
}

func mapDriveToResponse(d *entities.Drive) dto.DriveResponse {
	var locIDStr *string
	if d.LocationID != nil {
		s := d.LocationID.Hex()
		locIDStr = &s
	}
	return dto.DriveResponse{
		ID:         d.ID.Hex(),
		AgencyID:   d.AgencyID.Hex(),
		LocationID: locIDStr,
		Title:      d.Title,
		Address:    d.Address,
		City:       d.City,
		StartsAt:   d.StartsAt.Format(time.RFC3339),
		EndsAt:     d.EndsAt.Format(time.RFC3339),
		Notes:      d.Notes,
		CreatedAt:  d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  d.UpdatedAt.Format(time.RFC3339),
	}
}
