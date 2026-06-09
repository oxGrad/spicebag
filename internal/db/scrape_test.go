package db_test

import (
	"testing"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrapeCompaniesCRUD(t *testing.T) {
	store := openTestStore(t)

	c, err := store.AddScrapeCompany(db.ScrapeCompany{
		Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme",
		CareersURL: "https://boards.greenhouse.io/acme",
	})
	require.NoError(t, err)
	assert.Greater(t, c.ID, int64(0))

	list, err := store.ListScrapeCompanies()
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Acme", list[0].Name)
	assert.Equal(t, "never", list[0].LastScrapeStatus)

	require.NoError(t, store.UpdateScrapeCompanyStatus(c.ID, "2026-06-08 10:00:00", "ok", "", 12))
	list, err = store.ListScrapeCompanies()
	require.NoError(t, err)
	assert.Equal(t, "ok", list[0].LastScrapeStatus)
	assert.Equal(t, 12, list[0].LastJobCount)

	require.NoError(t, store.DeleteScrapeCompany(c.ID))
	list, err = store.ListScrapeCompanies()
	require.NoError(t, err)
	assert.Len(t, list, 0)
}

func TestScrapeRolesCRUD(t *testing.T) {
	store := openTestStore(t)

	r, err := store.AddScrapeRole("SRE")
	require.NoError(t, err)
	assert.Greater(t, r.ID, int64(0))

	roles, err := store.ListScrapeRoles()
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "SRE", roles[0].Keyword)

	require.NoError(t, store.DeleteScrapeRole(r.ID))
	roles, err = store.ListScrapeRoles()
	require.NoError(t, err)
	assert.Len(t, roles, 0)
}

func TestScrapePrefs(t *testing.T) {
	store := openTestStore(t)

	prefs, err := store.GetScrapePrefs()
	require.NoError(t, err)
	assert.Equal(t, "", prefs.HomeTimezone)

	require.NoError(t, store.UpdateScrapePrefs(db.ScrapePrefs{
		HomeTimezone: "UTC+7", LocationNotes: "Accept anywhere; APAC; Indonesia",
	}))
	prefs, err = store.GetScrapePrefs()
	require.NoError(t, err)
	assert.Equal(t, "UTC+7", prefs.HomeTimezone)
	assert.Contains(t, prefs.LocationNotes, "APAC")
}

