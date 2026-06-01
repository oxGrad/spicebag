---
name: apply
description: Create a complete job application — tailored CV + cover letter + saved job post — from a job post URL or file
---

Create a complete job application for the job post provided in $ARGUMENTS.

Required: job post content (file reference or URL). Optional: company name and role title if not clear from the post.

## Process

1. Read the job post from $ARGUMENTS:
   - If it starts with `@` it is a file reference — read that file
   - If it starts with `http` it is a URL — fetch the content
2. Extract from the job post: company name, role title, and today's date (YYYY-MM-DD)
   - If company or role cannot be reliably determined, ask the user before continuing
3. Call `list_cvs` to see available base CVs
4. Select the most relevant CV for this role and call `read_cv` to load it
5. Call `get_experience_stats` for accurate years of experience per role type
6. Write both documents in one pass:
   - **Tailored CV**: adjust emphasis and wording for this role (keep all facts accurate)
   - **Cover letter**: company-specific opener, concrete experience references, under 350 words
7. Call `create_application` with:
   - `company`: company name (exact, as it appears in the job post)
   - `role`: role title (exact)
   - `date`: today in YYYY-MM-DD format
   - `cv_content`: the tailored CV markdown
   - `cover_letter_content`: the cover letter markdown
   - `job_post_content`: the full job post text
   - `base_cv_used`: filename of the base CV selected in step 4
8. Report the application folder path created
9. Remind the user to open the dashboard (`prospector serve`) to track status

## Rules

- Use accurate years from `get_experience_stats` — never guess
- CV: keep facts accurate, only adjust emphasis and wording
- Cover letter: company-specific opener, concrete experience references, under 350 words
- Never invent company name or role — always extract from the job post or ask
