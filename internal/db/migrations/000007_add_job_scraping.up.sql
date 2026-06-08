CREATE TABLE scrape_companies (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  name               TEXT NOT NULL,
  ats_platform       TEXT NOT NULL,
  ats_token          TEXT NOT NULL,
  careers_url        TEXT NOT NULL,
  last_scraped_at    TEXT NOT NULL DEFAULT '',
  last_scrape_status TEXT NOT NULL DEFAULT 'never',
  last_scrape_error  TEXT NOT NULL DEFAULT '',
  last_job_count     INTEGER NOT NULL DEFAULT 0,
  UNIQUE(ats_platform, ats_token)
);

CREATE TABLE scrape_roles (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  keyword TEXT NOT NULL UNIQUE
);

CREATE TABLE scrape_prefs (
  id             INTEGER PRIMARY KEY CHECK (id = 1),
  home_timezone  TEXT NOT NULL DEFAULT '',
  location_notes TEXT NOT NULL DEFAULT ''
);

INSERT INTO scrape_prefs (id, home_timezone, location_notes) VALUES (1, '', '');

CREATE TABLE scraped_jobs (
  id                     INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id             INTEGER NOT NULL REFERENCES scrape_companies(id) ON DELETE CASCADE,
  company_name           TEXT NOT NULL,
  title                  TEXT NOT NULL,
  location               TEXT NOT NULL DEFAULT '',
  url                    TEXT NOT NULL UNIQUE,
  match_reason           TEXT NOT NULL DEFAULT '',
  status                 TEXT NOT NULL DEFAULT 'new',
  scraped_at             TEXT NOT NULL DEFAULT (datetime('now')),
  applied_application_id INTEGER REFERENCES applications(id)
);
