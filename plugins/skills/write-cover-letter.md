---
name: write-cover-letter
description: Write a cover letter for a job post, drawing on your CV library and experience stats, and save it to your cover letter library
---

Write a cover letter for the job post provided in $ARGUMENTS.

## Process

1. Read the job post from $ARGUMENTS:
   - If it starts with `@` it is a file reference: read that file
   - If it starts with `http` it is a URL: fetch the content
   - Otherwise treat it as a free-text job description
2. Call `list_cvs` to see available base CVs
3. Select the most relevant CV for this role and call `read_cv` to load it
4. Call `get_experience_stats` for accurate years of experience per role type
5. Write a compelling cover letter as an HTML fragment (no DOCTYPE, no `<html>`, `<head>`, or `<body>` tags) using this structure:
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
   - **Opening** (`<p class="salutation">` or first body `<p>`): specific to the company and role: never start with "I am writing to apply for"
   - **Body** (2–3 `<p>` paragraphs): connect experience directly to stated role requirements; cite specific projects or achievements; use accurate numbers from `get_experience_stats`
   - **Closing**: confident, concrete call to action
6. Generate a filename: `cl-{company}-{YYYY-MM-DD}.html` using today's date (lowercase, spaces as hyphens)
7. Call `write_cover_letter` with the filename and HTML fragment content
8. Confirm the filename saved

## Rules

- Use accurate years from `get_experience_stats`: never guess
- Target length: under 350 words (approximately one page)
- Address specific requirements from the job post, not generic ones
- If company name cannot be determined from the post, ask before saving
- Never use em dashes (—) in generated cover letter content; use a comma, colon, or rewrite the sentence instead
