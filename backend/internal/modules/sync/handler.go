package sync

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

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

func (h *Handler) Pull(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := r.URL.Query()
	cursor := int64(0)
	if value := query.Get("cursor"); value != "" {
		if !validCursorText(value) {
			writeError(w, http.StatusBadRequest, "invalid sync cursor")
			return
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid sync cursor")
			return
		}
		cursor = parsed
	}

	limit := 100
	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, "invalid sync limit")
			return
		}
		limit = parsed
	}

	page, err := h.service.Pull(r.Context(), userID, cursor, limit)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCursor):
			writeError(w, http.StatusBadRequest, "invalid sync cursor")
		case errors.Is(err, ErrInvalidLimit):
			writeError(w, http.StatusBadRequest, "invalid sync limit")
		default:
			h.logger.Error("failed to pull sync changes", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, page)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
