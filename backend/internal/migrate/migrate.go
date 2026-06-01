package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog/log"
)

// Run applies all migrations in the db/migrations/ folder.
func Run(ctx context.Context, databaseURL, migrationsPath string) error {
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required for migrations")
	}

	// golang-migrate file:// source requires an absolute path.
	// Use filepath.Abs to normalize the path and resolve any symlinks.
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for migrations: %w", err)
	}
	// Trim any trailing slashes the file:// source doesn't handle well.
	absPath = strings.TrimRight(absPath, "/")
	log.Info().Str("path", absPath).Str("url", fmt.Sprintf("file://%s", absPath)).Msg("starting migrations")

	// golang-migrate's file source reads directory entries with os.Open and then
	// calls os.Stat(entry.Name()) — which resolves relative to the process CWD,
	// not to the migration directory. Change to the migrations directory so that
	// relative stat calls resolve correctly.
	if err := os.Chdir(absPath); err != nil {
		return fmt.Errorf("failed to chdir to migrations directory %s: %w", absPath, err)
	}
	log.Info().Str("cwd", absPath).Msg("changed working directory for migration files")
	m, err := migrate.New(fmt.Sprintf("file://%s", absPath), databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Log current migration version
	v, dirty, _ := m.Version()
	log.Info().Uint("version", v).Bool("dirty", dirty).Msg("current migration state")

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Log new migration version
	v2, dirty2, _ := m.Version()
	log.Info().Uint("version", v2).Bool("dirty", dirty2).Msg("migrations complete")
	return nil
}