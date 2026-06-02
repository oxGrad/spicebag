---
name: apply
description: Create a complete job application — tailored CV + cover letter + saved job post — from a job post URL, file, or pasted text
---

Create a complete job application for the job post provided in $ARGUMENTS.

Required: job post content (file reference, URL, or free-text). Optional: company name and role title if not clear from the post.

## Process

1. Read the job post from $ARGUMENTS:
   - If it starts with `@` it is a file reference — read that file
   - If it starts with `http` it is a URL — fetch the content
   - Otherwise treat it as a free-text job description
2. Extract from the job post: company name, role title, and today's date (YYYY-MM-DD)
   - If company or role cannot be reliably determined, ask the user before continuing
3. Call `list_cvs` to see available base CVs
4. Select the most relevant CV for this role and call `read_cv` to load it
5. Call `get_experience_stats` for accurate years of experience per role type
6. Write both documents in one pass:

   **Tailored CV** — generate an HTML fragment (no DOCTYPE, no `<html>`, `<head>`, or `<body>` tags) using the standard CV structure:
   - `<h1>` for the person's name
   - `<p class="cv-contact">` for the contact line
   - `<h2>` for section headings (Experience, Education, Skills)
   - `<div class="cv-entry">` wrapping each job/education entry
   - `<div class="cv-entry-header">` containing `<h3>` (job title), `<span class="cv-company">`, `<span class="cv-dates">`
   - `<ul>/<li>` for bullet points
   - Keep all factual data accurate; only adjust wording and emphasis for the role

   **Cover letter** — generate an HTML fragment (no DOCTYPE, no `<html>`, `<head>`, or `<body>` tags) using this structure:
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
   - Company-specific opener, concrete experience references, under 350 words of body text

7. Call `create_application` with:
   - `company`: company name (exact, as it appears in the job post)
   - `role`: role title (exact)
   - `date`: today in YYYY-MM-DD format
   - `cv_content`: the tailored CV HTML fragment
   - `cover_letter_content`: the cover letter HTML fragment
   - `job_post_content`: the full job post text
   - `base_cv_used`: filename of the base CV selected in step 4 (e.g. `base.html` or `cv-{role}.html`)
8. Report the application folder path created
9. Remind the user to open the dashboard (`spicebag start`) to track status

## Rules

- Use accurate years from `get_experience_stats` — never guess
- CV: keep facts accurate, only adjust emphasis and wording
- Cover letter: company-specific opener, concrete experience references, under 350 words
- Never invent company name or role — always extract from the job post or ask
