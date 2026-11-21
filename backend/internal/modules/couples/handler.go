package couples

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/W0LIF/LoveTea/backend/internal/modules/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateCouple(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.service.CreateCouple(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	coupleID, err := uuid.Parse(chi.URLParam(r, "coupleID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid couple id")
		return
	}

	result, err := h.service.CreateInvitation(
		r.Context(),
		coupleID,
		userID,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var request AcceptInvitationRequest

	decoder := json.NewDecoder(io.LimitReader(r.Body, 4096))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request must contain one JSON object")
		return
	}

	result, err := h.service.AcceptInvitation(
		r.Context(),
		userID,
		request,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAlreadyInCouple):
		writeError(w, http.StatusConflict, "user already belongs to a couple")
	case errors.Is(err, ErrCoupleNotFound):
		writeError(w, http.StatusNotFound, "couple not found")
	case errors.Is(err, ErrCoupleNotPending):
		writeError(w, http.StatusConflict, "couple is not waiting for a partner")
	case errors.Is(err, ErrInvitationNotFound):
		writeError(w, http.StatusNotFound, "invitation not found")
	case errors.Is(err, ErrInvitationExpired):
		writeError(w, http.StatusGone, "invitation has expired")
	case errors.Is(err, ErrInvitationUsed):
		writeError(w, http.StatusConflict, "invitation has already been used")
	case errors.Is(err, ErrCannotJoinOwnCouple):
		writeError(w, http.StatusConflict, "cannot accept your own invitation")
	case errors.Is(err, ErrCoupleIsFull):
		writeError(w, http.StatusConflict, "couple already has two members")
	case errors.Is(err, ErrInvitationAttempts):
		writeError(w, http.StatusTooManyRequests, "too many attempts")
	case errors.Is(err, ErrNoActiveCouple):
		writeError(w, http.StatusNotFound, "user does not belong to a couple")
	case errors.Is(err, ErrCoupleNotActive):
		writeError(w, http.StatusConflict, "couple is not active")
	case errors.Is(err, ErrInvalidInvitationInput),
		errors.Is(err, ErrInvalidMemberProfile):
		writeError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, ErrInvitationRateLimited):
		writeError(w, http.StatusTooManyRequests, "too many attempts; try again later")
	case errors.Is(err, ErrRateLimitUnavailable):
		writeError(w, http.StatusServiceUnavailable, "invitation verification temporarily unavailable")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func (h *Handler) GetMyCouple(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.service.GetMyCouple(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var request UpdateMemberProfileRequest

	decoder := json.NewDecoder(io.LimitReader(r.Body, 4096))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request must contain one JSON object")
		return
	}

	if err := h.service.UpdateMemberProfile(
		r.Context(),
		userID,
		request,
	); err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "profile updated successfully",
	})
}

func (h *Handler) ConfirmDetach(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.service.ConfirmDetach(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
