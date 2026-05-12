package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrate runs all pending up migrations from the given directory.
// It is safe to call on every app startup — already-applied migrations
// are skipped. Pass migrationsPath as "file://migrations" (relative to
// where the binary runs) or an absolute path.
func Migrate(databaseURL, migrationsPath string) error {
	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		return fmt.Errorf("db: failed to init migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("db: migration failed: %w", err)
	}

	return nil
}
