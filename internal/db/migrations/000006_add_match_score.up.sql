ALTER TABLE applications ADD COLUMN match_score INTEGER;
ALTER TABLE applications ADD COLUMN tailoring_notes TEXT NOT NULL DEFAULT '';
