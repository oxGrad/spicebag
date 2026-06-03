package db

import (
	"encoding/json"
	"time"
)

type ApplicationQuestion struct {
	ID            int64     `json:"id"`
	ApplicationID int64     `json:"application_id"`
	Question      string    `json:"question"`
	Answer        string    `json:"answer"`
	UserBullets   []string  `json:"user_bullets"`
	Position      int       `json:"position"`
	CreatedAt     time.Time `json:"created_at"`
}

func scanQuestion(scan func(...any) error) (ApplicationQuestion, error) {
	var q ApplicationQuestion
	var bulletsJSON string
	err := scan(&q.ID, &q.ApplicationID, &q.Question, &q.Answer, &bulletsJSON, &q.Position, &q.CreatedAt)
	if err != nil {
		return q, err
	}
	if err := json.Unmarshal([]byte(bulletsJSON), &q.UserBullets); err != nil || q.UserBullets == nil {
		q.UserBullets = []string{}
	}
	return q, nil
}

func (s *Store) ListQuestions(applicationID int64) ([]ApplicationQuestion, error) {
	rows, err := s.db.Query(`
		SELECT id, application_id, question, answer, user_bullets, position, created_at
		FROM application_questions
		WHERE application_id = ?
		ORDER BY position, id`, applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var qs []ApplicationQuestion
	for rows.Next() {
		q, err := scanQuestion(rows.Scan)
		if err != nil {
			return nil, err
		}
		qs = append(qs, q)
	}
	return qs, rows.Err()
}

func (s *Store) AddQuestion(applicationID int64, question string, position int) (ApplicationQuestion, error) {
	res, err := s.db.Exec(`
		INSERT INTO application_questions (application_id, question, position)
		VALUES (?, ?, ?)`, applicationID, question, position)
	if err != nil {
		return ApplicationQuestion{}, err
	}
	id, _ := res.LastInsertId()
	q := ApplicationQuestion{
		ID: id, ApplicationID: applicationID,
		Question: question, Position: position,
		UserBullets: []string{},
	}
	return q, nil
}

func (s *Store) UpdateAnswer(questionID int64, answer string) error {
	_, err := s.db.Exec(`UPDATE application_questions SET answer = ? WHERE id = ?`, answer, questionID)
	return err
}

func (s *Store) UpdateBullets(questionID int64, bullets []string) error {
	b, err := json.Marshal(bullets)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE application_questions SET user_bullets = ? WHERE id = ?`, string(b), questionID)
	return err
}

func (s *Store) DeleteQuestion(questionID int64) error {
	_, err := s.db.Exec(`DELETE FROM application_questions WHERE id = ?`, questionID)
	return err
}

func (s *Store) BulkUpdateAnswers(answers []struct{ QuestionID int64; Answer string }) error {
	for _, a := range answers {
		if err := s.UpdateAnswer(a.QuestionID, a.Answer); err != nil {
			return err
		}
	}
	return nil
}
