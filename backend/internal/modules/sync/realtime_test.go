package sync

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/W0LIF/LoveTea/backend/internal/modules/auth"
	"github.com/W0LIF/LoveTea/backend/internal/platform/security"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

func TestRealtimeWebSocketSendsOnlySyncCursor(t *testing.T) {
	userID := uuid.New()
	tokens := security.NewTokenManager("01234567890123456789012345678901")
	accessToken, err := tokens.GenerateAccessToken(userID)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	hub := NewHub()
	handler := NewRealtimeHandler(hub, slog.Default())
	server := httptest.NewUnstartedServer(auth.AuthMiddleware(tokens)(handler))
	server.Config.ReadTimeout = 100 * time.Millisecond
	server.Config.WriteTimeout = 100 * time.Millisecond
	server.Start()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, server.URL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + accessToken}},
	})
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	time.Sleep(250 * time.Millisecond)
	hub.Publish(userID, 42)

	var event RealtimeEvent
	if err := wsjson.Read(ctx, conn, &event); err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if event != (RealtimeEvent{Type: "sync.available", Cursor: 42}) {
		t.Fatalf("WebSocket event = %+v, want sync cursor only", event)
	}
}

func TestHubPublishesOnlyToTargetUser(t *testing.T) {
	hub := NewHub()
	targetID := uuid.New()
	otherID := uuid.New()
	targetEvents, unsubscribeTarget := hub.Subscribe(targetID)
	defer unsubscribeTarget()
	otherEvents, unsubscribeOther := hub.Subscribe(otherID)
	defer unsubscribeOther()

	hub.Publish(targetID, 17)

	select {
	case event := <-targetEvents:
		if event != (RealtimeEvent{Type: "sync.available", Cursor: 17}) {
			t.Fatalf("target event = %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("target subscriber did not receive an event")
	}

	select {
	case event := <-otherEvents:
		t.Fatalf("unrelated subscriber received event: %+v", event)
	default:
	}
}
