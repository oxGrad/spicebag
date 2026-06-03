---
name: apply
description: Create a complete job application: tailored CV + cover letter + saved job post: from a job post URL, file, or pasted text
---

Create a complete job application for the job post provided in $ARGUMENTS.

Required: job post content (file reference, URL, or free-text). Optional: company name and role title if not clear from the post.

## Process

### 1. Read the job post

From $ARGUMENTS:
- If it starts with `@`: read that file
- If it starts with `http`: it is a URL: fetch the content **and save the URL** for step 7
- Otherwise treat as free-text job description

Extract: company name, role title, today's date (YYYY-MM-DD). If company or role cannot be reliably determined, ask the user before continuing.

**If a URL was provided**, write a 2–3 sentence summary of the role covering: what the company does, the core responsibilities, and the 2–3 most distinctive requirements. Save this as `job_summary` for step 7. Example format: *"[Company] is a [type] company. The [Role] will [key responsibilities]. Key requirements include [top requirements]."*

### 2. Load CV and experience data

- Call `list_cvs` to see available base CVs
- Select the most relevant CV for this role; call `read_cv` to load it
- Call `get_experience_stats` for accurate years of experience per role type

### 3. Assess tailoring need

Compare the job requirements against the CV. State your assessment clearly before proceeding:

**As-is** (minor wording only): use when the CV already covers ≥80% of the key requirements with matching terminology. State: *"CV is a strong match: will use as-is with minor emphasis adjustments."*

**Light tailoring**: use when 1–2 sections need reordering or specific bullets need to move up/down. State what you'll change and why.

**Full tailoring**: use when the role has a materially different emphasis (e.g., backend CV for a full-stack role). State the structural changes planned.

Present this assessment and wait for the user to confirm before writing anything.

### 4. Ask clarifying questions

Before writing, identify the 2–3 experiences in the CV most relevant to this specific role. For each, ask one targeted question that would make the output more specific and accurate.

Good questions:
- Scale: *"At [Company] you mention scaling [system]: what was the peak load or user count? This will strengthen the bullet."*
- Impact: *"What was the measurable outcome of [initiative]? (e.g. % reduction, revenue impact, time saved)"*
- Depth: *"For [technology], were you the lead, a contributor, or a reviewer? The cover letter will reference this."*
- Scope: *"How many engineers were in the team you led at [Company]?"*

Ask all questions in a single numbered list. Wait for answers before writing.

Rules for questions:
- Only ask about things that appear in the job post requirements
- Never invent details: only use what the user confirms
- If the user says "I don't remember" or "skip", note it and do not include unverified claims

### 5. Save verified experience to memory

After receiving the user's answers, save them to the project memory system before writing the documents.

Create or update the file `memory/user_exp_{company-slug}.md` (where `{company-slug}` is the slugified company name from the CV entry, e.g. `user_exp_acme-corp.md`). Use this frontmatter format:

```
---
name: user-exp-{company-slug}
description: Verified experience facts from {Company Name} ({role}): use when writing applications requiring {skill area}
metadata:
  type: user
---

## {Job Title} at {Company Name}

- [Paste the specific verified fact the user confirmed]
- [Scale/impact figures]
- [Team size, leadership scope, etc.]
```

Also update `memory/MEMORY.md` with a pointer line if the file is new.

If a memory file for this company already exists, append new facts rather than overwriting.

### 6. Write the documents

Use only verified facts. Do not include any experience or achievement the user could not confirm in step 4.

**Tailored CV**: generate an HTML fragment (no DOCTYPE, no `<html>`, `<head>`, or `<body>` tags):
- `<h1>` for the person's name
- `<p class="cv-contact">` for the contact line
- `<h2>` for section headings (Experience, Education, Skills)
- `<div class="cv-entry">` wrapping each job/education entry
- `<div class="cv-entry-header">` containing `<h3>` (job title), `<span class="cv-company">`, `<span class="cv-dates">`
- `<ul>/<li>` for bullet points
- Apply the tailoring level decided in step 3
- Every job/education entry MUST be wrapped in `<div class="cv-entry">`: this is what the theme's `page-break-inside: avoid` rule targets to prevent entries from splitting across A4 pages

**Cover letter**: generate an HTML fragment (no DOCTYPE, no `<html>`, `<head>`, or `<body>` tags):
```html
<header>
  <h1>Full Name</h1>
  <div class="contact">email · phone · location · <a href="...">linkedin</a> · <a href="...">github</a></div>
</header>
<div class="date-block">DD Month YYYY</div>
<p class="salutation">Dear [Company] Hiring Team,</p>
<p>Body paragraph...</p>
<p>Body paragraph...</p>
<div class="closing">
  <div class="sign-off">Sincerely,</div>
  <div class="name">Full Name</div>
</div>
```
- Company-specific opener referencing the role
- Use verified facts from step 4 for concrete claims
- Under 350 words of body text

### 7. Save the application

Call `create_application` with:
- `company`: exact company name from the job post
- `role`: exact role title
- `date`: today in YYYY-MM-DD format
- `cv_content`: the tailored CV HTML fragment
- `cover_letter_content`: the cover letter HTML fragment
- `job_post_content`: the full job post text
- `base_cv_used`: filename of the base CV selected in step 2
- `job_url`: the original URL if $ARGUMENTS was a URL, otherwise omit
- `job_summary`: the 2–3 sentence summary written in step 1 if a URL was provided, otherwise omit

### 8. Report

- State the application folder path created
- Briefly summarise what tailoring was done (or that CV was used as-is)
- Remind the user to open the dashboard (`spicebag start`) to track status

## Rules

- Use accurate years from `get_experience_stats`: never guess
- Never invent achievements, scales, or impact figures: only use what the user confirms
- CV facts must be accurate; only emphasis and wording may be adjusted
- Cover letter body must reference specific verified experiences, not generic claims
- If the user skips a question, omit that specific claim from the output entirely
- Never use em dashes (—) in generated CV or cover letter content; use a comma, colon, or rewrite the sentence instead
