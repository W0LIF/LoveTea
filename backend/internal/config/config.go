package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort string

	PostgresHost     string
	PostgresPort     string
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
	PostgresSSLMode  string
	databaseURL      string

	RedisAddr     string
	RedisPassword string
	redisURL      string

	JWTSecret string

	MinIOEndpoint       string
	MinIOPublicEndpoint string
	MinIOAccessKey      string
	MinIOSecretKey      string
	MinIOBucket         string
	MinIOUseTLS         bool

	MailHost   string
	MailPort   int
	MailUser   string
	MailPass   string
	MailFrom   string
	MailUseTLS bool

	// Push notifications
	FCMProjectID  string
	FCMAccessToken string
	APNSTopic     string
}

func Load() (*Config, error) {
	// .env находится в корне проекта, на уровень выше backend.
	if err := godotenv.Load("../.env"); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	cfg := &Config{
		HTTPPort:            getEnv("HTTP_PORT", "8080"),
		PostgresHost:        getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:        getEnv("POSTGRES_PORT", "5434"),
		PostgresDB:          os.Getenv("POSTGRES_DB"),
		PostgresUser:        os.Getenv("POSTGRES_USER"),
		PostgresPassword:    os.Getenv("POSTGRES_PASSWORD"),
		PostgresSSLMode:     getEnv("POSTGRES_SSLMODE", "disable"),
		RedisAddr:           getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:       os.Getenv("REDIS_PASSWORD"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		MinIOEndpoint:       getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOPublicEndpoint: getEnv("MINIO_PUBLIC_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:      getEnv("MINIO_ACCESS_KEY", os.Getenv("MINIO_ROOT_USER")),
		MinIOSecretKey:      getEnv("MINIO_SECRET_KEY", os.Getenv("MINIO_ROOT_PASSWORD")),
		MinIOBucket:         getEnv("MINIO_BUCKET", "lovetea-media"),
	}

	// DATABASE_URL overrides individual Postgres fields (Render managed DB).
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		cfg.PostgresHost = ""
		cfg.PostgresPort = ""
		cfg.PostgresDB = ""
		cfg.PostgresUser = ""
		cfg.PostgresPassword = ""
		cfg.databaseURL = databaseURL
	}

	// REDIS_URL overrides individual Redis fields.
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		cfg.RedisAddr = ""
		cfg.RedisPassword = ""
		cfg.redisURL = redisURL
	}

	if cfg.PostgresDB == "" || cfg.PostgresUser == "" || cfg.PostgresPassword == "" {
		if cfg.databaseURL == "" {
			return nil, fmt.Errorf("POSTGRES_DB, POSTGRES_USER and POSTGRES_PASSWORD must be set, or DATABASE_URL")
		}
	}

	if port, err := strconv.Atoi(cfg.HTTPPort); err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("HTTP_PORT must be a valid TCP port")
	}

	if cfg.databaseURL == "" {
		if port, err := strconv.Atoi(cfg.PostgresPort); err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("POSTGRES_PORT must be a valid TCP port")
		}
	}

	if cfg.redisURL == "" {
		if _, _, err := net.SplitHostPort(cfg.RedisAddr); err != nil {
			return nil, fmt.Errorf("invalid REDIS_ADDR: %w", err)
		}
	}

	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	if cfg.MinIOAccessKey == "" || cfg.MinIOSecretKey == "" {
		return nil, fmt.Errorf("MINIO_ACCESS_KEY and MINIO_SECRET_KEY must be set")
	}
	if _, _, err := net.SplitHostPort(cfg.MinIOEndpoint); err != nil {
		return nil, fmt.Errorf("invalid MINIO_ENDPOINT: %w", err)
	}
	if _, _, err := net.SplitHostPort(cfg.MinIOPublicEndpoint); err != nil {
		return nil, fmt.Errorf("invalid MINIO_PUBLIC_ENDPOINT: %w", err)
	}
	if cfg.MinIOBucket == "" {
		return nil, fmt.Errorf("MINIO_BUCKET must be set")
	}
	if raw := os.Getenv("MINIO_USE_TLS"); raw != "" {
		useTLS, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid MINIO_USE_TLS: %w", err)
		}
		cfg.MinIOUseTLS = useTLS
	}

	cfg.MailHost = os.Getenv("MAIL_HOST")
	if raw := os.Getenv("MAIL_PORT"); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid MAIL_PORT: %w", err)
		}
		cfg.MailPort = port
	}
	cfg.MailUser = os.Getenv("MAIL_USER")
	cfg.MailPass = os.Getenv("MAIL_PASS")
	cfg.MailFrom = os.Getenv("MAIL_FROM")
	if raw := os.Getenv("MAIL_USE_TLS"); raw != "" {
		useTLS, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid MAIL_USE_TLS: %w", err)
		}
		cfg.MailUseTLS = useTLS
	}

	cfg.FCMProjectID = os.Getenv("FCM_PROJECT_ID")
	cfg.FCMAccessToken = os.Getenv("FCM_ACCESS_TOKEN")
	cfg.APNSTopic = os.Getenv("APNS_TOPIC")

	return cfg, nil
}

func (c *Config) PostgresDSN() string {
	if c.databaseURL != "" {
		return c.databaseURL
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.PostgresUser, c.PostgresPassword),
		Host:   net.JoinHostPort(c.PostgresHost, c.PostgresPort),
		Path:   c.PostgresDB,
	}

	query := u.Query()
	query.Set("sslmode", c.PostgresSSLMode)
	query.Set("connect_timeout", "5")
	u.RawQuery = query.Encode()

	return u.String()
}

func (c *Config) RedisDSN() string {
	if c.redisURL != "" {
		return c.redisURL
	}
	if c.RedisPassword != "" {
		return "redis://:" + c.RedisPassword + "@" + c.RedisAddr
	}
	return "redis://" + c.RedisAddr
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
