package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Application struct {
	ID              int64
	Company         string
	Role            string
	AppliedDate     string
	BaseCVUsed      string
	Notes           string
	FolderPath      string
	Source          string
	JobURL          string
	JobSummary      string
	MatchScore      *int   // nil if not assessed
	TailoringNotes  string
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
		INSERT INTO applications (company, role, applied_date, base_cv_used, notes, folder_path, source, job_url, job_summary, match_score, tailoring_notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(folder_path) DO UPDATE SET
			company=excluded.company, role=excluded.role,
			applied_date=excluded.applied_date, base_cv_used=excluded.base_cv_used,
			notes=excluded.notes, source=excluded.source,
			job_url=excluded.job_url, job_summary=excluded.job_summary,
			match_score=excluded.match_score, tailoring_notes=excluded.tailoring_notes`,
		app.Company, app.Role, app.AppliedDate, app.BaseCVUsed, app.Notes, app.FolderPath, app.Source, app.JobURL, app.JobSummary, app.MatchScore, app.TailoringNotes,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListApplications() ([]Application, error) {
	rows, err := s.db.Query(`SELECT id, company, role, applied_date, base_cv_used, notes, folder_path, source, job_url, job_summary, match_score, tailoring_notes FROM applications ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []Application
	for rows.Next() {
		var a Application
		if err := rows.Scan(&a.ID, &a.Company, &a.Role, &a.AppliedDate, &a.BaseCVUsed, &a.Notes, &a.FolderPath, &a.Source, &a.JobURL, &a.JobSummary, &a.MatchScore, &a.TailoringNotes); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (s *Store) UpdateAppliedDate(id int64, date string) error {
	_, err := s.db.Exec(`UPDATE applications SET applied_date = ? WHERE id = ?`, date, id)
	return err
}

func (s *Store) UpdateApplicationSource(id int64, source string) error {
	_, err := s.db.Exec(`UPDATE applications SET source = ? WHERE id = ?`, source, id)
	return err
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
	FromScrape    bool
}

// ListApplicationsWithStatus returns all applications with their latest status.
func (s *Store) ListApplicationsWithStatus() ([]ApplicationWithStatus, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.company, a.role, a.applied_date, a.base_cv_used, a.notes, a.folder_path, a.source, a.job_url, a.job_summary, a.match_score, a.tailoring_notes,
		       COALESCE(
		         (SELECT status FROM application_status_history
		          WHERE application_id = a.id
		          ORDER BY changed_at DESC, id DESC LIMIT 1),
		         'unknown'
		       ) AS current_status,
		       EXISTS(SELECT 1 FROM scraped_jobs sj WHERE sj.applied_application_id = a.id) AS from_scrape
		FROM applications a
		ORDER BY a.id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []ApplicationWithStatus
	for rows.Next() {
		var a ApplicationWithStatus
		if err := rows.Scan(
			&a.ID, &a.Company, &a.Role, &a.AppliedDate, &a.BaseCVUsed, &a.Notes, &a.FolderPath, &a.Source, &a.JobURL, &a.JobSummary, &a.MatchScore, &a.TailoringNotes,
			&a.CurrentStatus, &a.FromScrape,
		); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

// ListApplicationsByBaseCV returns all applications that used the given CV filename as their base.
func (s *Store) ListApplicationsByBaseCV(baseCVName string) ([]ApplicationWithStatus, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.company, a.role, a.applied_date, a.base_cv_used, a.notes, a.folder_path, a.source, a.job_url, a.job_summary, a.match_score, a.tailoring_notes,
		       COALESCE(
		         (SELECT status FROM application_status_history
		          WHERE application_id = a.id
		          ORDER BY changed_at DESC, id DESC LIMIT 1),
		         'unknown'
		       ) AS current_status
		FROM applications a
		WHERE a.base_cv_used = ?
		ORDER BY a.id DESC
	`, baseCVName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []ApplicationWithStatus
	for rows.Next() {
		var a ApplicationWithStatus
		if err := rows.Scan(
			&a.ID, &a.Company, &a.Role, &a.AppliedDate, &a.BaseCVUsed, &a.Notes, &a.FolderPath, &a.Source, &a.JobURL, &a.JobSummary, &a.MatchScore, &a.TailoringNotes,
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
		`SELECT id, company, role, applied_date, base_cv_used, notes, folder_path, source, job_url, job_summary, match_score, tailoring_notes FROM applications WHERE id = ?`,
		id,
	).Scan(&a.ID, &a.Company, &a.Role, &a.AppliedDate, &a.BaseCVUsed, &a.Notes, &a.FolderPath, &a.Source, &a.JobURL, &a.JobSummary, &a.MatchScore, &a.TailoringNotes)
	if errors.Is(err, sql.ErrNoRows) {
		return Application{}, fmt.Errorf("application %d not found: %w", id, sql.ErrNoRows)
	}
	return a, err
}

// DeleteApplication removes an application and all its status history.
func (s *Store) DeleteApplication(id int64) error {
	_, err := s.db.Exec(`DELETE FROM application_status_history WHERE application_id = ?`, id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM applications WHERE id = ?`, id)
	return err
}

// parseFlexDate accepts YYYY-MM-DD or YYYY-MM.
func parseFlexDate(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01", s)
}

func (s *Store) GetExperienceStats() (ExperienceStats, error) {
	entries, err := s.ListExperience()
	if err != nil {
		return ExperienceStats{}, err
	}

	now := time.Now()
	stats := ExperienceStats{ByRole: make(map[string]float64)}

	for _, e := range entries {
		start, err := parseFlexDate(e.StartDate)
		if err != nil {
			continue
		}
		end := now
		if e.EndDate != "" {
			end, err = parseFlexDate(e.EndDate)
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
