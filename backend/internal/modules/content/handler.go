package content

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

func (h *Handler) ListMedia(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	media, err := h.service.ListMedia(r.Context(), userID)
	if err != nil {
		h.internalError(w, "failed to list media", err)
		return
	}
	writeJSON(w, http.StatusOK, media)
}

func (h *Handler) CreateUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var request UploadRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	response, err := h.service.CreateUpload(r.Context(), userID, request)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid media id")
		return
	}
	media, err := h.service.CompleteUpload(r.Context(), userID, mediaID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, media)
}

func (h *Handler) CreateDownloadURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid media id")
		return
	}
	url, expiresAt, err := h.service.CreateDownloadURL(r.Context(), userID, mediaID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"url":        url,
		"expires_at": expiresAt,
	})
}

func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid media id")
		return
	}
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid media version")
		return
	}
	if err := h.service.DeleteMedia(r.Context(), userID, mediaID, version); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListMoments(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	moments, err := h.service.ListMoments(r.Context(), userID)
	if err != nil {
		h.internalError(w, "failed to list moments", err)
		return
	}
	writeJSON(w, http.StatusOK, moments)
}

func (h *Handler) CreateMoment(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var request CreateMomentRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	moment, err := h.service.CreateMoment(r.Context(), userID, request)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, moment)
}

func (h *Handler) MemoriesUnavailable(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeError(w, http.StatusNotImplemented, "memory storage is unavailable until end-to-end encryption is designed")
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidContent):
		writeError(w, http.StatusBadRequest, "invalid content data")
	case errors.Is(err, ErrNoActiveCouple):
		writeError(w, http.StatusConflict, "an active couple is required")
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "content not found")
	case errors.Is(err, ErrNotReady):
		writeError(w, http.StatusConflict, "media upload is not complete")
	case errors.Is(err, ErrVersionConflict):
		writeError(w, http.StatusConflict, "resource version conflict")
	default:
		h.internalError(w, "content operation failed", err)
	}
}

func (h *Handler) internalError(w http.ResponseWriter, message string, err error) {
	h.logger.Error(message, "error", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
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
