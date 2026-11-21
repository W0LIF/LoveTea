package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/W0LIF/LoveTea/backend/internal/platform/security"
)

func TestAuthMiddlewareReturnsJSONForMissingCredentials(t *testing.T) {
	handler := AuthMiddleware(security.NewTokenManager(strings.Repeat("s", 32)))(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("next handler was called without credentials")
		}),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}

	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["error"] != "unauthorized" {
		t.Fatalf("unexpected error body: %v", body)
	}
}
