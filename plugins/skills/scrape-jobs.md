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
