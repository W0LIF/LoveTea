package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

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

func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	conv, err := h.service.GetConversation(r.Context(), userID)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, conv)
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit := 30
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := parseInt(v, 1, 50); err == nil {
			limit = parsed
		}
	}
	result, err := h.service.ListMessages(
		r.Context(), userID, limit, r.URL.Query().Get("before"),
	)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req CreateMessageRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	msg, err := h.service.SendMessage(r.Context(), userID, req.Content, req.Ciphertext)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	messageID, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message id")
		return
	}
	if err := h.service.DeleteMessage(r.Context(), userID, messageID); err != nil {
		h.error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AddReaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	messageID, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message id")
		return
	}
	var req AddReactionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rxn, err := h.service.AddReaction(r.Context(), userID, messageID, req.Emoji)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rxn)
}

func (h *Handler) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	messageID, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message id")
		return
	}
	emoji := r.URL.Query().Get("emoji")
	if err := h.service.RemoveReaction(r.Context(), userID, messageID, emoji); err != nil {
		h.error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListReactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	messagesParam := r.URL.Query().Get("messages")
	var messageIDs []string
	if messagesParam != "" {
		messageIDs = strings.Split(messagesParam, ",")
	}
	byMessage, err := h.service.ListReactions(r.Context(), userID, messageIDs)
	if err != nil {
		h.error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, byMessage)
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
		h.logger.Error("chat handler error", "error", err)
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

func parseInt(s string, min, max int) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return 0, err
	}
	if n < min || n > max {
		return 0, errors.New("out of range")
	}
	return n, nil
}
