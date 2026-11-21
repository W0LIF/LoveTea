package sync

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/W0LIF/LoveTea/backend/internal/modules/auth"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

type RealtimeEvent struct {
	Type   string `json:"type"`
	Cursor int64  `json:"cursor"`
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[chan RealtimeEvent]struct{}
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[uuid.UUID]map[chan RealtimeEvent]struct{})}
}

func (h *Hub) Subscribe(userID uuid.UUID) (<-chan RealtimeEvent, func()) {
	events := make(chan RealtimeEvent, 32)
	h.mu.Lock()
	if h.subscribers[userID] == nil {
		h.subscribers[userID] = make(map[chan RealtimeEvent]struct{})
	}
	h.subscribers[userID][events] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			delete(h.subscribers[userID], events)
			if len(h.subscribers[userID]) == 0 {
				delete(h.subscribers, userID)
			}
			close(events)
		})
	}
	return events, unsubscribe
}

func (h *Hub) Publish(userID uuid.UUID, cursor int64) {
	event := RealtimeEvent{Type: "sync.available", Cursor: cursor}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for events := range h.subscribers[userID] {
		select {
		case events <- event:
		default:
		}
	}
}

type RealtimeHandler struct {
	hub    *Hub
	logger *slog.Logger
}

func NewRealtimeHandler(hub *Hub, logger *slog.Logger) *RealtimeHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &RealtimeHandler{hub: hub, logger: logger}
}

func (h *RealtimeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	controller := http.NewResponseController(w)
	if err := controller.SetReadDeadline(time.Time{}); err != nil {
		h.logger.Error("clear WebSocket read deadline", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if err := controller.SetWriteDeadline(time.Time{}); err != nil {
		h.logger.Error("clear WebSocket write deadline", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	events, unsubscribe := h.hub.Subscribe(userID)
	defer unsubscribe()

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.logger.Warn("WebSocket handshake failed", "error", err)
		return
	}
	defer func() {
		if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
			h.logger.Debug("close WebSocket connection", "error", err)
		}
	}()

	ctx := conn.CloseRead(context.Background())
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-events:
			writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := wsjson.Write(writeCtx, conn, event)
			cancel()
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					h.logger.Debug("write WebSocket event", "error", err)
				}
				return
			}
		case <-heartbeat.C:
			pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					h.logger.Debug("ping WebSocket client", "error", err)
				}
				return
			}
		}
	}
}
