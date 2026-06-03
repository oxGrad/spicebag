---
name: sync-experience
description: Extract work experience entries from a base CV and sync them into the spicebag experience database so that get_experience_stats reflects your actual CV history
---

Sync work experience from a CV into the spicebag database.

Required: none. Reads `base.html` by default. Pass a CV filename in $ARGUMENTS to target a different CV.

## Process

### 1. Load the CV

Determine the source CV filename:
- If $ARGUMENTS is non-empty, use it as the filename
- Otherwise default to `base.html`

Call `read_cv` with that filename. If the CV is not found, tell the user and stop.

### 2. Extract experience entries

Parse the HTML for work experience entries. Each entry in the CV maps to one database record. For each job:

- `company`: company name (from `.cv-company` span or equivalent)
- `start_date`: start date in `YYYY-MM` format (e.g. `2022-03`). Parse the date from the cv-dates span — convert month names to numbers (`Jan` → `01`, `Mar` → `03`, etc.)
- `end_date`: end date in `YYYY-MM` format, or empty string `""` if the role is current (present/ongoing)
- `role_type`: infer from the job title using these categories:
  - `DevOps` — DevOps Engineer, Infrastructure Engineer, Platform Engineer, SRE, Site Reliability Engineer
  - `Engineering Leadership` — Head of Engineering, VP Engineering, CTO, Engineering Manager, Tech Lead
  - `Software Engineering` — Software Engineer, Backend Engineer, Frontend Engineer, Full Stack, Developer
  - `Founder` — Founder, Co-founder, CEO
  - If the title fits multiple, pick the most specific. If none match, use the title verbatim.

Do not include education entries. Only include employment.

### 3. Preview and confirm

Show a compact table of what will be synced:

| Company | Role Type | Start | End |
|---------|-----------|-------|-----|
| Acme Corp | DevOps | 2022-03 | 2024-06 |
| ...      | ...     | ...   | (current) |

Ask the user to confirm before writing. If they correct any entry, apply the correction before proceeding.

### 4. Sync to database

Call `add_experience` with:
- `synced_from`: the CV filename (e.g. `base.html`)
- `entries`: JSON array of all confirmed entries

This replaces any previous entries synced from the same CV, so it is safe to re-run after CV updates.

### 5. Report

State how many entries were synced. Remind the user they can view the breakdown at `/stats` in the dashboard or by calling `get_experience_stats`.

## Rules

- Dates must be `YYYY-MM` format. If only a year is given (e.g. `2019`), use `YYYY-01` for start and `YYYY-12` for end.
- Do not invent or rewrite company names or dates. Extract verbatim.
- If the CV has no work experience section, say so and stop.
- Re-running is safe: `add_experience` clears the previous entries for this CV before inserting.
