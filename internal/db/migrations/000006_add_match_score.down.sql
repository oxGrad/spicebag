-- SQLite does not support DROP COLUMN before 3.35.0; recreate without match_score/tailoring_notes.
CREATE TABLE applications_old (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  company      TEXT NOT NULL,
  role         TEXT NOT NULL,
  applied_date TEXT NOT NULL,
  base_cv_used TEXT,
  notes        TEXT,
  folder_path  TEXT NOT NULL UNIQUE,
  source       TEXT NOT NULL DEFAULT '',
  job_url      TEXT NOT NULL DEFAULT '',
  job_summary  TEXT NOT NULL DEFAULT ''
);

INSERT INTO applications_old
  SELECT id, company, role, applied_date, base_cv_used, notes, folder_path, source, job_url, job_summary
  FROM applications;

DROP TABLE applications;

ALTER TABLE applications_old RENAME TO applications;
