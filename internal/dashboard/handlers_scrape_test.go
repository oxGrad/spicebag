package dashboard_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScrapeCompanyAddDetectsPlatform(t *testing.T) {
	srv := newTestServer(t)

	form := strings.NewReader("careers_url=https://boards.greenhouse.io/acme")
	req := httptest.NewRequest(http.MethodPost, "/api/scrape/companies", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "greenhouse")

	req2 := httptest.NewRequest(http.MethodGet, "/api/scrape/companies", nil)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "acme")
}

func TestScrapeCompanyAddRejectsUnsupported(t *testing.T) {
	srv := newTestServer(t)
	form := strings.NewReader("careers_url=https://acme.com/careers")
	req := httptest.NewRequest(http.MethodPost, "/api/scrape/companies", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestScrapeRoleAddAndPrefs(t *testing.T) {
	srv := newTestServer(t)

	roleReq := httptest.NewRequest(http.MethodPost, "/api/scrape/roles", strings.NewReader("keyword=SRE"))
	roleReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rw := httptest.NewRecorder()
	srv.ServeHTTP(rw, roleReq)
	assert.Equal(t, http.StatusCreated, rw.Code)

	prefReq := httptest.NewRequest(http.MethodPost, "/api/scrape/prefs",
		strings.NewReader("home_timezone=UTC%2B7&location_notes=APAC"))
	prefReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	pw := httptest.NewRecorder()
	srv.ServeHTTP(pw, prefReq)
	assert.Equal(t, http.StatusNoContent, pw.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/scrape/prefs", nil)
	gw := httptest.NewRecorder()
	srv.ServeHTTP(gw, getReq)
	assert.Contains(t, gw.Body.String(), "UTC+7")
}
