---
name: customize-cv
description: Tailor a base CV for a specific role type and save it as a new versioned CV in your CV library
---

Tailor a base CV for the job post or role type provided in $ARGUMENTS.

## Process

1. Call `list_cvs` to see all available base CVs
2. Select the most relevant CV for the target role (use filename to determine role type)
3. Call `read_cv` with the selected filename to load the full content
4. Call `get_experience_stats` to get accurate years of experience per role type
5. Read the job post or role description from $ARGUMENTS:
   - If it starts with `@` it is a file reference — read that file
   - If it starts with `http` it is a URL — fetch the content
   - Otherwise treat it as a free-text role description
6. Rewrite the CV as an HTML fragment (no DOCTYPE, no `<html>`, `<head>`, or `<body>` tags — just semantic body content):
   - Keep all factual data accurate: companies, dates, actual tools used
   - Adjust wording, bullet point emphasis, and section ordering
   - Update the summary/objective section to match the target role
   - Use exact years from `get_experience_stats` — never guess or round
   - Use `<h1>` for the person's name, `<h2>` for section headings (Experience, Education, Skills), `<h3>` for job titles with company on the same line or next line, and `<ul>/<li>` for bullet points
7. Generate a filename: `cv-{role-type}-{YYYY-MM-DD}.html` using today's date
8. Call `write_cv` with the new filename and tailored HTML fragment content
9. Confirm the filename saved and summarize the key changes made

## Rules

- This creates a **base CV variant** — not a job-specific application. Use `/apply` for job applications.
- Never alter factual data (companies, dates, actual skills used)
- Always use `get_experience_stats` for total years of experience — do not compute from CV text
- If $ARGUMENTS is empty, ask the user for the target role type before proceeding
