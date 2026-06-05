-- SQLite does not support ALTER TABLE to modify CHECK constraints,
-- so we recreate application_status_history with 'pending' added.
CREATE TABLE application_status_history_new (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  application_id INTEGER NOT NULL REFERENCES applications(id),
  status         TEXT NOT NULL CHECK(status IN ('pending','applied','assessment','interview','offer','rejected','withdrawn','ghosted','skipped')),
  changed_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  notes          TEXT
);

INSERT INTO application_status_history_new
  SELECT id, application_id, status, changed_at, notes
  FROM application_status_history;

DROP TABLE application_status_history;

ALTER TABLE application_status_history_new RENAME TO application_status_history;
