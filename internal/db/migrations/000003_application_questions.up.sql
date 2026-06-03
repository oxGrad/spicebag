CREATE TABLE IF NOT EXISTS application_questions (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  application_id INTEGER NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
  question       TEXT NOT NULL,
  answer         TEXT NOT NULL DEFAULT '',
  position       INTEGER NOT NULL DEFAULT 0,
  created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
