CREATE TABLE application_status_history_old (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  application_id INTEGER NOT NULL REFERENCES applications(id),
  status         TEXT NOT NULL CHECK(status IN ('applied','assessment','interview','offer','rejected','withdrawn','ghosted')),
  changed_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  notes          TEXT
);

INSERT INTO application_status_history_old
  SELECT id, application_id, status, changed_at, notes
  FROM application_status_history
  WHERE status != 'pending';

DROP TABLE application_status_history;

ALTER TABLE application_status_history_old RENAME TO application_status_history;