func TestScrapedJobsSaveAndList(t *testing.T) {
	store := openTestStore(t)
	c, _ := store.AddScrapeCompany(db.ScrapeCompany{
		Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme",
		CareersURL: "https://boards.greenhouse.io/acme",
	})

	jobs := []db.ScrapedJob{
		{CompanyID: c.ID, CompanyName: "Acme", Title: "SRE", Location: "Remote", URL: "https://j/1", MatchReason: "worldwide"},
		{CompanyID: c.ID, CompanyName: "Acme", Title: "DevOps", Location: "Remote APAC", URL: "https://j/2", MatchReason: "APAC"},
	}
	added, err := store.SaveScrapedJobs(jobs)
	require.NoError(t, err)
	assert.Equal(t, 2, added)

	added, err = store.SaveScrapedJobs(jobs)
	require.NoError(t, err)
	assert.Equal(t, 0, added)

	list, err := store.ListScrapedJobs("new")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestScrapedJobStatusUpdate(t *testing.T) {
	store := openTestStore(t)
	c, _ := store.AddScrapeCompany(db.ScrapeCompany{Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u"})
	store.SaveScrapedJobs([]db.ScrapedJob{{CompanyID: c.ID, CompanyName: "Acme", Title: "SRE", URL: "https://j/1"}})

	list, _ := store.ListScrapedJobs("new")
	require.Len(t, list, 1)
	require.NoError(t, store.SetScrapedJobStatus(list[0].ID, "dismissed"))

	assert.Len(t, mustList(t, store, "new"), 0)
	assert.Len(t, mustList(t, store, "dismissed"), 1)
}

func TestApplyLinkageByURL(t *testing.T) {
	store := openTestStore(t)
	c, _ := store.AddScrapeCompany(db.ScrapeCompany{Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u"})
	store.SaveScrapedJobs([]db.ScrapedJob{
		{CompanyID: c.ID, CompanyName: "Acme", Title: "SRE", URL: "https://boards.greenhouse.io/acme/jobs/42"},
	})

	appID, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "SRE", FolderPath: "/tmp/acme",
		JobURL: "https://boards.greenhouse.io/acme/jobs/42?utm_source=x",
	})
	require.NoError(t, err)

	linked, err := store.LinkApplicationToScrapedJob(appID, "https://boards.greenhouse.io/acme/jobs/42?utm_source=x")
	require.NoError(t, err)
	assert.True(t, linked)

	assert.Len(t, mustList(t, store, "new"), 0)
	applied := mustList(t, store, "applied")
	require.Len(t, applied, 1)
	assert.Equal(t, appID, applied[0].AppliedApplicationID.Int64)

	apps, err := store.ListApplicationsWithStatus()
	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.True(t, apps[0].FromScrape)
}

func TestScrapeSkillsCRUD(t *testing.T) {
	store := openTestStore(t)

	s, err := store.AddScrapeSkill("Go")
	require.NoError(t, err)
	assert.Greater(t, s.ID, int64(0))
	assert.Equal(t, "Go", s.Keyword)

	skills, err := store.ListScrapeSkills()
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, "Go", skills[0].Keyword)

	require.NoError(t, store.DeleteScrapeSkill(s.ID))
	skills, err = store.ListScrapeSkills()
	require.NoError(t, err)
	assert.Len(t, skills, 0)
}

func TestScrapedJobsSaveWithSkillScore(t *testing.T) {
	store := openTestStore(t)
	c, _ := store.AddScrapeCompany(db.ScrapeCompany{
		Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u",
	})

	jobs := []db.ScrapedJob{
		{
			CompanyID: c.ID, CompanyName: "Acme",
			Title: "Go SRE", Location: "Remote", URL: "https://j/1",
			MatchReason: "worldwide remote · Go", MatchedSkills: "Go", SkillScore: 1,
		},
		{
			CompanyID: c.ID, CompanyName: "Acme",
			Title: "Go Kubernetes Platform Engineer", Location: "Remote APAC", URL: "https://j/2",
			MatchReason: "APAC includes UTC+7 · Go,Kubernetes", MatchedSkills: "Go,Kubernetes", SkillScore: 2,
		},
	}
	added, err := store.SaveScrapedJobs(jobs)
	require.NoError(t, err)
	assert.Equal(t, 2, added)

	list, err := store.ListScrapedJobs("new")
	require.NoError(t, err)
	require.Len(t, list, 2)
	// higher skill_score sorts first
	assert.Equal(t, "Go Kubernetes Platform Engineer", list[0].Title)
	assert.Equal(t, "Go,Kubernetes", list[0].MatchedSkills)
	assert.Equal(t, 2, list[0].SkillScore)
	assert.Equal(t, "Go", list[1].MatchedSkills)
}

func mustList(t *testing.T, store *db.Store, status string) []db.ScrapedJob {
	t.Helper()
	l, err := store.ListScrapedJobs(status)
	require.NoError(t, err)
	return l
}

func TestDeleteScrapeCompanyCascadesJobs(t *testing.T) {
	store := openTestStore(t)
	c, err := store.AddScrapeCompany(db.ScrapeCompany{
		Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u",
	})
	require.NoError(t, err)
	// Insert a scraped job for this company directly.
	_, err = store.DB().Exec(
		`INSERT INTO scraped_jobs (company_id, company_name, title, url) VALUES (?, 'Acme', 'SRE', 'https://j/1')`,
		c.ID)
	require.NoError(t, err)

	require.NoError(t, store.DeleteScrapeCompany(c.ID))

	var n int
	require.NoError(t, store.DB().QueryRow(`SELECT COUNT(*) FROM scraped_jobs WHERE company_id=?`, c.ID).Scan(&n))
	assert.Equal(t, 0, n, "scraped_jobs for the company must be deleted")
}

func TestScrapeBoardsSeededAndToggle(t *testing.T) {
	store := openTestStore(t)

	boards, err := store.ListScrapeBoards()
	require.NoError(t, err)
	require.Len(t, boards, 4) // seeded by migration

	var remotiveID int64
	for _, b := range boards {
		if b.Name == "remotive" {
			remotiveID = b.ID
			assert.True(t, b.Enabled)
			assert.Equal(t, "Remotive", b.Label)
			assert.Equal(t, "never", b.LastScrapeStatus)
		}
	}
	require.NotZero(t, remotiveID)

	require.NoError(t, store.ToggleScrapeBoard(remotiveID, false))
	boards, _ = store.ListScrapeBoards()
	for _, b := range boards {
		if b.ID == remotiveID {
			assert.False(t, b.Enabled)
		}
	}

	require.NoError(t, store.UpdateScrapeBoardStatus(remotiveID, "2026-06-09 10:00:00", "ok", "", 42))
	boards, _ = store.ListScrapeBoards()
	for _, b := range boards {
		if b.ID == remotiveID {
			assert.Equal(t, "ok", b.LastScrapeStatus)
			assert.Equal(t, 42, b.LastJobCount)
		}
	}
}

func TestBoardJobsSaveAndList(t *testing.T) {
	store := openTestStore(t)

	jobs := []db.BoardJob{
		{SourceBoard: "remotive", CompanyName: "Acme", Title: "SRE", Location: "Worldwide", URL: "https://remotive.com/1", MatchReason: "worldwide remote", MatchedSkills: "Go", SkillScore: 1},
		{SourceBoard: "remoteok", CompanyName: "Beta", Title: "DevOps Engineer", Location: "Remote", URL: "https://remoteok.com/2", MatchReason: "worldwide remote"},
	}

	added, err := store.SaveBoardJobs(jobs)
	require.NoError(t, err)
	assert.Equal(t, 2, added)

	// duplicate URL is ignored
	added2, err := store.SaveBoardJobs(jobs[:1])
	require.NoError(t, err)
	assert.Equal(t, 0, added2)

	list, err := store.ListBoardJobs("new")
	require.NoError(t, err)
	require.Len(t, list, 2)
	// ordered by skill_score DESC
	assert.Equal(t, "SRE", list[0].Title)
	assert.Equal(t, "Go", list[0].MatchedSkills)

	require.NoError(t, store.SetBoardJobStatus(list[1].ID, "dismissed"))
	newList, _ := store.ListBoardJobs("new")
	assert.Len(t, newList, 1)
	dismissed, _ := store.ListBoardJobs("dismissed")
	assert.Len(t, dismissed, 1)
}
