# Job Scraping — Design Spec

**Date:** 2026-06-08
**Status:** Approved (design); pending implementation plan

## Summary

Add a feature that scrapes job vacancies directly from companies' ATS-backed
career pages, filters them against the user's remote/timezone/region/role
preferences, and surfaces the matches in the dashboard. The user manually
triggers a scrape by running `/scrape-jobs` in Claude Code. From the dashboard
the user can review matches, copy a ready-to-run `/apply <url>` command for any
job, and dismiss the rest. Applications created from a scraped job are
automatically linked back and marked **Scraped**, proving the scraper's
effectiveness.

## Goals

- Pull job listings from the major ATS platforms behind company career pages.
- Filter for remote roles that fit the user's location rule (home timezone +
  accepted regions) and target role keywords.
- Keep token cost low: the scraper returns a compact `title | location | url`
  list; full job descriptions are never sent to Claude during scraping.
- Display matches in the dashboard with a legible reason for each match.
- Hand off cleanly to the existing `/apply` flow; mark scrape-sourced
  applications.
- Make scrape failures visible and persistent, never silent.

## Non-goals

- Job boards / aggregators (RemoteOK, LinkedIn, etc.) — explicitly out; this is
  "direct from company website".
- Scheduled / automatic scraping — trigger is manual only.
- Scraping arbitrary custom career pages — only the supported ATS platforms.
- Sending full job descriptions to Claude during scraping (only at `/apply`
  time, which already happens today).

## Architecture

The work splits along spicebag's existing seam: **Go does the mechanical
fetching, Claude does the fuzzy judgment, the dashboard displays.**

```
   ┌─ Dashboard (Vue) ─────────────┐         ┌─ Claude Code ──────────────┐
   │ Settings: companies, roles,   │         │  /scrape-jobs (skill)      │
   │           location prefs      │         │   1. get_scrape_preferences│
   │ Jobs page: scraped results    │         │   2. fetch_ats_jobs (all)  │
   │   → "Apply" → copyable /apply  │         │   3. Claude filters by     │
   │ Settings: per-company status  │         │      tz/region/role        │
   └───────────────┬───────────────┘         │   4. save_scraped_jobs     │
                   │ REST                     └──────────┬─────────────────┘
                   ▼                                     ▼
            ┌─ Go backend ───────────────────────────────────────┐
            │ internal/scrape/  (ATS adapters + URL detect/norm)  │
            │ internal/db/      (new tables + queries)            │
            │ internal/mcp/     (3 new tools)                     │
            │ internal/dashboard/ (handlers_scrape.go + routes)   │
            └────────────────────────────────────────────────────┘
```

### Loop when the user runs `/scrape-jobs`

1. Skill reads saved preferences + companies (`get_scrape_preferences`).
2. Skill calls `fetch_ats_jobs` (all companies) → one MCP call hits every
   registered ATS, returns a compact `[{company, title, location, url}]` list
   plus per-company outcomes (success+count or classified error).
3. Claude judges each job against the timezone/region/role rule, keeping a
   one-line `match_reason` per kept job.
4. Skill calls `save_scraped_jobs` with the matches (`INSERT OR IGNORE` on URL).
5. Skill reports: new vs. already-seen counts, plus any company errors.

The user then opens the dashboard **Jobs** page, clicks **Apply** on jobs they
like, and pastes the `/apply <url>` it provides. When that application is
created, the backend auto-links it to the scraped job by normalized URL and
tags it **Scraped** in the Applications list.

## Data model (migration 007)

Four new tables.

### `scrape_companies`
| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| name | TEXT | display name |
| ats_platform | TEXT | enum: greenhouse, lever, ashby, smartrecruiters, workable, recruitee, breezy, bamboohr, personio, workday |
| ats_token | TEXT | platform-specific identifier extracted from the careers URL |
| careers_url | TEXT | original pasted URL |
| last_scraped_at | TEXT | timestamp of last attempt |
| last_scrape_status | TEXT | enum: never, ok, error |
| last_scrape_error | TEXT | human-readable classified error, empty when ok |
| last_job_count | INTEGER | jobs returned on last successful fetch |

### `scrape_roles`
| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| keyword | TEXT | target title/keyword, e.g. "SRE" |

### `scrape_prefs` (single row, id = 1)
| Column | Type | Notes |
|---|---|---|
| home_timezone | TEXT | e.g. "UTC+7" |
| location_notes | TEXT | freeform rule Claude reads (accepted regions, tolerance, rejects) |

