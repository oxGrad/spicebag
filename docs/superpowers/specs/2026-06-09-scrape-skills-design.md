# Design: Skill-based job scraping match path

Date: 2026-06-09
Branch: feat/job-scraping

## Problem

The existing scrape filter qualifies jobs by role title match only. Unusual but relevant titles like "CI Engineer" (DevOps-adjacent, CI/CD focused) are missed because they don't substring-match any target role. The user wants a second match path: if the job title signals a target skill (Go, Rust, Kubernetes, etc.), the job qualifies regardless of title.

Multiple skill hits should surface higher in the dashboard — a "Go Kubernetes Platform Engineer" is a stronger signal than a "Go Backend Engineer".

## Decision: Option B — `scrape_skills` table, Claude semantic matching, score stored

Mirrors the existing `scrape_roles` pattern exactly. Claude does semantic title matching (handles "Golang" → Go, "k8s" → Kubernetes). Score and matched skill names are stored on `scraped_jobs` for dashboard sorting and badge display.

## Match logic

Location is always the gate. A job passes if:

```
location_passes AND (role_match OR skill_score >= 1)
```

Both can be true simultaneously. A "Go DevOps Engineer" records both a role match and a skill match. Role-only and skill-only jobs are both saved; the report distinguishes them.

## Data model

### New table (migration 008)

```sql
CREATE TABLE scrape_skills (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  keyword TEXT NOT NULL UNIQUE
);
```

### New columns on `scraped_jobs`

```sql
ALTER TABLE scraped_jobs ADD COLUMN matched_skills TEXT NOT NULL DEFAULT '';
ALTER TABLE scraped_jobs ADD COLUMN skill_score    INTEGER NOT NULL DEFAULT 0;
```

- `matched_skills` — comma-separated skill names found in the job title (e.g. `"Go,Kubernetes"`). Empty when no skill matched.
- `skill_score` — integer hit count (0, 1, 2, …). Zero for role-only matches.

## Backend (Go)

### `internal/db/migrations/000008_add_scrape_skills.up.sql`
Creates `scrape_skills` table and adds the two columns to `scraped_jobs`.

### `internal/db/scrape.go`
- Add `ScrapeSkill` struct `{ID int64, Keyword string}`.
- Add `ListScrapeSkills() ([]ScrapeSkill, error)`.
- Add `AddScrapeSkill(keyword string) error` and `DeleteScrapeSkill(id int64) error`.
- Add `MatchedSkills string` and `SkillScore int` fields to `ScrapedJob` struct.
- Include both fields in the `SaveScrapedJobs` INSERT.

### `internal/mcp/scrape_tools.go`
- `get_scrape_preferences`: add `"skills": skills` to the JSON payload alongside `"roles"`.
- `save_scraped_jobs`: add optional `matched_skills` (string) and `skill_score` (int) fields to the input schema; pass through to `db.ScrapedJob`.

### `internal/dashboard/` (scrape handlers)
- GET `/api/scrape/jobs` — include `matched_skills` and `skill_score` in each row's JSON.
- GET `/api/scrape/skills` — return all rows from `scrape_skills`.
- POST `/api/scrape/skills` — add a keyword (body: `{"keyword": "..."}`).
- DELETE `/api/scrape/skills/:id` — delete by ID.
- GET `/api/scrape/jobs` sort order: `ORDER BY skill_score DESC, scraped_at DESC`.

No changes to `internal/scrape/` adapters — skill matching stays in Claude.

## Claude skill (`plugins/skills/scrape-jobs.md`)

### Step 1 — Load preferences
`get_scrape_preferences` now returns `skills`. Stop-early condition: if both `roles` and `skills` are empty, tell the user to add at least one in Settings.

### Step 3 — Judge each job
Expand the keep condition:

> Keep a job if location passes AND either:
> - **Role fit** — title is a semantic match for a target role
> - **Skill fit** — title signals one or more target skills (e.g. "Golang" → Go, "k8s" → Kubernetes)
>
> For each kept job record:
> - `matched_skills` — comma-separated skills found (empty if role-only)
> - `skill_score` — count of matched skills (0 if role-only)
> - `match_reason` — include skill hits when present, e.g. `"worldwide remote · Go,Kubernetes"`

### Step 4 — Save
`save_scraped_jobs` call includes `matched_skills` and `skill_score` per job.

### Step 5 — Report
Add: `"X new matches (Y skill-only, Z role+skill)."` so the user knows how many skill-path matches were found.

## Frontend

### `JobsView.vue`
- Render skill badges next to the job title link for each item in `matched_skills.split(',')` when non-empty.
- Badge style: compact pill, indigo/violet color to visually distinguish from location/reason text.
- Sort is handled server-side (`skill_score DESC, scraped_at DESC`); no Vue-side sort logic.

### `SettingsView.vue`
- Add a "Target Skills" subsection inside Job Scraping, structurally identical to the "Target Roles" subsection.
- Text input + Add button; list of current keywords each with a Delete button.
- Calls `/api/scrape/skills` (GET, POST, DELETE).

## Out of scope

- Skill matching against full job descriptions (happens later via `/apply`).
- Weighted scoring (skills are weighted equally; score = raw hit count).
- Skill synonyms managed outside Claude (Claude handles "Golang"/"Go", "k8s"/"Kubernetes" semantically).
