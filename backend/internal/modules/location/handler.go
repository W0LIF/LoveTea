package location

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/W0LIF/LoveTea/backend/internal/modules/auth"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{service: service, logger: logger}
}

func (h *Handler) GetSharing(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	status, err := h.service.GetSharingStatus(r.Context(), userID)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) ToggleSharing(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req ToggleSharingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	status, err := h.service.ToggleSharing(r.Context(), userID, req)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) UpdatePosition(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req UpdatePositionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.service.UpdatePosition(r.Context(), userID, req); err != nil {
		h.error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetPartnerPosition(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	pos, err := h.service.GetPartnerPosition(r.Context(), userID)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pos)
}

func (h *Handler) ListGeofences(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	geofences, err := h.service.ListGeofences(r.Context(), userID)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, geofences)
}

func (h *Handler) CreateGeofence(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req CreateGeofenceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	gf, err := h.service.CreateGeofence(r.Context(), userID, req)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, gf)
}

func (h *Handler) error(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNoActiveCouple):
		writeError(w, http.StatusConflict, "an active couple is required")
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	default:
		h.logger.Error("location error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request must contain one JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
