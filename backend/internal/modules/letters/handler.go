package letters

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

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

func (h *Handler) ListMyLetters(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	letters, err := h.service.ListMyLetters(r.Context(), userID)
	if err != nil {
		h.internalError(w, "failed to list letters", err)
		return
	}
	writeJSON(w, http.StatusOK, letters)
}

func (h *Handler) ListForMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	letters, err := h.service.ListForMe(r.Context(), userID)
	if err != nil {
		h.internalError(w, "failed to list incoming letters", err)
		return
	}
	writeJSON(w, http.StatusOK, letters)
}

func (h *Handler) CreateLetter(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req CreateLetterRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	letter, err := h.service.CreateLetter(r.Context(), userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, letter)
}

func (h *Handler) GetLetter(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	letterID, err := uuid.Parse(chi.URLParam(r, "letterID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid letter id")
		return
	}
	letter, err := h.service.GetLetter(r.Context(), userID, letterID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, letter)
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	letterID, err := uuid.Parse(chi.URLParam(r, "letterID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid letter id")
		return
	}
	if err := h.service.MarkRead(r.Context(), userID, letterID); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	letterID, err := uuid.Parse(chi.URLParam(r, "letterID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid letter id")
		return
	}
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid version")
		return
	}
	if err := h.service.Archive(r.Context(), userID, letterID, version); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CooldownRemaining(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	remaining, err := h.service.GetCooldownRemaining(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cooldown_remaining_seconds": int(remaining.Seconds()),
		"cooldown_until":             time.Now().Add(remaining).Format(time.RFC3339),
	})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid letter data")
	case errors.Is(err, ErrNoActiveCouple):
		writeError(w, http.StatusConflict, "an active couple is required")
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "letter not found")
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, "letter is not yet ready to read")
	case errors.Is(err, ErrCooldownActive):
		writeError(w, http.StatusConflict, "cooldown is still active")
	default:
		h.internalError(w, "letter operation failed", err)
	}
}

func (h *Handler) internalError(w http.ResponseWriter, msg string, err error) {
	h.logger.Error(msg, "error", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
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
