package app

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/W0LIF/LoveTea/backend/internal/config"
	"github.com/W0LIF/LoveTea/backend/internal/platform/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func TestRoutesRequireAuthentication(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "create couple", method: http.MethodPost, path: "/api/v1/couples"},
		{name: "pull sync changes", method: http.MethodGet, path: "/api/v1/sync/pull"},
		{name: "push sync operations", method: http.MethodPost, path: "/api/v1/sync/push"},
		{name: "connect sync websocket", method: http.MethodGet, path: "/api/v1/sync/ws"},
		{name: "get home snapshot", method: http.MethodGet, path: "/api/v1/home"},
		{name: "list tasks", method: http.MethodGet, path: "/api/v1/tasks"},
		{name: "list calendar events", method: http.MethodGet, path: "/api/v1/calendar/events"},
		{name: "list important dates", method: http.MethodGet, path: "/api/v1/calendar/dates"},
		{name: "update calendar event", method: http.MethodPut, path: "/api/v1/calendar/events/00000000-0000-0000-0000-000000000001"},
		{name: "delete important date", method: http.MethodDelete, path: "/api/v1/calendar/dates/00000000-0000-0000-0000-000000000001"},
		{name: "list mood", method: http.MethodGet, path: "/api/v1/wellness/mood"},
		{name: "save sleep", method: http.MethodPost, path: "/api/v1/wellness/sleep"},
		{name: "save mood", method: http.MethodPost, path: "/api/v1/wellness/mood"},
		{name: "list sleep", method: http.MethodGet, path: "/api/v1/wellness/sleep"},
		{name: "list habits", method: http.MethodGet, path: "/api/v1/habits"},
		{name: "complete habit", method: http.MethodPost, path: "/api/v1/habits/00000000-0000-0000-0000-000000000001/completions"},
		{name: "get habit statistics", method: http.MethodGet, path: "/api/v1/habits/00000000-0000-0000-0000-000000000001/stats"},
		{name: "list media", method: http.MethodGet, path: "/api/v1/media"},
		{name: "request media upload", method: http.MethodPost, path: "/api/v1/media/uploads"},
		{name: "complete media upload", method: http.MethodPost, path: "/api/v1/media/00000000-0000-0000-0000-000000000001/complete"},
		{name: "list moments", method: http.MethodGet, path: "/api/v1/moments"},
		{name: "create moment", method: http.MethodPost, path: "/api/v1/moments"},
		{name: "list memories", method: http.MethodGet, path: "/api/v1/memories"},
		{name: "create memory", method: http.MethodPost, path: "/api/v1/memories"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
			t.Cleanup(func() {
				_ = redisClient.Close()
			})

			application, err := New(
				&config.Config{
					HTTPPort:            "8080",
					JWTSecret:           "01234567890123456789012345678901",
					MinIOEndpoint:       "localhost:9000",
					MinIOPublicEndpoint: "localhost:9000",
					MinIOAccessKey:      "minio-access",
					MinIOSecretKey:      "minio-secret",
					MinIOBucket:         "lovetea-media",
				},
				slog.Default(),
				&pgxpool.Pool{},
				redisClient,
				security.NewTokenManager("01234567890123456789012345678901"),
			)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()
			application.Server.Handler.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s status = %d, want %d", test.method, test.path, response.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestMemoriesRoutesAreNowActive(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer redisClient.Close()

	tm := security.NewTokenManager("01234567890123456789012345678901")
	application, err := New(
		&config.Config{
			HTTPPort:            "8080",
			JWTSecret:           "01234567890123456789012345678901",
			MinIOEndpoint:       "localhost:9000",
			MinIOPublicEndpoint: "localhost:9000",
			MinIOAccessKey:      "minio-access",
			MinIOSecretKey:      "minio-secret",
			MinIOBucket:         "lovetea-media",
		},
		slog.Default(),
		&pgxpool.Pool{},
		redisClient,
		tm,
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	accessToken, err := tm.GenerateAccessToken(uuid.New())
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// Routes are registered — unauthenticated requests return 401.
	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/memories"},
		{method: http.MethodPost, path: "/api/v1/memories"},
	} {
		request := httptest.NewRequest(test.method, test.path, nil)
		response := httptest.NewRecorder()
		application.Server.Handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("unauth %s %s status = %d, want %d", test.method, test.path, response.Code, http.StatusUnauthorized)
		}
	}

	// With a valid token but no real DB, routes are reachable (500 or other, not 401/404).
	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/memories"},
		{method: http.MethodPost, path: "/api/v1/memories"},
	} {
		request := httptest.NewRequest(test.method, test.path, nil)
		request.Header.Set("Authorization", "Bearer "+accessToken)
		response := httptest.NewRecorder()
		application.Server.Handler.ServeHTTP(response, request)
		if response.Code == http.StatusUnauthorized || response.Code == http.StatusNotFound {
			t.Errorf("auth %s %s status = %d, want reachable (not 401/404)", test.method, test.path, response.Code)
		}
	}
}

func TestEmailDependentAuthRoutesAreNotRegistered(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer redisClient.Close()

	application, err := New(
		&config.Config{
			HTTPPort:            "8080",
			JWTSecret:           "01234567890123456789012345678901",
			MinIOEndpoint:       "localhost:9000",
			MinIOPublicEndpoint: "localhost:9000",
			MinIOAccessKey:      "minio-access",
			MinIOSecretKey:      "minio-secret",
			MinIOBucket:         "lovetea-media",
		},
		slog.Default(),
		&pgxpool.Pool{},
		redisClient,
		security.NewTokenManager("01234567890123456789012345678901"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	for _, path := range []string{"/api/v1/auth/password/forgot", "/api/v1/auth/password/reset"} {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		response := httptest.NewRecorder()
		application.Server.Handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Errorf("POST %s status = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}
}
