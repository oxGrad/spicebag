package db

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
