# Scrape Skills Match Path Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `scrape_skills` table so job scraping has a second match path — jobs qualify if they signal a target skill (Go, Rust, Kubernetes, etc.) in the title, even if the title doesn't match a target role. Skill hits produce a numeric score and named badges in the Jobs dashboard.

**Architecture:** New `scrape_skills` table mirrors `scrape_roles` exactly. Two new columns on `scraped_jobs` (`matched_skills` TEXT, `skill_score` INTEGER) store Claude's judgment. Claude receives the skills list via `get_scrape_preferences` and does semantic title matching; scores are stored and used for dashboard sort order and badge display. The scrape-jobs skill is updated to apply the new match logic.

**Tech Stack:** Go 1.22+, modernc.org/sqlite (pure Go SQLite), Vue 3 + Tailwind CSS, mcp-go

---

## File Map

| File | Change |
|---|---|
| `internal/db/migrations/000008_add_scrape_skills.up.sql` | Create |
| `internal/db/migrations/000008_add_scrape_skills.down.sql` | Create |
| `internal/db/scrape.go` | Modify — new struct + CRUD + updated ScrapedJob |
| `internal/db/scrape_test.go` | Modify — new tests for skills CRUD and scored save |
| `internal/dashboard/handlers_scrape.go` | Modify — 3 new skill handlers |
| `internal/dashboard/server.go` | Modify — 3 new route registrations |
| `internal/dashboard/handlers_scrape_test.go` | Modify — skill handler tests |
| `internal/mcp/scrape_tools.go` | Modify — skills in prefs; matched_skills/skill_score in save |
| `internal/mcp/scrape_tools_test.go` | Modify — updated tests for both tools |
| `frontend/src/api.js` | Modify — skills(), addSkill(), deleteSkill() |
| `frontend/src/views/SettingsView.vue` | Modify — Target Skills section |
| `frontend/src/views/JobsView.vue` | Modify — skill badges in role column |
| `plugins/skills/scrape-jobs.md` | Modify — updated match logic and skill instructions |

---

## Task 1: DB migration

**Files:**
- Create: `internal/db/migrations/000008_add_scrape_skills.up.sql`
- Create: `internal/db/migrations/000008_add_scrape_skills.down.sql`

- [ ] **Step 1: Write up migration**

```sql
-- internal/db/migrations/000008_add_scrape_skills.up.sql
CREATE TABLE scrape_skills (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  keyword TEXT NOT NULL UNIQUE
);

ALTER TABLE scraped_jobs ADD COLUMN matched_skills TEXT    NOT NULL DEFAULT '';
ALTER TABLE scraped_jobs ADD COLUMN skill_score    INTEGER NOT NULL DEFAULT 0;
```

- [ ] **Step 2: Write down migration**

```sql
-- internal/db/migrations/000008_add_scrape_skills.down.sql
ALTER TABLE scraped_jobs DROP COLUMN skill_score;
ALTER TABLE scraped_jobs DROP COLUMN matched_skills;
DROP TABLE scrape_skills;
```

- [ ] **Step 3: Verify the migration applies cleanly**

Run:
```bash
cd /home/graditya/projects/oxGrad/spicebag && go test ./internal/db/... -v -run TestMigration 2>&1 | head -30
go test ./internal/db/... 2>&1
```

Expected: all DB tests pass (migrations run automatically in `openTestStore`).

- [ ] **Step 4: Commit**

```bash
git add internal/db/migrations/000008_add_scrape_skills.up.sql \
        internal/db/migrations/000008_add_scrape_skills.down.sql
git commit -m "feat: migration 008 — scrape_skills table and scored columns on scraped_jobs"
```

---

## Task 2: DB layer — skills CRUD + scored ScrapedJob

**Files:**
- Modify: `internal/db/scrape.go`
- Modify: `internal/db/scrape_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/db/scrape_test.go` after `TestScrapeRolesCRUD`:

```go
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
```

- [ ] **Step 2: Run tests — expect failure**

```bash
cd /home/graditya/projects/oxGrad/spicebag && go test ./internal/db/... -run "TestScrapeSkillsCRUD|TestScrapedJobsSaveWithSkillScore" -v 2>&1
```

Expected: compile error — `ScrapeSkill`, `AddScrapeSkill`, `ListScrapeSkills`, `DeleteScrapeSkill`, `MatchedSkills`, `SkillScore` undefined.

- [ ] **Step 3: Implement — add ScrapeSkill struct and CRUD, update ScrapedJob**

In `internal/db/scrape.go`:

