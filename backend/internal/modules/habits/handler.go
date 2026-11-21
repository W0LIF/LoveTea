package habits

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/W0LIF/LoveTea/backend/internal/modules/auth"
	"github.com/go-chi/chi/v5"
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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	result, err := h.service.List(r.Context(), userID)
	if err != nil {
		h.internalError(w, "failed to list habits", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var request CreateHabitRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	habit, err := h.service.Create(r.Context(), userID, request)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, habit)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	habitID, err := parseHabitID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid habit id")
		return
	}
	var request UpdateHabitRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	habit, err := h.service.Update(r.Context(), userID, habitID, request)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, habit)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	habitID, err := parseHabitID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid habit id")
		return
	}
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid habit version")
		return
	}
	if err := h.service.Delete(r.Context(), userID, habitID, version); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListCompletions(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	habitID, err := parseHabitID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid habit id")
		return
	}
	result, err := h.service.ListCompletions(
		r.Context(), userID, habitID,
		r.URL.Query().Get("from"), r.URL.Query().Get("to"),
	)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	habitID, err := parseHabitID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid habit id")
		return
	}
	var request CompleteHabitRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	completion, err := h.service.Complete(r.Context(), userID, habitID, request)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, completion)
}

func (h *Handler) Uncomplete(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	habitID, err := parseHabitID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid habit id")
		return
	}
	if err := h.service.Uncomplete(r.Context(), userID, habitID, r.URL.Query().Get("completed_on")); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	habitID, err := parseHabitID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid habit id")
		return
	}
	stats, err := h.service.Stats(
		r.Context(), userID, habitID,
		r.URL.Query().Get("from"), r.URL.Query().Get("to"),
	)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, "invalid habit data")
	case errors.Is(err, ErrVersionConflict):
		writeError(w, http.StatusConflict, "habit version conflict")
	case errors.Is(err, ErrNoActiveCouple):
		writeError(w, http.StatusConflict, "an active couple is required")
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "habit not found")
	default:
		h.internalError(w, "habit operation failed", err)
	}
}

func (h *Handler) internalError(w http.ResponseWriter, message string, err error) {
	h.logger.Error(message, "error", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func parseHabitID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "habitID"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, ErrInvalidRequest
	}
	return id, nil
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
