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

INSERT OR IGNORE INTO sources (name) VALUES
  ('LinkedIn'),
  ('Indeed'),
  ('Greenhouse'),
  ('Lever'),
  ('Workday'),
  ('Referral'),
  ('Company website'),
  ('Other');
