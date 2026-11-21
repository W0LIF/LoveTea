package pet

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/W0LIF/LoveTea/backend/internal/modules/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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

func (h *Handler) GetPetState(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	coupleID, err := h.service.repo.GetActiveCoupleID(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get active couple", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if coupleID == uuid.Nil {
		writeError(w, http.StatusNotFound, "no active couple")
		return
	}

	pet, err := h.service.GetPetState(r.Context(), coupleID)
	if err != nil {
		h.logger.Error("failed to get pet state", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, pet)
}

func (h *Handler) UpdatePetAppearance(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	coupleID, err := h.service.repo.GetActiveCoupleID(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get active couple", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if coupleID == uuid.Nil {
		writeError(w, http.StatusNotFound, "no active couple")
		return
	}

	var req UpdatePetAppearanceRequest
	if err := decodeJSONBody(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pet, err := h.service.UpdatePetAppearance(r.Context(), coupleID, req)
	if err != nil {
		var invErr errInvalidAppearanceKey
		if errors.As(err, &invErr) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("failed to update pet appearance", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, pet)
}

var _ = context.Background
var _ = pgxpool.Pool{}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeJSONBody(body io.ReadCloser, dst any) error {
	body = http.MaxBytesReader(nil, body, 16*1024)
	defer body.Close()
	decoder := json.NewDecoder(body)
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
