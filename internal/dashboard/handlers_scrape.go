package dashboard

import (
	"net/http"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/oxGrad/spicebag/internal/scrape"
)

func (s *Server) handleAPIScrapeCompaniesList(w http.ResponseWriter, r *http.Request) {
	cs, err := s.store.ListScrapeCompanies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if cs == nil {
		cs = []db.ScrapeCompany{}
	}
	writeJSON(w, cs)
}

func (s *Server) handleAPIScrapeCompanyCreate(w http.ResponseWriter, r *http.Request) {
	careersURL := r.FormValue("careers_url")
	name := r.FormValue("name")
	if careersURL == "" {
		http.Error(w, "careers_url is required", http.StatusBadRequest)
		return
	}
	platform, token, err := scrape.Detect(careersURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if name == "" {
		name = token
	}
	c, err := s.store.AddScrapeCompany(db.ScrapeCompany{
		Name: name, ATSPlatform: platform, ATSToken: token, CareersURL: careersURL,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, c)
}

func (s *Server) handleAPIScrapeCompanyDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteScrapeCompany(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIScrapeRolesList(w http.ResponseWriter, r *http.Request) {
	rs, err := s.store.ListScrapeRoles()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rs == nil {
		rs = []db.ScrapeRole{}
	}
	writeJSON(w, rs)
}

func (s *Server) handleAPIScrapeRoleCreate(w http.ResponseWriter, r *http.Request) {
	keyword := r.FormValue("keyword")
	if keyword == "" {
		http.Error(w, "keyword is required", http.StatusBadRequest)
		return
	}
	role, err := s.store.AddScrapeRole(keyword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, role)
}

func (s *Server) handleAPIScrapeRoleDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteScrapeRole(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIScrapePrefsGet(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.GetScrapePrefs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, p)
}

func (s *Server) handleAPIScrapePrefsUpdate(w http.ResponseWriter, r *http.Request) {
	err := s.store.UpdateScrapePrefs(db.ScrapePrefs{
		HomeTimezone:  r.FormValue("home_timezone"),
		LocationNotes: r.FormValue("location_notes"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIScrapeJobsList(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "new"
	}
	jobs, err := s.store.ListScrapedJobs(status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if jobs == nil {
		jobs = []db.ScrapedJob{}
	}
	writeJSON(w, jobs)
}

func (s *Server) handleAPIScrapeJobStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	status := r.FormValue("status")
	switch status {
	case "new", "dismissed":
	default:
		http.Error(w, "status must be new or dismissed", http.StatusBadRequest)
		return
	}
	if err := s.store.SetScrapedJobStatus(id, status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIScrapeSkillsList(w http.ResponseWriter, r *http.Request) {
	sks, err := s.store.ListScrapeSkills()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if sks == nil {
		sks = []db.ScrapeSkill{}
	}
	writeJSON(w, sks)
}

func (s *Server) handleAPIScrapeSkillCreate(w http.ResponseWriter, r *http.Request) {
	keyword := r.FormValue("keyword")
	if keyword == "" {
		http.Error(w, "keyword is required", http.StatusBadRequest)
		return
	}
	sk, err := s.store.AddScrapeSkill(keyword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, sk)
}

func (s *Server) handleAPIScrapeSkillDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteScrapeSkill(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
