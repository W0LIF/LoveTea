package migrations

import (
	"errors"
	"fmt"

	"github.com/W0LIF/LoveTea/backend/migrations"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func Apply(dsn string) (err error) {
	source, err := iofs.New(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	runner, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return fmt.Errorf("initialize migration runner: %w", err)
	}

	defer func() {
		sourceErr, databaseErr := runner.Close()
		if sourceErr != nil {
			err = errors.Join(err, fmt.Errorf("close migration source: %w", sourceErr))
		}
		if databaseErr != nil {
			err = errors.Join(err, fmt.Errorf("close migration database: %w", databaseErr))
		}
	}()

	if err := runner.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply database migrations: %w", err)
	}

	return nil
}
