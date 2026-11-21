package config

import (
	"net/url"
	"testing"
)

func setValidEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("HTTP_PORT", "8080")
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5434")
	t.Setenv("POSTGRES_DB", "lovetea")
	t.Setenv("POSTGRES_USER", "love tea")
	t.Setenv("POSTGRES_PASSWORD", "p@ss:/word")
	t.Setenv("POSTGRES_SSLMODE", "verify-full")
	t.Setenv("REDIS_ADDR", "localhost:6379")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_PUBLIC_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "minio-access")
	t.Setenv("MINIO_SECRET_KEY", "minio-secret")
	t.Setenv("MINIO_BUCKET", "lovetea-media")
	t.Setenv("MINIO_USE_TLS", "")
}

func TestPostgresDSNEscapesCredentials(t *testing.T) {
	setValidEnvironment(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	parsed, err := url.Parse(cfg.PostgresDSN())
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}
	username := parsed.User.Username()
	password, _ := parsed.User.Password()
	if username != "love tea" || password != "p@ss:/word" {
		t.Fatalf("credentials did not round-trip: %q / %q", username, password)
	}
	if got := parsed.Query().Get("sslmode"); got != "verify-full" {
		t.Fatalf("sslmode = %q, want verify-full", got)
	}
}

func TestLoadRejectsOutOfRangeHTTPPort(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("HTTP_PORT", "65536")

	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted an out-of-range HTTP_PORT")
	}
}

