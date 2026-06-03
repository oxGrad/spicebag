package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/oxGrad/spicebag/internal/db"
)

func (s *Server) handleAPIQuestionsList(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	qs, err := s.store.ListQuestions(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if qs == nil {
		qs = []db.ApplicationQuestion{}
	}
	writeJSON(w, qs)
}

func (s *Server) handleAPIQuestionsAdd(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	question := r.FormValue("question")
	if question == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}
	posStr := r.FormValue("position")
	pos, _ := strconv.Atoi(posStr)

	q, err := s.store.AddQuestion(id, question, pos)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, q)
}

func (s *Server) handleAPIQuestionUpdate(w http.ResponseWriter, r *http.Request) {
	qid, ok := parseID(r, "qid")
	if !ok {
		http.NotFound(w, r)
		return
	}
	answer := r.FormValue("answer")
	if err := s.store.UpdateAnswer(qid, answer); err != nil {
		http.Error(w, fmt.Sprintf("update answer: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIQuestionBulletsUpdate(w http.ResponseWriter, r *http.Request) {
	qid, ok := parseID(r, "qid")
	if !ok {
		http.NotFound(w, r)
		return
	}
	var bullets []string
	if err := json.NewDecoder(r.Body).Decode(&bullets); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if bullets == nil {
		bullets = []string{}
	}
	if err := s.store.UpdateBullets(qid, bullets); err != nil {
		http.Error(w, fmt.Sprintf("update bullets: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIQuestionDelete(w http.ResponseWriter, r *http.Request) {
	qid, ok := parseID(r, "qid")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := s.store.DeleteQuestion(qid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type bulkAnswerEntry struct {
	QuestionID int64  `json:"question_id"`
	Answer     string `json:"answer"`
}

func (s *Server) handleAPIQuestionsBulkAnswer(w http.ResponseWriter, r *http.Request) {
	var entries []bulkAnswerEntry
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	pairs := make([]struct{ QuestionID int64; Answer string }, len(entries))
	for i, e := range entries {
		pairs[i] = struct{ QuestionID int64; Answer string }{e.QuestionID, e.Answer}
	}
	if err := s.store.BulkUpdateAnswers(pairs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
