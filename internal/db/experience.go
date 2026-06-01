package db

type ExperienceEntry struct {
	ID         int64
	RoleType   string
	Company    string
	StartDate  string
	EndDate    string // empty = ongoing
	SyncedFrom string
}

func (s *Store) UpsertExperience(entries []ExperienceEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, e := range entries {
		_, err = tx.Exec(`
			INSERT INTO experience (role_type, company, start_date, end_date, synced_from)
			VALUES (?, ?, ?, ?, ?)`,
			e.RoleType, e.Company, e.StartDate, e.EndDate, e.SyncedFrom,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListExperience() ([]ExperienceEntry, error) {
	rows, err := s.db.Query(`SELECT id, role_type, company, start_date, end_date, synced_from FROM experience ORDER BY start_date`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ExperienceEntry
	for rows.Next() {
		var e ExperienceEntry
		if err := rows.Scan(&e.ID, &e.RoleType, &e.Company, &e.StartDate, &e.EndDate, &e.SyncedFrom); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) DeleteExperienceBySyncedFrom(filename string) error {
	_, err := s.db.Exec(`DELETE FROM experience WHERE synced_from = ?`, filename)
	return err
}
