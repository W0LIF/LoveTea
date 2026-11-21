package planning

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

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, err := parseListPage(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pagination parameters")
		return
	}
	tasks, err := h.service.ListTasks(r.Context(), userID, page)
	if err != nil {
		if errors.Is(err, ErrInvalidListPage) {
			writeError(w, http.StatusBadRequest, "invalid pagination parameters")
			return
		}
		h.logger.Error("failed to list tasks", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) PushSyncChanges(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 512*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request SyncPushRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid sync push request")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request must contain one JSON object")
		return
	}

	result, err := h.service.PushSyncOperations(r.Context(), userID, request)
	if err != nil {
		if errors.Is(err, ErrInvalidSyncPush) {
			writeError(w, http.StatusBadRequest, "invalid sync push request")
			return
		}
		h.logger.Error("failed to push sync operations", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ListCalendarEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	page, err := parseListPage(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pagination parameters")
		return
	}
	events, err := h.service.ListCalendarEvents(r.Context(), userID, page)
	if err != nil {
		if errors.Is(err, ErrInvalidListPage) {
			writeError(w, http.StatusBadRequest, "invalid pagination parameters")
			return
		}
		h.logger.Error("failed to list calendar events", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *Handler) CreateCalendarEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var request CreateCalendarEventRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	event, err := h.service.CreateCalendarEvent(r.Context(), userID, request)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

func (h *Handler) UpdateCalendarEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid calendar event id")
		return
	}
	var request UpdateCalendarEventRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	event, err := h.service.UpdateCalendarEvent(r.Context(), userID, eventID, request)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (h *Handler) DeleteCalendarEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid calendar event id")
		return
	}
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid calendar event version")
		return
	}
	if err := h.service.DeleteCalendarEvent(r.Context(), userID, eventID, version); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListImportantDates(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	page, err := parseListPage(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pagination parameters")
		return
	}
	dates, err := h.service.ListImportantDates(r.Context(), userID, page)
	if err != nil {
		if errors.Is(err, ErrInvalidListPage) {
			writeError(w, http.StatusBadRequest, "invalid pagination parameters")
			return
		}
		h.logger.Error("failed to list important dates", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, dates)
}

func (h *Handler) CreateImportantDate(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var request CreateImportantDateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	date, err := h.service.CreateImportantDate(r.Context(), userID, request)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, date)
}

func (h *Handler) UpdateImportantDate(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	dateID, err := uuid.Parse(chi.URLParam(r, "dateID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid important date id")
		return
	}
	var request UpdateImportantDateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	date, err := h.service.UpdateImportantDate(r.Context(), userID, dateID, request)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, date)
}

func (h *Handler) DeleteImportantDate(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	dateID, err := uuid.Parse(chi.URLParam(r, "dateID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid important date id")
		return
	}
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid important date version")
		return
	}
	if err := h.service.DeleteImportantDate(r.Context(), userID, dateID, version); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var request CreateTaskRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	task, err := h.service.CreateTask(r.Context(), userID, request)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	taskID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	var request UpdateTaskRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	task, err := h.service.UpdateTask(r.Context(), userID, taskID, request)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	taskID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task version")
		return
	}
	if err := h.service.DeleteTask(r.Context(), userID, taskID, version); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidTask),
		errors.Is(err, ErrInvalidCalendarEvent),
		errors.Is(err, ErrInvalidImportantDate):
		writeError(w, http.StatusBadRequest, "invalid planning data")
	case errors.Is(err, ErrNoActiveCouple):
		writeError(w, http.StatusConflict, "an active couple is required")
	case errors.Is(err, ErrTaskNotFound):
		writeError(w, http.StatusNotFound, "task not found")
	case errors.Is(err, ErrCalendarEventNotFound):
		writeError(w, http.StatusNotFound, "calendar event not found")
	case errors.Is(err, ErrImportantDateNotFound):
		writeError(w, http.StatusNotFound, "important date not found")
	case errors.Is(err, ErrVersionConflict):
		writeError(w, http.StatusConflict, "resource version conflict")
	default:
		h.logger.Error("task operation failed", "error", err)
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

func parseListPage(r *http.Request) (ListPage, error) {
	page := ListPage{Limit: defaultListLimit}
	query := r.URL.Query()
	if values, exists := query["limit"]; exists {
		if len(values) != 1 {
			return ListPage{}, ErrInvalidListPage
		}
		limit, err := strconv.Atoi(values[0])
		if err != nil {
			return ListPage{}, ErrInvalidListPage
		}
		page.Limit = limit
	}
	if values, exists := query["offset"]; exists {
		if len(values) != 1 {
			return ListPage{}, ErrInvalidListPage
		}
		offset, err := strconv.Atoi(values[0])
		if err != nil {
			return ListPage{}, ErrInvalidListPage
		}
		page.Offset = offset
	}
	if err := validateListPage(page); err != nil {
		return ListPage{}, err
	}
	return page, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