Add `ScrapeSkill` struct after `ScrapeRole`:
```go
type ScrapeSkill struct {
	ID      int64  `json:"id"`
	Keyword string `json:"keyword"`
}
```

Add CRUD methods after `DeleteScrapeRole`:
```go
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
```

Add `MatchedSkills` and `SkillScore` to `ScrapedJob` (after `MatchReason`):
```go
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
```

Update `SaveScrapedJobs` INSERT:
```go
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
```

Update `ListScrapedJobs` SELECT and Scan (sort by `skill_score DESC, scraped_at DESC`):
```go
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
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd /home/graditya/projects/oxGrad/spicebag && go test ./internal/db/... -v 2>&1
```

Expected: all DB tests pass including the two new ones.

- [ ] **Step 5: Commit**

```bash
git add internal/db/scrape.go internal/db/scrape_test.go
git commit -m "feat: ScrapeSkill CRUD and scored ScrapedJob fields"
```

---

## Task 3: Dashboard HTTP handlers — skills endpoints

**Files:**
- Modify: `internal/dashboard/handlers_scrape.go`
- Modify: `internal/dashboard/server.go`
- Modify: `internal/dashboard/handlers_scrape_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/dashboard/handlers_scrape_test.go`:

```go
func TestScrapeSkillAddListDelete(t *testing.T) {
	srv := newTestServer(t)

	// Add
	req := httptest.NewRequest(http.MethodPost, "/api/scrape/skills",
		strings.NewReader("keyword=Go"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Go")

	// List
	req2 := httptest.NewRequest(http.MethodGet, "/api/scrape/skills", nil)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "Go")

	// Extract ID and delete
	var skills []struct{ ID int64 `json:"id"` }
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &skills))
	require.Len(t, skills, 1)

	req3 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/scrape/skills/%d", skills[0].ID), nil)
	w3 := httptest.NewRecorder()
	srv.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusNoContent, w3.Code)
}
```

You'll need `encoding/json` and `fmt` in the import block if not already present.

- [ ] **Step 2: Run test — expect failure**

```bash
cd /home/graditya/projects/oxGrad/spicebag && go test ./internal/dashboard/... -run TestScrapeSkillAddListDelete -v 2>&1
```

Expected: 404 — routes not registered yet.

- [ ] **Step 3: Add handler functions**

Append to `internal/dashboard/handlers_scrape.go`:

```go
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
```

- [ ] **Step 4: Register routes**

In `internal/dashboard/server.go`, add after the `DELETE /api/scrape/roles/{id}` line:

```go
s.mux.HandleFunc("GET /api/scrape/skills", s.handleAPIScrapeSkillsList)
s.mux.HandleFunc("POST /api/scrape/skills", s.handleAPIScrapeSkillCreate)
s.mux.HandleFunc("DELETE /api/scrape/skills/{id}", s.handleAPIScrapeSkillDelete)
```

- [ ] **Step 5: Run tests — expect pass**

```bash
cd /home/graditya/projects/oxGrad/spicebag && go test ./internal/dashboard/... -v 2>&1
```

Expected: all dashboard tests pass including `TestScrapeSkillAddListDelete`.

- [ ] **Step 6: Commit**

```bash
git add internal/dashboard/handlers_scrape.go \
        internal/dashboard/server.go \
        internal/dashboard/handlers_scrape_test.go
git commit -m "feat: dashboard API endpoints for scrape_skills (list, add, delete)"
```

---

## Task 4: MCP tools — skills in preferences, scored save

**Files:**
- Modify: `internal/mcp/scrape_tools.go`
- Modify: `internal/mcp/scrape_tools_test.go`

- [ ] **Step 1: Write failing tests**

In `internal/mcp/scrape_tools_test.go`, update `TestGetScrapePreferences` to also seed and assert skills:

