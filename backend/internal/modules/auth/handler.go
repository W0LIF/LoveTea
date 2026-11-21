package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	result, err := h.service.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeAuthJSON(w, http.StatusCreated, result)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	result, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeAuthJSON(w, http.StatusOK, result)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	result, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeAuthJSON(w, http.StatusOK, result)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeAuthJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	var req LogoutRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if err := h.service.Logout(r.Context(), userID, req.RefreshToken); err != nil {
		h.handleError(w, err)
		return
	}

	writeAuthJSON(w, http.StatusOK, map[string]string{
		"message": "logged out",
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeAuthJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	user, err := h.service.GetUser(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeAuthJSON(w, http.StatusOK, user)
}

// ForgotPassword POST /auth/forgot-password
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := h.service.ForgotPassword(r.Context(), req.Email); err != nil {
		h.handleError(w, err)
		return
	}
	writeAuthJSON(w, http.StatusOK, map[string]string{"message": "if the email exists, a reset link has been sent"})
}

// ResetPassword POST /auth/reset-password
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := h.service.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		h.handleError(w, err)
		return
	}
	writeAuthJSON(w, http.StatusOK, map[string]string{"message": "password reset successfully"})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeAuthJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid email or password format",
		})

	case errors.Is(err, ErrEmailAlreadyExists):
		writeAuthJSON(w, http.StatusConflict, map[string]string{
			"error": "email already registered",
		})

	case errors.Is(err, ErrInvalidCredentials):
		writeAuthJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "invalid email or password",
		})

	case errors.Is(err, ErrInvalidRefreshToken):
		writeAuthJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "invalid refresh token",
		})

	case errors.Is(err, ErrUserNotFound):
		writeAuthJSON(w, http.StatusNotFound, map[string]string{
			"error": "user not found",
		})

	default:
		writeAuthJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "internal server error",
		})
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
		return errors.New("request must contain a single JSON object")
	}

	return nil
}

func writeAuthJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
