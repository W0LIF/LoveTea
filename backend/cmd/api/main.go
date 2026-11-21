package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/W0LIF/LoveTea/backend/internal/app"
	"github.com/W0LIF/LoveTea/backend/internal/config"
	"github.com/W0LIF/LoveTea/backend/internal/platform/database"
	appLogger "github.com/W0LIF/LoveTea/backend/internal/platform/logger"
	"github.com/W0LIF/LoveTea/backend/internal/platform/migrations"
	redisClient "github.com/W0LIF/LoveTea/backend/internal/platform/redis"
	"github.com/W0LIF/LoveTea/backend/internal/platform/security"
	"github.com/W0LIF/LoveTea/backend/internal/platform/storage"
	reminders "github.com/W0LIF/LoveTea/backend/internal/modules/reminders"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := appLogger.New()
	if err := migrations.Apply(cfg.PostgresDSN()); err != nil {
		return err
	}
	logger.Info("Database migrations applied")

	startupCtx, cancelStartup := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancelStartup()

	db, err := database.NewPostgres(startupCtx, cfg.PostgresDSN())
	if err != nil {
		return err
	}
	defer db.Close()

	logger.Info("PostgreSQL connected")

	rdb, err := redisClient.NewRedis(
		startupCtx,
		cfg.RedisAddr,
		cfg.RedisPassword,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			logger.Error("Redis close failed", slog.String("error", err.Error()))
		}
	}()

	logger.Info("Redis connected")

	tokenManager := security.NewTokenManager(cfg.JWTSecret)

	application, err := app.New(
		cfg,
		logger,
		db,
		rdb,
		tokenManager,
	)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}

	stopCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	workerMode := os.Getenv("WORKER_MODE")

	if workerMode == "reminders" {
		logger.Info("Starting reminders worker mode")
		worker := reminders.NewWorker(
			reminders.NewRepository(db),
			logger,
			60*time.Second,
			nil,
		)
		worker.Run(stopCtx)
		return nil
	}

	objectStore, err := storage.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("configure object storage: %w", err)
	}
	bucketCtx, cancelBucket := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelBucket()
	for {
		if err := objectStore.EnsureBucket(bucketCtx); err == nil {
			break
		} else if bucketCtx.Err() != nil {
			return fmt.Errorf("initialize object storage bucket: %w", err)
		}
		select {
		case <-bucketCtx.Done():
			return fmt.Errorf("initialize object storage bucket: %w", bucketCtx.Err())
		case <-time.After(time.Second):
		}
	}
	logger.Info("Object storage connected")

	go application.Outbox.Run(stopCtx)

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info(
			"HTTP server starting",
			slog.String("address", ":"+cfg.HTTPPort),
		)

		if err := application.Server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		return err

	case <-stopCtx.Done():
		logger.Info("Shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := application.Server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("HTTP server stopped")

	return nil
}