```go
func TestGetScrapePreferences(t *testing.T) {
	_, srv := setup(t)
	store := srv.Store()

	_, err := store.AddScrapeCompany(db.ScrapeCompany{
		Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u",
	})
	require.NoError(t, err)
	_, err = store.AddScrapeRole("SRE")
	require.NoError(t, err)
	_, err = store.AddScrapeSkill("Go")
	require.NoError(t, err)
	require.NoError(t, store.UpdateScrapePrefs(db.ScrapePrefs{HomeTimezone: "UTC+7", LocationNotes: "APAC"}))

	out, err := srv.CallTool(context.Background(), "get_scrape_preferences", map[string]any{})
	require.NoError(t, err)

	var got struct {
		Companies []db.ScrapeCompany `json:"companies"`
		Roles     []db.ScrapeRole    `json:"roles"`
		Skills    []db.ScrapeSkill   `json:"skills"`
		HomeTZ    string             `json:"home_timezone"`
		Notes     string             `json:"location_notes"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Len(t, got.Companies, 1)
	assert.Len(t, got.Roles, 1)
	assert.Len(t, got.Skills, 1)
	assert.Equal(t, "Go", got.Skills[0].Keyword)
	assert.Equal(t, "UTC+7", got.HomeTZ)
}
```

Add a new test for scored save after `TestSaveScrapedJobs`:

```go
func TestSaveScrapedJobsWithSkillScore(t *testing.T) {
	_, srv := setup(t)
	store := srv.Store()
	c, _ := store.AddScrapeCompany(db.ScrapeCompany{Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u"})

	jobsJSON := fmt.Sprintf(`[
		{"company_id":%d,"title":"Go SRE","location":"Remote","url":"https://j/10",
		 "match_reason":"worldwide remote · Go","matched_skills":"Go","skill_score":1}
	]`, c.ID)

	out, err := srv.CallTool(context.Background(), "save_scraped_jobs", map[string]any{"jobs": jobsJSON})
	require.NoError(t, err)
	assert.Contains(t, out, `"new":1`)

	jobs, _ := store.ListScrapedJobs("new")
	require.Len(t, jobs, 1)
	assert.Equal(t, "Go", jobs[0].MatchedSkills)
	assert.Equal(t, 1, jobs[0].SkillScore)
}
```

- [ ] **Step 2: Run tests — expect failure**

```bash
cd /home/graditya/projects/oxGrad/spicebag && go test ./internal/mcp/... -run "TestGetScrapePreferences|TestSaveScrapedJobsWithSkillScore" -v 2>&1
```

Expected: `TestGetScrapePreferences` fails (no `skills` key); `TestSaveScrapedJobsWithSkillScore` fails (fields ignored).

- [ ] **Step 3: Update get_scrape_preferences**

In `internal/mcp/scrape_tools.go`, inside the `get_scrape_preferences` handler, add skills fetch and include in payload. Replace the existing handler body:

```go
func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
    companies, err := s.store.ListScrapeCompanies()
    if err != nil {
        return mcplib.NewToolResultError(err.Error()), nil
    }
    roles, err := s.store.ListScrapeRoles()
    if err != nil {
        return mcplib.NewToolResultError(err.Error()), nil
    }
    skills, err := s.store.ListScrapeSkills()
    if err != nil {
        return mcplib.NewToolResultError(err.Error()), nil
    }
    prefs, err := s.store.GetScrapePrefs()
    if err != nil {
        return mcplib.NewToolResultError(err.Error()), nil
    }
    if companies == nil {
        companies = []db.ScrapeCompany{}
    }
    if roles == nil {
        roles = []db.ScrapeRole{}
    }
    if skills == nil {
        skills = []db.ScrapeSkill{}
    }
    payload := map[string]any{
        "companies":      companies,
        "roles":          roles,
        "skills":         skills,
        "home_timezone":  prefs.HomeTimezone,
        "location_notes": prefs.LocationNotes,
    }
    b, _ := json.Marshal(payload)
    return mcplib.NewToolResultText(string(b)), nil
},
```

- [ ] **Step 4: Update save_scraped_jobs input schema and handler**

In `registerSaveScrapedJobs`, add two optional string/int fields to the tool definition:

```go
mcplib.NewTool(
    "save_scraped_jobs",
    mcplib.WithDescription("Save matched jobs (those that pass the user's timezone/region/role/skill rules). Jobs whose URL already exists are ignored. Returns counts of new vs already-seen."),
    mcplib.WithString("jobs", mcplib.Required(),
        mcplib.Description(`JSON array of {"company_id": <id>, "title": "...", "location": "...", "url": "...", "match_reason": "...", "matched_skills": "...", "skill_score": <int>}`)),
),
```

Update the `entries` struct and `db.ScrapedJob` construction inside the handler:

```go
var entries []struct {
    CompanyID     int64  `json:"company_id"`
    Title         string `json:"title"`
    Location      string `json:"location"`
    URL           string `json:"url"`
    MatchReason   string `json:"match_reason"`
    MatchedSkills string `json:"matched_skills"`
    SkillScore    int    `json:"skill_score"`
}
if err := json.Unmarshal([]byte(jobsJSON), &entries); err != nil {
    return mcplib.NewToolResultError(fmt.Sprintf("invalid jobs JSON: %v", err)), nil
}
names := s.companyNames()
var jobs []db.ScrapedJob
for _, e := range entries {
    jobs = append(jobs, db.ScrapedJob{
        CompanyID:     e.CompanyID,
        CompanyName:   names[e.CompanyID],
        Title:         e.Title,
        Location:      e.Location,
        URL:           e.URL,
        MatchReason:   e.MatchReason,
        MatchedSkills: e.MatchedSkills,
        SkillScore:    e.SkillScore,
    })
}
```

- [ ] **Step 5: Run tests — expect pass**

```bash
cd /home/graditya/projects/oxGrad/spicebag && go test ./internal/mcp/... -v 2>&1
```

Expected: all MCP tests pass including the two updated/new ones. (The `TestFetchATSJobsRecordsStatus` test is a network test skipped by default — that's fine.)

- [ ] **Step 6: Commit**

```bash
git add internal/mcp/scrape_tools.go internal/mcp/scrape_tools_test.go
git commit -m "feat: MCP tools expose skills in preferences and accept skill score on save"
```

---

## Task 5: Frontend — API, Settings, Jobs badges

**Files:**
- Modify: `frontend/src/api.js`
- Modify: `frontend/src/views/SettingsView.vue`
- Modify: `frontend/src/views/JobsView.vue`

- [ ] **Step 1: Add skills calls to api.js**

In `frontend/src/api.js`, add to the `scrape` object after `deleteRole`:

```js
skills: () => get("/scrape/skills"),
addSkill: (keyword) => post("/scrape/skills", { keyword }),
deleteSkill: (id) => del(`/scrape/skills/${id}`),
```

- [ ] **Step 2: Add Target Skills section to SettingsView.vue**

Insert this new `<section>` block in `frontend/src/views/SettingsView.vue` after the closing `</section>` of "Target Roles" (after line 132) and before the "Location Preferences" section:

```html
<section class="bg-white rounded-lg shadow p-5 mb-6">
  <h2 class="font-semibold mb-3">Target Skills</h2>
  <p class="text-xs text-gray-500 mb-3">Jobs mentioning these skills in their title qualify even if the role title doesn't match. Multiple hits raise the skill score.</p>
  <form @submit.prevent="addSkill" class="flex gap-2 mb-3">
    <input v-model="newSkill" type="text" placeholder="e.g. Go, Rust, Kubernetes"
      class="flex-1 border rounded px-2 py-1.5 text-sm" />
    <button class="bg-blue-600 text-white px-3 py-1.5 rounded text-sm">Add</button>
  </form>
  <div class="flex flex-wrap gap-2">
    <span v-for="sk in skills" :key="sk.id" class="flex items-center gap-1 bg-indigo-50 border border-indigo-200 rounded-full px-2.5 py-1 text-xs text-indigo-700">
      {{ sk.keyword }}
      <button @click="deleteSkill(sk.id)" class="text-indigo-400 hover:text-red-600">×</button>
    </span>
  </div>
</section>
```

In the `<script setup>` block, add `skills` ref alongside `roles`:

```js
const skills        = ref([])
const newSkill      = ref('')
```

In `loadScrape()`, add:

```js
skills.value = await api.scrape.skills()
```

Add handler functions after `deleteRole`:

```js
async function addSkill() {
  if (!newSkill.value.trim()) return
  await api.scrape.addSkill(newSkill.value.trim()); newSkill.value = ''; await loadScrape()
}
async function deleteSkill(id) { await api.scrape.deleteSkill(id); await loadScrape() }
```

- [ ] **Step 3: Add skill badges to JobsView.vue**

In `frontend/src/views/JobsView.vue`, update the Role cell (`<td>` containing the title link) to render skill badges below the link. Replace the current role `<td>`:

```html
<td class="px-4 py-3">
  <a :href="job.url" target="_blank" rel="noopener" class="text-blue-600 hover:underline">{{ job.title }}</a>
  <div v-if="job.matched_skills" class="flex flex-wrap gap-1 mt-1">
    <span
      v-for="sk in job.matched_skills.split(',')"
      :key="sk"
      class="inline-block bg-indigo-100 text-indigo-700 text-xs font-medium px-2 py-0.5 rounded-full"
    >{{ sk }}</span>
  </div>
</td>
```

- [ ] **Step 4: Build frontend and verify no errors**

```bash
cd /home/graditya/projects/oxGrad/spicebag/frontend && npm run build 2>&1
```

Expected: build completes with no errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api.js \
        frontend/src/views/SettingsView.vue \
        frontend/src/views/JobsView.vue
git commit -m "feat: skills settings section and skill badges in Jobs dashboard"
```

---

## Task 6: Update scrape-jobs skill

**Files:**
- Modify: `plugins/skills/scrape-jobs.md`

- [ ] **Step 1: Update the skill file**

Replace the entire contents of `plugins/skills/scrape-jobs.md` with:

```markdown
---
name: scrape-jobs
description: Scrape job vacancies from your registered ATS companies, filter them by your timezone/region/role/skill preferences, and save the matches for review in the dashboard.
---

Scrape and filter job vacancies from the user's registered ATS company career pages.

## Process

### 1. Load preferences

Call `get_scrape_preferences`. It returns `companies`, `roles`, `skills`, `home_timezone`,
and `location_notes`.

Stop early and tell the user to configure the dashboard **Settings → Job
Scraping** sections if any of these are empty:
- no `companies` — "Add at least one company (paste a careers URL) in Settings."
- both `roles` and `skills` are empty — "Add at least one target role or skill in Settings."
- empty `home_timezone` or `location_notes` — "Set your Location Preferences in Settings."

### 2. Fetch listings

Call `fetch_ats_jobs` (no arguments). It returns:
- `jobs`: `[{company_id, company, title, location, url}]` — already coarse-filtered for a remote signal
- `errors`: `[{company, error}]` — companies that failed this run (already recorded for the dashboard)

Do not abort if some companies errored; continue with the jobs you did get.

### 3. Judge each job

**Location is the gate — check this first.** A job that fails location is dropped regardless of role or skill.

**Location fit** — apply `home_timezone` + `location_notes`. Accept worldwide/anywhere; accept any
stated timezone window that includes the home timezone; accept the regions the notes list as
acceptable. Reject roles locked to regions that exclude the home timezone (e.g. "US only",
"EMEA only").

Keep a job only if location passes AND either:

- **Role fit** — the title is a semantic match for one of the user's target roles. Match meaning,
  not substrings: "Site Reliability Engineer" matches an "SRE" target; "Developer Relations"
  does NOT match "DevOps". "CI Engineer" DOES match "DevOps Engineer" because it's clearly
  an infra/pipeline role.

- **Skill fit** — the title signals one or more target skills. Match semantically: "Golang"
  matches "Go"; "k8s" matches "Kubernetes"; "Rust developer" matches "Rust". A job can match
  on skill alone — the title does not need to match any target role.

For each kept job, record:
- `match_reason` — one short line, e.g. `"worldwide remote"`, `"APAC includes UTC+7 · Go,Kubernetes"`
- `matched_skills` — comma-separated skills found (e.g. `"Go,Kubernetes"`); empty string if role-only match
- `skill_score` — count of matched skills (0 for role-only matches)

### 4. Save matches

Call `save_scraped_jobs` with the kept jobs as a JSON array of
`{company_id, title, location, url, match_reason, matched_skills, skill_score}`.
It ignores URLs already saved and returns `{new, already_seen}`.

### 5. Report

Summarize:
- "X new matches saved, Y already seen."
- Break down by match type: "N role-only, M skill-only, K role+skill."
- If any companies errored: "Z companies failed: Acme (token not found), …"
- "Open the dashboard **Jobs** page to review and apply."

## Rules

- Never invent jobs — only judge and save what `fetch_ats_jobs` returned.
- Location fit is required for all jobs — skills do not override location.
- Keep `match_reason` to one short line; it shows in the dashboard.
- Do not fetch full job descriptions here — that happens later via `/apply`.
```

- [ ] **Step 2: Run the full test suite**

```bash
cd /home/graditya/projects/oxGrad/spicebag && go test ./... 2>&1
```

Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add plugins/skills/scrape-jobs.md
git commit -m "feat: update scrape-jobs skill with skill-based match path and scored output"
```

---

## Task 7: Seed skills via DB and verify end-to-end

This task seeds the skills you configured earlier (Go, Rust) and verifies the full Go build compiles cleanly.

- [ ] **Step 1: Seed Go and Rust skills**

```bash
sqlite3 ~/.config/spicebag/spicebag.db "
INSERT OR IGNORE INTO scrape_skills (keyword) VALUES ('Go'), ('Rust');
SELECT * FROM scrape_skills;
"
```

Expected output:
```
1|Go
2|Rust
```

- [ ] **Step 2: Full build**

```bash
cd /home/graditya/projects/oxGrad/spicebag && just build 2>&1
```

Expected: frontend builds, Go binary compiles — no errors.

- [ ] **Step 3: Commit**

No code changes. This task is verification only — no commit needed.
