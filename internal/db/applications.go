package db

import "time"

type Application struct {
	ID          int64
	Company     string
	Role        string
	AppliedDate string
	BaseCVUsed  string
	Notes       string
	FolderPath  string
}

type StatusHistoryEntry struct {
	ID            int64
	ApplicationID int64
	Status        string
	ChangedAt     time.Time
	Notes         string
}

type ExperienceStats struct {
	Total  float64
	ByRole map[string]float64
}

func (s *Store) UpsertApplication(app Application) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO applications (company, role, applied_date, base_cv_used, notes, folder_path)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(folder_path) DO UPDATE SET
			company=excluded.company, role=excluded.role,
			applied_date=excluded.applied_date, base_cv_used=excluded.base_cv_used,
			notes=excluded.notes`,
		app.Company, app.Role, app.AppliedDate, app.BaseCVUsed, app.Notes, app.FolderPath,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListApplications() ([]Application, error) {
	rows, err := s.db.Query(`SELECT id, company, role, applied_date, base_cv_used, notes, folder_path FROM applications ORDER BY applied_date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []Application
	for rows.Next() {
		var a Application
		if err := rows.Scan(&a.ID, &a.Company, &a.Role, &a.AppliedDate, &a.BaseCVUsed, &a.Notes, &a.FolderPath); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (s *Store) AddStatusHistory(applicationID int64, status, notes string) error {
	_, err := s.db.Exec(`INSERT INTO application_status_history (application_id, status, notes) VALUES (?, ?, ?)`,
		applicationID, status, notes)
	return err
}

func (s *Store) GetStatusHistory(applicationID int64) ([]StatusHistoryEntry, error) {
	rows, err := s.db.Query(`SELECT id, application_id, status, changed_at, notes FROM application_status_history WHERE application_id = ? ORDER BY changed_at`,
		applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []StatusHistoryEntry
	for rows.Next() {
		var h StatusHistoryEntry
		if err := rows.Scan(&h.ID, &h.ApplicationID, &h.Status, &h.ChangedAt, &h.Notes); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

// ApplicationWithStatus is an Application plus its most recent status.
type ApplicationWithStatus struct {
	Application
	CurrentStatus string
}

// ListApplicationsWithStatus returns all applications with their latest status.
func (s *Store) ListApplicationsWithStatus() ([]ApplicationWithStatus, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.company, a.role, a.applied_date, a.base_cv_used, a.notes, a.folder_path,
		       COALESCE(
		         (SELECT status FROM application_status_history
		          WHERE application_id = a.id
		          ORDER BY changed_at DESC, id DESC LIMIT 1),
		         'unknown'
		       ) AS current_status
		FROM applications a
		ORDER BY a.applied_date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []ApplicationWithStatus
	for rows.Next() {
		var a ApplicationWithStatus
		if err := rows.Scan(
			&a.ID, &a.Company, &a.Role, &a.AppliedDate, &a.BaseCVUsed, &a.Notes, &a.FolderPath,
			&a.CurrentStatus,
		); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

// GetApplicationByID returns a single application by its primary key.
func (s *Store) GetApplicationByID(id int64) (Application, error) {
	var a Application
	err := s.db.QueryRow(
		`SELECT id, company, role, applied_date, base_cv_used, notes, folder_path FROM applications WHERE id = ?`,
		id,
	).Scan(&a.ID, &a.Company, &a.Role, &a.AppliedDate, &a.BaseCVUsed, &a.Notes, &a.FolderPath)
	return a, err
}

func (s *Store) GetExperienceStats() (ExperienceStats, error) {
	entries, err := s.ListExperience()
	if err != nil {
		return ExperienceStats{}, err
	}

	now := time.Now()
	stats := ExperienceStats{ByRole: make(map[string]float64)}

	for _, e := range entries {
		start, err := time.Parse("2006-01-02", e.StartDate)
		if err != nil {
			continue
		}
		end := now
		if e.EndDate != "" {
			end, err = time.Parse("2006-01-02", e.EndDate)
			if err != nil {
				continue
			}
		}
		years := end.Sub(start).Hours() / (24 * 365.25)
		stats.ByRole[e.RoleType] += years
		stats.Total += years
	}
	return stats, nil
}