### `scraped_jobs`
| Column | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| company_id | INTEGER FK → scrape_companies | |
| company_name | TEXT | denormalized for display stability |
| title | TEXT | |
| location | TEXT | raw location string from ATS |
| url | TEXT UNIQUE | dedup key; `INSERT OR IGNORE` |
| match_reason | TEXT | Claude's one-line reason |
| status | TEXT | enum: new, applied, dismissed |
| scraped_at | TEXT | |
| applied_application_id | INTEGER FK → applications, NULL | set on apply linkage |

Only **matched** jobs are saved (Claude already filtered). Re-running a scrape
never duplicates thanks to `url UNIQUE` + `INSERT OR IGNORE`; the skill reports
"N new, M already seen".

## Go scraper (`internal/scrape/`)

Shared interface, one adapter file per platform:

```go
type Adapter interface {
    FetchJobs(ctx context.Context, token string) ([]Job, error)
}
type Job struct { Title, Location, URL, RawDept string }
```

### Supported platforms

**Tier 1 — public JSON, no auth, no browser:**

| Platform | Endpoint shape |
|---|---|
| Greenhouse | `boards-api.greenhouse.io/v1/boards/{token}/jobs` |
| Lever | `api.lever.co/v0/postings/{token}?mode=json` |
| Ashby | `api.ashbyhq.com/posting-api/job-board/{token}` |
| SmartRecruiters | `api.smartrecruiters.com/v1/companies/{token}/postings` |
| Workable | `apply.workable.com/api/v3/accounts/{token}/jobs` |
| Recruitee | `{token}.recruitee.com/api/offers/` |
| Breezy HR | `{token}.breezy.hr/json` |
| BambooHR | `{token}.bamboohr.com/careers/list` |

**Tier 2 — public XML feed:**

| Personio | `{token}.jobs.personio.de/xml` |
|---|---|

**Tier 3 — defended, browser required (rod + stealth):**

| Workday | `{tenant}.myworkdayjobs.com/wday/cxs/{tenant}/{site}/jobs` |
|---|---|

Exact endpoint/field paths are verified per-adapter at implementation time;
the structures above are the plan. Adding an 11th platform later = one new file
implementing `Adapter` + one enum entry.

### URL detection & normalization

- `Detect(url) (platform, token, error)` — recognizes platform by hostname
  pattern and extracts the token. Unknown hosts return an "unsupported
  platform" error so Settings refuses to save a broken target.
- `NormalizeURL(url) string` — strip query + fragment, drop trailing slash,
  lowercase host. Shared by `save_scraped_jobs` and the apply linkage lookup so
  both agree on what "same job" means.

### Anti-bot

- Tier 1/2 use plain public endpoints with a realistic User-Agent — minimal
  defenses to evade.
- Tier 3 (Workday) uses the rod browser already vendored
  (`internal/pdf/chrome.go` pattern) with stealth: real Chrome session,
  realistic UA, randomized delays.
- All runs rate-limit with jitter between companies so a scrape doesn't hammer
  any host.

### Error classification

`fetch_ats_jobs` records a per-company outcome rather than aborting the run.
Raw errors map to human-readable messages:

| Cause | Message |
|---|---|
| 404 / bad token | "Company not found on {platform} — the token may have changed" |
| Unexpected response | "Couldn't parse {platform} response — format may have changed" |
| Network / timeout | "Couldn't reach {platform} (timeout)" |
| Workday blocked | "Request was blocked — try again later" |
| 0 openings | "No open roles right now" (status ok, count 0) |

## MCP tools (`internal/mcp/scrape_tools.go`)

1. **`get_scrape_preferences`** → `{ companies, roles, home_timezone,
   location_notes }`. Everything Claude needs to match.
2. **`fetch_ats_jobs`** (no args = all registered companies) → runs each
   adapter, applies a loose coarse server-side pre-filter (drops only listings
   with no remote signal at all), returns the compact list + per-company
   outcomes. Writes `last_scraped_at`, `last_scrape_status`,
   `last_scrape_error`, `last_job_count` for each company.
3. **`save_scraped_jobs`** (`[{company_id, title, location, url, match_reason}]`)
   → `INSERT OR IGNORE`, returns new-vs-existing counts.

The coarse pre-filter is deliberately loose; the real timezone/region/role
judgment stays with Claude so nothing borderline is dropped before Claude sees
it.

## Dashboard UI

### New top-level menu "Jobs" → `JobsView.vue`

