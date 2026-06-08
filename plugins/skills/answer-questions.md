---
name: answer-questions
description: Craft answers for application form questions (e.g. "Why are you a great fit?") and save them to the application record
---

Craft answers for the application form questions linked to the application in $ARGUMENTS.

Required: application integer ID (visible in the dashboard URL and next to the role name). The questions must already be added via the dashboard UI.

## Process

### 1. Load application context

Parse $ARGUMENTS for the integer application ID.

Call `list_application_questions` with that ID. If there are no questions, tell the user to add them via the dashboard first and stop.

Call `get_application_by_id` if available, or use the questions response to confirm the application exists. Read the application's CV and cover letter for voice and context:
- Call `list_cvs` and load the base CV referenced by the application (from `base_cv_used` if available in the questions response metadata)
- The job post content is available in the application folder — use it for role context

### 2. Check memory

Before writing, read memory files relevant to this company and role. Look for:
- `memory/user_exp_{company-slug}.md` — verified experience facts
- Any other user memory files that contain skills or achievements relevant to the job's requirements

Use these facts in answers. Do not invent anything not present in memory or the CV.

### 3. Route each question based on whether bullets are present

For each question in the list, check `user_bullets`:

**Questions WITH bullets** (non-empty `user_bullets` array):
- The user has provided their own key points as source material
- Cross-reference each bullet against memory to add supporting facts or numbers where available
- If a bullet is vague or needs a specific figure to be compelling (e.g. "improved performance" needs a metric), note it as a clarifying question
- Craft the answer by expanding and connecting the bullets into flowing prose

**Questions WITHOUT bullets** (empty `user_bullets`):
- Fall back to the standard approach: assess whether the CV and memory provide enough material
- If a strong answer requires specific claims not in CV/memory, formulate one targeted clarifying question

**Use `AskUserQuestion` — do not write clarifying questions as plain text.**

If there are clarifying questions (from either route), call `AskUserQuestion` one question at a time, waiting for the answer before asking the next. For each question:

1. Set `multiSelect: true`
2. Generate exactly 3 pre-populated options using this priority order:
   - **First**: any verified fact already saved in memory for that company (e.g. `memory/user_exp_{slug}.md`)
   - **Second**: plausible inferences from the CV wording or the user's own bullets
   - **Third**: a reasonable generic answer typical for that role level/domain
3. Set option 4 to: label `"Other / Combine"`, description `"I'll type a custom answer or combine the above"` — always last

After all questions are answered, summarise the confirmed answers as a numbered list and use `AskUserQuestion` to ask:
- Question: "Do these answers look correct? You can revise before I write the application answers."
- `multiSelect: false`
- Option 1: `"Looks good, proceed"` — description: "Use these answers as-is"
- Option 2: `"I want to revise"` — description: "I'll tell you which answer to change"

If the user chooses "I want to revise", let them say which answer to update (free text via the notes field or a follow-up message), update your record, re-summarise, and confirm again before proceeding.

If no clarifying questions are needed, state that briefly and proceed directly to step 4.

### 4. Craft answers

Write one answer per question. Rules:
- Length: 1–3 short paragraphs (under 200 words per answer)
- Voice: consistent with the cover letter tone (professional, direct)
- For bullet-sourced questions: the answer must reflect the user's stated points — do not substitute them with different claims, even better-sounding ones
- Grounded: every specific claim must come from the CV, memory, user bullets, or the user's answers in step 3
- No em dashes (use commas or colons instead)
- No invented achievements, figures, or scale claims
- If the user skipped a clarifying question, write around that claim rather than including it

### 5. Save answers

Call `write_application_answers` with the application ID and an array of `{"question_id": <id>, "answer": "<text>"}` entries.

### 6. Report

State how many answers were saved. Remind the user they can view, copy, and edit them in the dashboard under the application's Questions & Answers section.

## Rules

- Never invent experience or achievements
- If memory contains verified facts about the company, use them
- If a question is about the company (culture fit, mission alignment), fetch the job post URL for context before answering
- Answers must be self-contained — the recruiter reads only the answer, not the question context
