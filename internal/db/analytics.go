package db

type MonthCount struct {
	Month string `json:"month"` // YYYY-MM
	Count int    `json:"count"`
}

type DayCount struct {
	Day   string `json:"day"`   // YYYY-MM-DD
	Count int    `json:"count"`
}

// ApplicationsPerDay returns application counts grouped by day.
// from and to are optional YYYY-MM strings; empty = no constraint.
func (s *Store) ApplicationsPerDay(from, to string) ([]DayCount, error) {
	q := `
		SELECT strftime('%Y-%m-%d', applied_date) AS day, COUNT(*) AS count
		FROM applications
		WHERE applied_date != ''
		AND id NOT IN (
			SELECT DISTINCT application_id FROM application_status_history
			WHERE status = 'skipped'
		)
	`
	var args []any
	if from != "" {
		q += ` AND strftime('%Y-%m', applied_date) >= ?`
		args = append(args, from)
	}
	if to != "" {
		q += ` AND strftime('%Y-%m', applied_date) <= ?`
		args = append(args, to)
	}
	q += ` GROUP BY day ORDER BY day ASC`

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DayCount
	for rows.Next() {
		var d DayCount
		if err := rows.Scan(&d.Day, &d.Count); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

type SourceStat struct {
	Source      string  `json:"source"`
	Total       int     `json:"total"`
	Responded   int     `json:"responded"`   // reached assessment/interview/offer/rejected
	Interviewed int     `json:"interviewed"` // reached interview or offer
	Offered     int     `json:"offered"`
	DropRate    float64 `json:"drop_rate"` // % that never moved past applied/unknown/ghosted
}

// ApplicationsPerMonth returns application counts grouped by month.
// from and to are optional YYYY-MM strings; empty = no constraint.
func (s *Store) ApplicationsPerMonth(from, to string) ([]MonthCount, error) {
	q := `
		SELECT strftime('%Y-%m', applied_date) AS month, COUNT(*) AS count
		FROM applications
		WHERE id NOT IN (
			SELECT DISTINCT application_id FROM application_status_history
			WHERE status = 'skipped'
		)
	`
	var args []any
	if from != "" {
		q += ` AND strftime('%Y-%m', applied_date) >= ?`
		args = append(args, from)
	}
	if to != "" {
		q += ` AND strftime('%Y-%m', applied_date) <= ?`
		args = append(args, to)
	}
	q += ` GROUP BY month ORDER BY month ASC`

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MonthCount
	for rows.Next() {
		var m MonthCount
		if err := rows.Scan(&m.Month, &m.Count); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// SourceStats returns per-source funnel statistics.
// from and to are optional YYYY-MM strings; empty = no constraint.
func (s *Store) SourceStats(from, to string) ([]SourceStat, error) {
	dateFilter := ""
	var args []any
	if from != "" {
		dateFilter += ` AND strftime('%Y-%m', a.applied_date) >= ?`
		args = append(args, from)
	}
	if to != "" {
		dateFilter += ` AND strftime('%Y-%m', a.applied_date) <= ?`
		args = append(args, to)
	}

	q := `
		SELECT
			CASE WHEN source = '' THEN 'Unknown' ELSE source END AS src,
			COUNT(*) AS total,
			SUM(CASE WHEN cs IN ('assessment','interview','offer','rejected') THEN 1 ELSE 0 END) AS responded,
			SUM(CASE WHEN cs IN ('interview','offer') THEN 1 ELSE 0 END) AS interviewed,
			SUM(CASE WHEN cs = 'offer' THEN 1 ELSE 0 END) AS offered
		FROM (
			SELECT a.source,
				COALESCE(
					(SELECT status FROM application_status_history
					 WHERE application_id = a.id
					 ORDER BY changed_at DESC, id DESC LIMIT 1),
					'unknown'
				) AS cs
			FROM applications a
			WHERE 1=1` + dateFilter + `
		)
		WHERE cs != 'skipped'
		GROUP BY src
		ORDER BY total DESC
	`

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SourceStat
	for rows.Next() {
		var st SourceStat
		if err := rows.Scan(&st.Source, &st.Total, &st.Responded, &st.Interviewed, &st.Offered); err != nil {
			return nil, err
		}
		if st.Total > 0 {
			st.DropRate = float64(st.Total-st.Responded) / float64(st.Total) * 100
		}
		result = append(result, st)
	}
	return result, rows.Err()
}
