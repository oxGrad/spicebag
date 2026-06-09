package db

import (
	"database/sql"
	"net/url"
	"strings"
)

type ScrapeCompany struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	ATSPlatform      string `json:"ats_platform"`
	ATSToken         string `json:"ats_token"`
	CareersURL       string `json:"careers_url"`
	LastScrapedAt    string `json:"last_scraped_at"`
	LastScrapeStatus string `json:"last_scrape_status"`
	LastScrapeError  string `json:"last_scrape_error"`
	LastJobCount     int    `json:"last_job_count"`
}

type ScrapeRole struct {
	ID      int64  `json:"id"`
	Keyword string `json:"keyword"`
}

type ScrapeSkill struct {
	ID      int64  `json:"id"`
	Keyword string `json:"keyword"`
}

type ScrapePrefs struct {
	HomeTimezone  string `json:"home_timezone"`
	LocationNotes string `json:"location_notes"`
}

func (s *Store) AddScrapeCompany(c ScrapeCompany) (ScrapeCompany, error) {
	res, err := s.db.Exec(
		`INSERT INTO scrape_companies (name, ats_platform, ats_token, careers_url)
		 VALUES (?, ?, ?, ?)`,
		c.Name, c.ATSPlatform, c.ATSToken, c.CareersURL,
	)
	if err != nil {
		return ScrapeCompany{}, err
	}
	id, _ := res.LastInsertId()
	c.ID = id
	c.LastScrapeStatus = "never"
	return c, nil
}

func (s *Store) ListScrapeCompanies() ([]ScrapeCompany, error) {
	rows, err := s.db.Query(
		`SELECT id, name, ats_platform, ats_token, careers_url,
		        last_scraped_at, last_scrape_status, last_scrape_error, last_job_count
		 FROM scrape_companies ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ScrapeCompany
	for rows.Next() {
		var c ScrapeCompany
		if err := rows.Scan(&c.ID, &c.Name, &c.ATSPlatform, &c.ATSToken, &c.CareersURL,
			&c.LastScrapedAt, &c.LastScrapeStatus, &c.LastScrapeError, &c.LastJobCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) UpdateScrapeCompanyStatus(id int64, scrapedAt, status, errMsg string, jobCount int) error {
	_, err := s.db.Exec(
		`UPDATE scrape_companies
		 SET last_scraped_at=?, last_scrape_status=?, last_scrape_error=?, last_job_count=?
		 WHERE id=?`,
		scrapedAt, status, errMsg, jobCount, id,
	)
	return err
}

func (s *Store) DeleteScrapeCompany(id int64) error {
	// Manual cascade (SQLite FK enforcement is off): remove the company's jobs first.
	if _, err := s.db.Exec(`DELETE FROM scraped_jobs WHERE company_id=?`, id); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM scrape_companies WHERE id=?`, id)
	return err
}

func (s *Store) AddScrapeRole(keyword string) (ScrapeRole, error) {
	res, err := s.db.Exec(`INSERT INTO scrape_roles (keyword) VALUES (?)`, keyword)
	if err != nil {
		return ScrapeRole{}, err
	}
	id, _ := res.LastInsertId()
	return ScrapeRole{ID: id, Keyword: keyword}, nil
}

