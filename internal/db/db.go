package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

//go:embed migrations
var migrationsFS embed.FS

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return &Store{db: db}, nil
}

func runMigrations(db *sql.DB) error {
	// Seed the migration version for pre-golang-migrate databases so that
	// already-applied DDL is not re-run.
	if err := seedMigrationStateIfNeeded(db); err != nil {
		return fmt.Errorf("seed migration state: %w", err)
	}

	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}

	driver, err := migratesqlite.WithInstance(db, &migratesqlite.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// detectExistingVersion inspects the schema to determine which migration the DB
// is effectively at, for databases created before golang-migrate was introduced.
// Returns 0 for a fresh (empty) database.
func detectExistingVersion(db *sql.DB) int {
	count := func(query string) int {
		var n int
		db.QueryRow(query).Scan(&n) //nolint:errcheck
		return n
	}

	// Migration 4: user_bullets column on application_questions
	if count(`SELECT COUNT(*) FROM pragma_table_info('application_questions') WHERE name='user_bullets'`) > 0 {
		return 4
	}
	// Migration 3: application_questions table
	if count(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='application_questions'`) > 0 {
		return 3
	}
	// Migration 2: source/job_url/job_summary columns on applications
	if count(`SELECT COUNT(*) FROM pragma_table_info('applications') WHERE name='source'`) > 0 {
		return 2
	}
	// Migration 1: base schema (applications table exists)
	if count(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='applications'`) > 0 {
		return 1
	}
	return 0 // fresh DB — let golang-migrate run all migrations
}

// seedMigrationStateIfNeeded creates the schema_migrations tracking table and
// marks the appropriate version for databases that predate golang-migrate.
// It is a no-op for fresh databases and for databases already tracked.
func seedMigrationStateIfNeeded(db *sql.DB) error {
	var hasTracking int
	db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'`).Scan(&hasTracking) //nolint:errcheck
	if hasTracking > 0 {
		return nil // already tracked by golang-migrate
	}

	version := detectExistingVersion(db)
	if version == 0 {
		return nil // fresh DB — golang-migrate will create the table itself
	}

	// Use the exact table structure that golang-migrate's sqlite driver creates.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version uint64, dirty bool)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS version_unique ON schema_migrations (version)`); err != nil {
		return err
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_migrations (version, dirty) VALUES (?, 0)`, version); err != nil {
		return err
	}
	return nil
}

func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) Close() error { return s.db.Close() }
