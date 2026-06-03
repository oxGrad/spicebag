ALTER TABLE applications ADD COLUMN source      TEXT NOT NULL DEFAULT '';
ALTER TABLE applications ADD COLUMN job_url     TEXT NOT NULL DEFAULT '';
ALTER TABLE applications ADD COLUMN job_summary TEXT NOT NULL DEFAULT '';