func (s *Store) ListScrapeRoles() ([]ScrapeRole, error) {
	rows, err := s.db.Query(`SELECT id, keyword FROM scrape_roles ORDER BY keyword`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ScrapeRole
	for rows.Next() {
		var r ScrapeRole
		if err := rows.Scan(&r.ID, &r.Keyword); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DeleteScrapeRole(id int64) error {
	_, err := s.db.Exec(`DELETE FROM scrape_roles WHERE id=?`, id)
	return err
}

func (s *Store) AddScrapeSkill(keyword string) (ScrapeSkill, error) {
	res, err := s.db.Exec(`INSERT INTO scrape_skills (keyword) VALUES (?)`, keyword)
	if err != nil {
		return ScrapeSkill{}, err
	}
	id, _ := res.LastInsertId()
	return ScrapeSkill{ID: id, Keyword: keyword}, nil
}

func (s *Store) ListScrapeSkills() ([]ScrapeSkill, error) {
	rows, err := s.db.Query(`SELECT id, keyword FROM scrape_skills ORDER BY keyword`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ScrapeSkill
	for rows.Next() {
		var sk ScrapeSkill
		if err := rows.Scan(&sk.ID, &sk.Keyword); err != nil {
			return nil, err
		}
		out = append(out, sk)
	}
	return out, rows.Err()
}

func (s *Store) DeleteScrapeSkill(id int64) error {
	_, err := s.db.Exec(`DELETE FROM scrape_skills WHERE id=?`, id)
	return err
}

func (s *Store) GetScrapePrefs() (ScrapePrefs, error) {
	var p ScrapePrefs
	err := s.db.QueryRow(
		`SELECT home_timezone, location_notes FROM scrape_prefs WHERE id=1`,
	).Scan(&p.HomeTimezone, &p.LocationNotes)
	return p, err
}

func (s *Store) UpdateScrapePrefs(p ScrapePrefs) error {
	_, err := s.db.Exec(
		`UPDATE scrape_prefs SET home_timezone=?, location_notes=? WHERE id=1`,
		p.HomeTimezone, p.LocationNotes,
	)
	return err
}

type ScrapedJob struct {
	ID                   int64         `json:"id"`
	CompanyID            int64         `json:"company_id"`
	CompanyName          string        `json:"company_name"`
	Title                string        `json:"title"`
	Location             string        `json:"location"`
	URL                  string        `json:"url"`
	MatchReason          string        `json:"match_reason"`
	MatchedSkills        string        `json:"matched_skills"`
	SkillScore           int           `json:"skill_score"`
	Status               string        `json:"status"`
	ScrapedAt            string        `json:"scraped_at"`
	AppliedApplicationID sql.NullInt64 `json:"applied_application_id"`
}

// SaveScrapedJobs inserts jobs, ignoring any whose URL already exists.
// Returns the count of newly inserted rows.
func (s *Store) SaveScrapedJobs(jobs []ScrapedJob) (int, error) {
	added := 0
	for _, j := range jobs {
		res, err := s.db.Exec(
			`INSERT OR IGNORE INTO scraped_jobs
			   (company_id, company_name, title, location, url, match_reason, matched_skills, skill_score)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			j.CompanyID, j.CompanyName, j.Title, j.Location, j.URL, j.MatchReason, j.MatchedSkills, j.SkillScore,
		)
		if err != nil {
			return added, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			added++
		}
	}
	return added, nil
}

func (s *Store) ListScrapedJobs(status string) ([]ScrapedJob, error) {
	rows, err := s.db.Query(
		`SELECT id, company_id, company_name, title, location, url, match_reason,
		        matched_skills, skill_score, status, scraped_at, applied_application_id
		 FROM scraped_jobs WHERE status=?
		 ORDER BY skill_score DESC, scraped_at DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ScrapedJob
	for rows.Next() {
		var j ScrapedJob
		if err := rows.Scan(&j.ID, &j.CompanyID, &j.CompanyName, &j.Title, &j.Location,
			&j.URL, &j.MatchReason, &j.MatchedSkills, &j.SkillScore,
			&j.Status, &j.ScrapedAt, &j.AppliedApplicationID); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) SetScrapedJobStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE scraped_jobs SET status=? WHERE id=?`, status, id)
	return err
}

// LinkApplicationToScrapedJob finds a scraped job whose normalized URL matches
// the application's normalized job URL, links them, and marks it applied.
// Returns true if a match was found and linked.
func (s *Store) LinkApplicationToScrapedJob(appID int64, jobURL string) (bool, error) {
	norm := NormalizeJobURL(jobURL)
	if norm == "" {
		return false, nil
	}
	// Full scan + Go-side normalization (not WHERE url = ?) because the stored
	// scraped-job URL and the application's job URL may differ by query string
	// or trailing slash; only their normalized forms are guaranteed to match.
	rows, err := s.db.Query(`SELECT id, url FROM scraped_jobs WHERE status != 'applied'`)
	if err != nil {
		return false, err
	}
	var matchID int64
	for rows.Next() {
		var id int64
		var u string
		if err := rows.Scan(&id, &u); err != nil {
			rows.Close()
			return false, err
		}
		if NormalizeJobURL(u) == norm {
			matchID = id
			break
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, err
	}
	if matchID == 0 {
		return false, nil
	}
	_, err = s.db.Exec(
		`UPDATE scraped_jobs SET applied_application_id=?, status='applied' WHERE id=?`,
		appID, matchID,
	)
	return err == nil, err
}

// NormalizeJobURL strips query/fragment, trailing slash, and lowercases the host.
func NormalizeJobURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	p := strings.TrimRight(u.Path, "/")
	return u.Scheme + "://" + u.Host + p
}
