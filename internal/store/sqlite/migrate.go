// Package sqlite provides SQLite database storage for clonr.
package sqlite

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migration represents a database migration.
type Migration struct {
	Version     int
	Description string
	UpSQL       string
	DownSQL     string
}

// MigrationRecord represents a record in the schema_migrations table.
type MigrationRecord struct {
	Version     int
	AppliedAt   time.Time
	Description string
}

// Migrator handles database migrations.
type Migrator struct {
	db *sql.DB
}

// NewMigrator creates a new migration handler.
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

// LoadMigrations loads all migrations from the embedded filesystem.
func (m *Migrator) LoadMigrations() ([]Migration, error) {
	migrations := make(map[int]*Migration)

	// Read all migration files
	err := fs.WalkDir(migrationsFS, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		filename := filepath.Base(path)
		if !strings.HasSuffix(filename, ".sql") {
			return nil
		}

		// Parse filename: 001_description.up.sql or 001_description.down.sql
		re := regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)

		matches := re.FindStringSubmatch(filename)
		if len(matches) != 4 {
			return nil
		}

		version, _ := strconv.Atoi(matches[1])
		description := strings.ReplaceAll(matches[2], "_", " ")
		direction := matches[3]

		content, err := migrationsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", path, err)
		}

		if _, exists := migrations[version]; !exists {
			migrations[version] = &Migration{
				Version:     version,
				Description: description,
			}
		}

		if direction == "up" {
			migrations[version].UpSQL = string(content)
		} else {
			migrations[version].DownSQL = string(content)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking migrations: %w", err)
	}

	// Convert map to sorted slice
	var result []Migration
	for _, mig := range migrations {
		result = append(result, *mig)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

// CurrentVersion returns the current schema version.
func (m *Migrator) CurrentVersion() (int, error) {
	// Check if schema_migrations table exists
	var tableName string

	err := m.db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name='schema_migrations'
	`).Scan(&tableName)
	if err == sql.ErrNoRows {
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("checking schema_migrations table: %w", err)
	}

	// Get the latest version
	var version int

	err = m.db.QueryRow(`
		SELECT COALESCE(MAX(version), 0) FROM schema_migrations
	`).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("getting current version: %w", err)
	}

	return version, nil
}

// AppliedMigrations returns all applied migrations.
func (m *Migrator) AppliedMigrations() ([]MigrationRecord, error) {
	currentVersion, err := m.CurrentVersion()
	if err != nil {
		return nil, err
	}

	if currentVersion == 0 {
		return nil, nil
	}

	rows, err := m.db.Query(`
		SELECT version, applied_at, COALESCE(description, '') as description
		FROM schema_migrations
		ORDER BY version ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying applied migrations: %w", err)
	}
	defer rows.Close()

	var records []MigrationRecord

	for rows.Next() {
		var rec MigrationRecord
		if err := rows.Scan(&rec.Version, &rec.AppliedAt, &rec.Description); err != nil {
			return nil, fmt.Errorf("scanning migration record: %w", err)
		}

		records = append(records, rec)
	}

	return records, rows.Err()
}

// MigrateUp applies all pending migrations.
func (m *Migrator) MigrateUp() error {
	migrations, err := m.LoadMigrations()
	if err != nil {
		return fmt.Errorf("loading migrations: %w", err)
	}

	currentVersion, err := m.CurrentVersion()
	if err != nil {
		return fmt.Errorf("getting current version: %w", err)
	}

	for _, mig := range migrations {
		if mig.Version <= currentVersion {
			continue
		}

		if mig.UpSQL == "" {
			return fmt.Errorf("migration %d has no up SQL", mig.Version)
		}

		if err := m.runMigration(mig.UpSQL); err != nil {
			return fmt.Errorf("applying migration %d (%s): %w", mig.Version, mig.Description, err)
		}
	}

	return nil
}

// MigrateDown rolls back the last migration.
func (m *Migrator) MigrateDown() error {
	migrations, err := m.LoadMigrations()
	if err != nil {
		return fmt.Errorf("loading migrations: %w", err)
	}

	currentVersion, err := m.CurrentVersion()
	if err != nil {
		return fmt.Errorf("getting current version: %w", err)
	}

	if currentVersion == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	// Find the current migration
	var currentMigration *Migration

	for i := range migrations {
		if migrations[i].Version == currentVersion {
			currentMigration = &migrations[i]
			break
		}
	}

	if currentMigration == nil {
		return fmt.Errorf("migration %d not found", currentVersion)
	}

	if currentMigration.DownSQL == "" {
		return fmt.Errorf("migration %d has no down SQL", currentVersion)
	}

	if err := m.runMigration(currentMigration.DownSQL); err != nil {
		return fmt.Errorf("rolling back migration %d (%s): %w",
			currentVersion, currentMigration.Description, err)
	}

	return nil
}

// MigrateTo migrates to a specific version.
func (m *Migrator) MigrateTo(targetVersion int) error {
	currentVersion, err := m.CurrentVersion()
	if err != nil {
		return fmt.Errorf("getting current version: %w", err)
	}

	if targetVersion == currentVersion {
		return nil
	}

	if targetVersion > currentVersion {
		// Migrate up
		migrations, err := m.LoadMigrations()
		if err != nil {
			return fmt.Errorf("loading migrations: %w", err)
		}

		for _, mig := range migrations {
			if mig.Version <= currentVersion || mig.Version > targetVersion {
				continue
			}

			if mig.UpSQL == "" {
				return fmt.Errorf("migration %d has no up SQL", mig.Version)
			}

			if err := m.runMigration(mig.UpSQL); err != nil {
				return fmt.Errorf("applying migration %d: %w", mig.Version, err)
			}
		}
	} else {
		// Migrate down
		migrations, err := m.LoadMigrations()
		if err != nil {
			return fmt.Errorf("loading migrations: %w", err)
		}

		// Reverse order for rollback
		for i := len(migrations) - 1; i >= 0; i-- {
			mig := migrations[i]
			if mig.Version <= targetVersion || mig.Version > currentVersion {
				continue
			}

			if mig.DownSQL == "" {
				return fmt.Errorf("migration %d has no down SQL", mig.Version)
			}

			if err := m.runMigration(mig.DownSQL); err != nil {
				return fmt.Errorf("rolling back migration %d: %w", mig.Version, err)
			}
		}
	}

	return nil
}

// runMigration executes a migration SQL script.
func (m *Migrator) runMigration(sqlScript string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(sqlScript); err != nil {
		return fmt.Errorf("executing migration: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// PendingMigrations returns migrations that have not been applied.
func (m *Migrator) PendingMigrations() ([]Migration, error) {
	migrations, err := m.LoadMigrations()
	if err != nil {
		return nil, err
	}

	currentVersion, err := m.CurrentVersion()
	if err != nil {
		return nil, err
	}

	var pending []Migration

	for _, mig := range migrations {
		if mig.Version > currentVersion {
			pending = append(pending, mig)
		}
	}

	return pending, nil
}