Table of `scraped_jobs` where `status = new`, newest first:
`Company | Role | Location | Found | Why matched | [Apply] [Dismiss]`.

- **Why matched** shows `match_reason` — makes the scraper's judgment legible.
- **Apply** opens an inline panel with a copyable `/apply <url>` command (reuse
  the copy-button pattern from AppDetailView) + a link to the job post. It does
  not change status by itself.
- **Dismiss** sets `status = dismissed`; a "Show dismissed" toggle reveals them.
- Header strip shows last run time and, if the last run had failures, an
  aggregate banner: ⚠ "N of M companies failed to scrape — see Settings."
- No scrape button — scraping is Claude-driven, consistent with the rest of the
  dashboard's slash-command handoffs.

### Settings additions (alongside Sources + Themes)

1. **Scrape Companies** — list with platform badge, job count, and per-company
   status indicator (✅ `12 jobs · 2h ago` / 🔴 `error reason · 2h ago`). "Add"
   field: paste a careers URL → backend detects platform/token, validates with
   one live fetch, shows count, saves. Delete per row.
2. **Target Roles** — add/remove chips.
3. **Location Preferences** — `home_timezone` input + `location_notes` textarea,
   with placeholder text seeding the rule (anywhere / UTC+7-inclusive windows /
   APAC / Asia / Indonesia; reject US-only, EMEA-only).

### Applications list addition

A **Scraped** badge on rows whose application is linked from a `scraped_jobs`
row, surfaced as `FromScrape` on `ApplicationWithStatus` via an `EXISTS`
subquery. This is the effectiveness proof.

### REST handlers (`handlers_scrape.go`)

CRUD for companies (with detect-and-validate), roles, prefs; list scraped jobs;
update job status (dismiss/undismiss). Router gets a `/jobs` SPA route and
`/api/scrape/*` endpoints.

## Apply linkage (automatic, by URL)

The only change to existing code outside the new feature. In `create_application`
(MCP) / `UpsertApplication` (db): after the application row is inserted,
normalize its `job_url` and look it up against `scraped_jobs`. On a match:

- set `scraped_jobs.applied_application_id` = new application id
- set that scraped job's `status = applied` (leaves the Jobs "new" list)

The copyable command stays plain `/apply <url>` — no new syntax, no change to
the apply *skill*, only to the create-application write path.

## The `/scrape-jobs` skill (`plugins/skills/scrape-jobs.md`)

Registered in `.claude-plugin/plugin.json`. Flow:

1. **Load preferences** — `get_scrape_preferences`. If no companies, no roles,
   or no `home_timezone`/`location_notes`, stop and tell the user to configure
   them in dashboard Settings (name the exact sections).
2. **Fetch** — `fetch_ats_jobs` (all). A dead token reports an error and the run
   continues for other companies.
3. **Judge each job:**
   - **Role fit** — semantic match to target roles, not raw substring ("Site
     Reliability Engineer" matches "SRE"; "DevRel" does not match "DevOps").
   - **Location fit** — apply `home_timezone` + `location_notes`: accept
     worldwide/anywhere, any stated timezone window that includes the home TZ,
     and the listed accepted regions; reject region-locked roles that exclude
     it.
   - Keep a one-line `match_reason` per kept job.
4. **Save** — `save_scraped_jobs` with matches.
5. **Report** — "X new, Y already seen, Z companies had errors (with reasons)" +
   "Open the dashboard Jobs page to review and apply."

## Testing approach

Matches the repo's existing style (temp-dir DB, no global state):

- **ATS adapters** — table tests against recorded JSON/XML fixtures per
  platform; no live network in tests.
- **URL detection + normalization** — pure unit tests.
- **MCP tools** — in-process `CallTool` against a temp-dir DB.
- **Dashboard handlers** — `httptest` for scrape CRUD + dismiss endpoints.
- **Linkage** — create a scraped job, run `create_application` with the matching
  URL, assert FK set + status flip + `FromScrape` true.

## Open items resolved during design

- **Target**: ATS-backed career pages (10 platforms), not job boards.
- **Orchestration**: Claude slash command; MCP fetches structured data, Claude
  matches. ~10–12K tokens for ~200 jobs.
- **Preferences storage**: SQLite, edited in dashboard Settings, read by MCP.
- **Location rule**: home timezone + freeform notes.
- **Apply linkage**: automatic by normalized URL; plain `/apply <url>`.
- **Error visibility**: persistent per-company status in Settings + aggregate
  banner on Jobs page + inline in the skill report.
