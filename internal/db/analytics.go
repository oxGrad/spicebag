package db

type MonthCount struct {
	Month string `json:"month"` // YYYY-MM
	Count int    `json:"count"`
}

type SourceStat struct {
	Source      string  `json:"source"`
	Total       int     `json:"total"`
	Responded   int     `json:"responded"`   // reached assessment/interview/offer/rejected
	Interviewed int     `json:"interviewed"` // reached interview or offer
	Offered     int     `json:"offered"`
	DropRate    float64 `json:"drop_rate"` // % that never moved past applied/unknown/ghosted
}

func (s *Store) ApplicationsPerMonth() ([]MonthCount, error) {
	rows, err := s.db.Query(`
		SELECT strftime('%Y-%m', applied_date) AS month, COUNT(*) AS count
		FROM applications
		WHERE id NOT IN (
			SELECT DISTINCT application_id FROM application_status_history
			WHERE status = 'skipped'
		)
		GROUP BY month
		ORDER BY month DESC
		LIMIT 12
	`)
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
	// Return in chronological order for charts.
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, rows.Err()
}

func (s *Store) SourceStats() ([]SourceStat, error) {
	rows, err := s.db.Query(`
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
		)
		WHERE cs != 'skipped'
		GROUP BY src
		ORDER BY total DESC
	`)
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
