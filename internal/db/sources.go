package db

type Source struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (s *Store) ListSources() ([]Source, error) {
	rows, err := s.db.Query(`SELECT id, name FROM sources ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		var src Source
		if err := rows.Scan(&src.ID, &src.Name); err != nil {
			return nil, err
		}
		sources = append(sources, src)
	}
	return sources, rows.Err()
}

func (s *Store) AddSource(name string) (Source, error) {
	res, err := s.db.Exec(`INSERT INTO sources (name) VALUES (?)`, name)
	if err != nil {
		return Source{}, err
	}
	id, _ := res.LastInsertId()
	return Source{ID: id, Name: name}, nil
}

func (s *Store) DeleteSource(id int64) error {
	_, err := s.db.Exec(`DELETE FROM sources WHERE id = ?`, id)
	return err
}
