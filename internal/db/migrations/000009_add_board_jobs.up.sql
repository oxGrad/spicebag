CREATE TABLE scrape_boards (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  name               TEXT NOT NULL UNIQUE,
  label              TEXT NOT NULL,
  enabled            INTEGER NOT NULL DEFAULT 1,
  last_scraped_at    TEXT NOT NULL DEFAULT '',
  last_scrape_status TEXT NOT NULL DEFAULT 'never',
  last_scrape_error  TEXT NOT NULL DEFAULT '',
  last_job_count     INTEGER NOT NULL DEFAULT 0
);

INSERT INTO scrape_boards (name, label, enabled) VALUES
  ('remotive',       'Remotive',          1),
  ('remoteok',       'Remote OK',         1),
  ('weworkremotely', 'We Work Remotely',  1),
  ('jobicy',         'Jobicy',            1);

CREATE TABLE board_jobs (
  id                     INTEGER PRIMARY KEY AUTOINCREMENT,
  source_board           TEXT NOT NULL,
  company_name           TEXT NOT NULL,
  title                  TEXT NOT NULL,
  location               TEXT NOT NULL DEFAULT '',
  url                    TEXT NOT NULL UNIQUE,
  match_reason           TEXT NOT NULL DEFAULT '',
  matched_skills         TEXT NOT NULL DEFAULT '',
  skill_score            INTEGER NOT NULL DEFAULT 0,
  status                 TEXT NOT NULL DEFAULT 'new',
  scraped_at             TEXT NOT NULL DEFAULT (datetime('now')),
  applied_application_id INTEGER REFERENCES applications(id)
);
