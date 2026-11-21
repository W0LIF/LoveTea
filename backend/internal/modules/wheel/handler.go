package wheel

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/W0LIF/LoveTea/backend/internal/modules/auth"
	"github.com/google/uuid"
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

func (h *Handler) ListSectors(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sectors, err := h.service.ListSectors(r.Context(), userID)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sectors)
}

func (h *Handler) CreateSector(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req CreateSectorRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	sector, err := h.service.CreateSector(r.Context(), userID, req)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sector)
}

func (h *Handler) DeleteSector(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sectorID, err := uuid.Parse(r.URL.Path[len("/api/v1/wheel/sectors/"):])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid sector id")
		return
	}
	if err := h.service.DeleteSector(r.Context(), userID, sectorID); err != nil {
		h.error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Spin(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	result, err := h.service.Spin(r.Context(), userID)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req ConfirmRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.service.Confirm(r.Context(), userID, req); err != nil {
		h.error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	spins, err := h.service.History(r.Context(), userID)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, spins)
}

func (h *Handler) error(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNoActiveCouple):
		writeError(w, http.StatusConflict, "an active couple is required")
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	default:
		h.logger.Error("wheel error", "error", err)
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
