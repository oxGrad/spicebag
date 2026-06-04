// internal/memory/db.go
package memory

import (
	"database/sql"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Memory struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DB struct {
	db *sql.DB
}

var schema = []string{
	`CREATE TABLE IF NOT EXISTS memories (
		name        TEXT PRIMARY KEY,
		type        TEXT NOT NULL CHECK(type IN ('user','feedback','project','reference')),
		description TEXT NOT NULL,
		body        TEXT NOT NULL,
		created_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
		updated_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
	)`,
	`CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
		name, description, body,
		content='memories',
		content_rowid='rowid'
	)`,
	`CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
		INSERT INTO memories_fts(rowid, name, description, body)
		VALUES (new.rowid, new.name, new.description, new.body);
	END`,
	`CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, name, description, body)
		VALUES ('delete', old.rowid, old.name, old.description, old.body);
	END`,
	`CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, name, description, body)
		VALUES ('delete', old.rowid, old.name, old.description, old.body);
		INSERT INTO memories_fts(rowid, name, description, body)
		VALUES (new.rowid, new.name, new.description, new.body);
	END`,
}

func Open(path string) (*DB, error) {
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	sqldb.SetMaxOpenConns(1)
	for _, stmt := range schema {
		if _, err := sqldb.Exec(stmt); err != nil {
			sqldb.Close()
			return nil, err
		}
	}
	return &DB{db: sqldb}, nil
}

func (d *DB) Close() { d.db.Close() }

func (d *DB) Write(m *Memory) error {
	_, err := d.db.Exec(`
		INSERT INTO memories (name, type, description, body, updated_at)
		VALUES (?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		ON CONFLICT(name) DO UPDATE SET
			type        = excluded.type,
			description = excluded.description,
			body        = excluded.body,
			updated_at  = excluded.updated_at
	`, m.Name, m.Type, m.Description, m.Body)
	return err
}

func (d *DB) Delete(name string) error {
	_, err := d.db.Exec(`DELETE FROM memories WHERE name = ?`, name)
	return err
}

func (d *DB) GetByName(name string) (*Memory, error) {
	var m Memory
	var createdStr, updatedStr string
	err := d.db.QueryRow(`
		SELECT name, type, description, body, created_at, updated_at
		FROM memories WHERE name = ?
	`, name).Scan(&m.Name, &m.Type, &m.Description, &m.Body, &createdStr, &updatedStr)
	if err != nil {
		return nil, err
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &m, nil
}

func (d *DB) List() ([]*Memory, error) {
	rows, err := d.db.Query(`
		SELECT name, type, description, body, created_at, updated_at
		FROM memories ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemories(rows)
}

var wordRE = regexp.MustCompile(`\w+`)

func (d *DB) Search(query string, limit int) ([]*Memory, error) {
	words := wordRE.FindAllString(query, -1)
	if len(words) == 0 {
		return nil, nil
	}
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = `"` + strings.ReplaceAll(w, `"`, `""`) + `"*`
	}
	ftsQuery := strings.Join(quoted, " OR ")
	rows, err := d.db.Query(`
		SELECT m.name, m.type, m.description, m.body, m.created_at, m.updated_at
		FROM memories m
		JOIN memories_fts ON m.rowid = memories_fts.rowid
		WHERE memories_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemories(rows)
}

func scanMemories(rows *sql.Rows) ([]*Memory, error) {
	var out []*Memory
	for rows.Next() {
		var m Memory
		var createdStr, updatedStr string
		if err := rows.Scan(&m.Name, &m.Type, &m.Description, &m.Body, &createdStr, &updatedStr); err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		out = append(out, &m)
	}
	return out, rows.Err()
}
