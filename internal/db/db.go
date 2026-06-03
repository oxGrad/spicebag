package db

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS experience (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  role_type   TEXT NOT NULL,
  company     TEXT NOT NULL,
  start_date  TEXT NOT NULL,
  end_date    TEXT,
  synced_from TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS applications (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  company      TEXT NOT NULL,
  role         TEXT NOT NULL,
  applied_date TEXT NOT NULL,
  base_cv_used TEXT,
  notes        TEXT,
  folder_path  TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS application_status_history (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  application_id INTEGER NOT NULL REFERENCES applications(id),
  status         TEXT NOT NULL CHECK(status IN ('applied','assessment','interview','offer','rejected','withdrawn','ghosted')),
  changed_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  notes          TEXT
);

CREATE TABLE IF NOT EXISTS sources (
  id   INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);
`

// defaultSources are inserted once on first open.
var defaultSources = []string{"LinkedIn", "Indeed", "Greenhouse", "Lever", "Workday", "Referral", "Company website", "Other"}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	// Migrate: add columns for existing databases (no-op if already present).
	db.Exec(`ALTER TABLE applications ADD COLUMN source TEXT NOT NULL DEFAULT ''`)    //nolint:errcheck
	db.Exec(`ALTER TABLE applications ADD COLUMN job_url TEXT NOT NULL DEFAULT ''`)   //nolint:errcheck
	db.Exec(`ALTER TABLE applications ADD COLUMN job_summary TEXT NOT NULL DEFAULT ''`) //nolint:errcheck
	// Seed default sources (INSERT OR IGNORE so it's idempotent).
	for _, name := range defaultSources {
		db.Exec(`INSERT OR IGNORE INTO sources (name) VALUES (?)`, name) //nolint:errcheck
	}
	return &Store{db: db}, nil
}

func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) Close() error { return s.db.Close() }
