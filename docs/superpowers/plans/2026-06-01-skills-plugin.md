# Skills + Plugin Packaging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Write the three Prospector slash command skills (`/customize-cv`, `/write-cover-letter`, `/apply`), bundle two default CSS themes, embed them all in the binary, and update `prospector init` to install them locally — completing the plugin ready for marketplace publishing.

**Architecture:** A new `internal/assets/` package uses `embed.FS` to bundle skills and themes inside the binary. `prospector init` walks those embedded directories and extracts them: themes to `~/.config/prospector/themes/`, skills to `~/.claude/commands/prospector/`. `plugin.json` references the `internal/assets/skills/` paths so the marketplace plugin and the embedded binary both use the same canonical files. README documents the full install flow.

**Tech Stack:** Go `embed.FS` (stdlib), existing `cobra` CLI, existing `internal/assets/` package (new), markdown skill files.

---

## File Map

```
internal/assets/
  assets.go                    ← NEW: embed.FS declarations for skills + themes
  assets_test.go               ← NEW: verify embedded files exist and have content
  skills/
    customize-cv.md            ← NEW: /customize-cv slash command skill
    write-cover-letter.md      ← NEW: /write-cover-letter slash command skill
    apply.md                   ← NEW: /apply slash command skill
  themes/
    minimal.css                ← NEW: serif minimal theme for PDF export
    modern.css                 ← NEW: sans-serif modern theme for PDF export

cmd/prospector/
  cmd_init.go                  ← MODIFY: extract themes + skills during init

plugin.json                    ← MODIFY: update skill paths to internal/assets/skills/
README.md                      ← MODIFY: add installation + usage docs
```

---

## Task 1: Assets package with embedded themes

**Files:**
- Create: `internal/assets/assets.go`
- Create: `internal/assets/assets_test.go`
- Create: `internal/assets/themes/minimal.css`
- Create: `internal/assets/themes/modern.css`

- [ ] **Step 1: Write failing test** (`internal/assets/assets_test.go`)

```go
// internal/assets/assets_test.go
package assets_test

import (
	"io/fs"
	"testing"

	"github.com/oxGrad/spicebag/internal/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThemesEmbedded(t *testing.T) {
	for _, name := range []string{"minimal.css", "modern.css"} {
		data, err := fs.ReadFile(assets.ThemesFS, "themes/"+name)
		require.NoError(t, err, "theme %s must be embedded", name)
		assert.NotEmpty(t, data, "theme %s must not be empty", name)
	}
}

func TestSkillsEmbedded(t *testing.T) {
	for _, name := range []string{"customize-cv.md", "write-cover-letter.md", "apply.md"} {
		data, err := fs.ReadFile(assets.SkillsFS, "skills/"+name)
		require.NoError(t, err, "skill %s must be embedded", name)
		assert.Contains(t, string(data), "name:", "skill %s must have frontmatter", name)
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

```bash
go test ./internal/assets/... 2>&1 | head -5
```

Expected: compile error — package doesn't exist.

- [ ] **Step 3: Create `internal/assets/assets.go`**

```go
// internal/assets/assets.go
package assets

import "embed"

//go:embed themes
var ThemesFS embed.FS

//go:embed skills
var SkillsFS embed.FS
```

- [ ] **Step 4: Create `internal/assets/themes/minimal.css`**

```css
body {
  font-family: Georgia, 'Times New Roman', serif;
  font-size: 11pt;
  line-height: 1.6;
  color: #1a1a1a;
  max-width: 750px;
  margin: 0 auto;
  padding: 40px 20px;
}
h1 {
  font-size: 22pt;
  margin-bottom: 4px;
  border-bottom: 2px solid #1a1a1a;
  padding-bottom: 6px;
}
h2 {
  font-size: 13pt;
  margin-top: 20px;
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
h3 { font-size: 11pt; font-weight: bold; margin-bottom: 2px; }
ul { margin: 4px 0; padding-left: 20px; }
li { margin-bottom: 2px; }
p { margin: 6px 0; }
a { color: #1a1a1a; }
```

- [ ] **Step 5: Create `internal/assets/themes/modern.css`**

```css
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  font-size: 10.5pt;
  line-height: 1.5;
  color: #2d2d2d;
  max-width: 780px;
  margin: 0 auto;
  padding: 40px 24px;
}
h1 {
  font-size: 24pt;
  font-weight: 700;
  color: #1a56db;
  margin-bottom: 4px;
}
h2 {
  font-size: 11pt;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #1a56db;
  border-bottom: 1px solid #e5e7eb;
  padding-bottom: 4px;
  margin-top: 22px;
}
h3 { font-size: 10.5pt; font-weight: 600; margin-bottom: 2px; }
ul { margin: 4px 0; padding-left: 18px; }
li { margin-bottom: 3px; }
p { margin: 5px 0; }
a { color: #1a56db; text-decoration: none; }
```

- [ ] **Step 6: Create placeholder skill files so the embed compiles**

Create `internal/assets/skills/customize-cv.md`:
```markdown
---
name: customize-cv
description: Tailor a base CV for a specific role type (placeholder — Task 2 fills this in)
---
```

Create `internal/assets/skills/write-cover-letter.md`:
```markdown
---
name: write-cover-letter
description: Write a cover letter for a job post (placeholder — Task 2 fills this in)
---
```

Create `internal/assets/skills/apply.md`:
```markdown
---
name: apply
description: Create a full job application (placeholder — Task 2 fills this in)
---
```

- [ ] **Step 7: Run tests**

```bash
go test ./internal/assets/... -v
```

Expected: `TestThemesEmbedded` and `TestSkillsEmbedded` both PASS (placeholder skills have `name:` in frontmatter).

- [ ] **Step 8: Commit**

```bash
git add internal/assets/
git commit -m "feat: assets package with embedded themes and skill placeholders"
```

---

## Task 2: Write the three skill files

**Files:**
- Modify: `internal/assets/skills/customize-cv.md` (replace placeholder)
- Modify: `internal/assets/skills/write-cover-letter.md` (replace placeholder)
- Modify: `internal/assets/skills/apply.md` (replace placeholder)
- Modify: `plugin.json` (update skill paths)

No new tests — `TestSkillsEmbedded` from Task 1 already verifies embedding and frontmatter. After this task it will continue to pass with richer content.

- [ ] **Step 1: Replace `internal/assets/skills/customize-cv.md`**

```markdown
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
6. Rewrite the CV to emphasize skills relevant to this role:
   - Keep all factual data accurate: companies, dates, actual tools used
   - Adjust wording, bullet point emphasis, and section ordering
   - Update the summary/objective section to match the target role
   - Use exact years from `get_experience_stats` — never guess or round
7. Generate a filename: `cv-{role-type}-{YYYY-MM-DD}.md` using today's date
8. Call `write_cv` with the new filename and tailored content
9. Confirm the filename saved and summarize the key changes made

## Rules

- This creates a **base CV variant** — not a job-specific application. Use `/apply` for job applications.
- Never alter factual data (companies, dates, actual skills used)
- Always use `get_experience_stats` for total years of experience — do not compute from CV text
- If $ARGUMENTS is empty, ask the user for the target role type before proceeding
```

- [ ] **Step 2: Replace `internal/assets/skills/write-cover-letter.md`**

```markdown
---
name: write-cover-letter
description: Write a cover letter for a job post, drawing on your CV library and experience stats, and save it to your cover letter library
---

Write a cover letter for the job post provided in $ARGUMENTS.

## Process

1. Read the job post from $ARGUMENTS:
   - If it starts with `@` it is a file reference — read that file
   - If it starts with `http` it is a URL — fetch the content
   - Otherwise treat it as a free-text job description
2. Call `list_cvs` to see available base CVs
3. Select the most relevant CV for this role and call `read_cv` to load it
4. Call `get_experience_stats` for accurate years of experience per role type
5. Write a compelling cover letter:
   - **Opening**: specific to the company and role — never start with "I am writing to apply for"
   - **Body** (2–3 paragraphs): connect experience directly to stated role requirements; cite specific projects or achievements; use accurate numbers from `get_experience_stats`
   - **Closing**: confident, concrete call to action
6. Generate a filename: `cl-{company}-{YYYY-MM-DD}.md` using today's date (lowercase, spaces as hyphens)
7. Call `write_cover_letter` with the filename and content
8. Confirm the filename saved

## Rules

- Use accurate years from `get_experience_stats` — never guess
- Target length: under 350 words (approximately one page)
- Address specific requirements from the job post, not generic ones
- If company name cannot be determined from the post, ask before saving
```

- [ ] **Step 3: Replace `internal/assets/skills/apply.md`**

```markdown
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
```

- [ ] **Step 4: Update `plugin.json`**

Read the current `plugin.json` first, then write:

```json
{
  "name": "prospector",
  "version": "1.0.0",
  "description": "CV, cover letter, and job application manager for Claude Code",
  "author": "dev@graditya.com",
  "mcp": {
    "command": "prospector",
    "args": ["mcp"]
  },
  "skills": [
    "internal/assets/skills/customize-cv.md",
    "internal/assets/skills/write-cover-letter.md",
    "internal/assets/skills/apply.md"
  ]
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/assets/... -v
```

Expected: both tests still pass — skills have `name:` frontmatter and themes are non-empty.

- [ ] **Step 6: Commit**

```bash
git add internal/assets/skills/ plugin.json
git commit -m "feat: customize-cv, write-cover-letter, and apply skill files"
```

---

## Task 3: Update `prospector init` to install themes and skills

**Files:**
- Modify: `cmd/prospector/cmd_init.go`

`prospector init` already copies `docker-compose.yml` from beside the binary. We extend it to also extract embedded themes to `~/.config/prospector/themes/` and skills to `~/.claude/commands/prospector/`.

- [ ] **Step 1: Write failing test**

Create `cmd/prospector/init_assets_test.go`:

```go
// cmd/prospector/init_assets_test.go
package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractEmbedDir(t *testing.T) {
	dst := t.TempDir()
	err := extractEmbedDir(themesFS, "themes", dst)
	require.NoError(t, err)

	entries, err := os.ReadDir(dst)
	require.NoError(t, err)
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	assert.Contains(t, names, "minimal.css")
	assert.Contains(t, names, "modern.css")

	data, err := os.ReadFile(filepath.Join(dst, "minimal.css"))
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestExtractEmbedDirSkipsExisting(t *testing.T) {
	dst := t.TempDir()
	sentinel := filepath.Join(dst, "minimal.css")
	require.NoError(t, os.WriteFile(sentinel, []byte("custom"), 0644))

	err := extractEmbedDir(themesFS, "themes", dst)
	require.NoError(t, err)

	data, err := os.ReadFile(sentinel)
	require.NoError(t, err)
	assert.Equal(t, "custom", string(data), "existing files must not be overwritten")
}

// themesFS and skillsFS are declared in cmd_init.go (same package).
// extractEmbedDir is a helper declared in cmd_init.go.
```

- [ ] **Step 2: Run to confirm it fails**

```bash
go test ./cmd/prospector/... 2>&1 | head -5
```

Expected: compile error — `themesFS`, `skillsFS`, `extractEmbedDir` not defined.

- [ ] **Step 3: Update `cmd/prospector/cmd_init.go`**

Read the current file first. Then apply the following changes:

**Add imports and embed declarations at the top (after the existing `package main` line):**

```go
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/db"
	"github.com/spf13/cobra"
)

//go:embed ../../internal/assets/themes
var themesFS embed.FS

//go:embed ../../internal/assets/skills
var skillsFS embed.FS
```

Wait — `//go:embed` cannot traverse with `..`. Instead, import the assets package and re-export the FS values.

**Correct approach — add a new file `cmd/prospector/assets.go`:**

```go
// cmd/prospector/assets.go
package main

import (
	"github.com/oxGrad/spicebag/internal/assets"
)

var themesFS = assets.ThemesFS
var skillsFS = assets.SkillsFS
```

This avoids the `..` issue by delegating to the `internal/assets` package which has the embed directives right next to the files.

- [ ] **Step 4: Create `cmd/prospector/assets.go`**

```go
// cmd/prospector/assets.go
package main

import "github.com/oxGrad/spicebag/internal/assets"

var themesFS = assets.ThemesFS
var skillsFS = assets.SkillsFS
```

- [ ] **Step 5: Run tests again to see compile progress**

```bash
go test ./cmd/prospector/... 2>&1 | head -10
```

Expected: now fails because `extractEmbedDir` is not defined yet (but FS vars resolve).

- [ ] **Step 6: Add `extractEmbedDir` helper and theme+skill installation to `cmd_init.go`**

Read `cmd/prospector/cmd_init.go` in full, then make these additions:

**Add the helper function** (append before the closing brace at end of file):

```go
// extractEmbedDir copies all files from an embedded directory (src prefix)
// into dst on disk. Existing files are skipped (not overwritten).
func extractEmbedDir(fsys embed.FS, src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	return fs.WalkDir(fsys, src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(src, path)
		destPath := filepath.Join(dst, rel)
		if _, err := os.Stat(destPath); err == nil {
			return nil // skip existing files
		}
		data, err := fsys.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0644)
	})
}
```

**Add import for `"embed"` and `"io/fs"` to `cmd_init.go` imports.** Current imports are:
```go
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/db"
	"github.com/spf13/cobra"
)
```

Change to:
```go
import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/db"
	"github.com/spf13/cobra"
)
```

**Add theme + skill extraction inside `newInitCmd()` RunE**, after the database initialization block and before the docker-compose copy block. Insert:

```go
// extract default themes (skip any already customised by the user)
if err := extractEmbedDir(themesFS, "themes", filepath.Join(root, "themes")); err != nil {
    fmt.Fprintf(os.Stderr, "Warning: could not extract default themes: %v\n", err)
} else {
    fmt.Println("Extracted default themes to", filepath.Join(root, "themes"))
}

// install slash command skills to ~/.claude/commands/prospector/
commandsDir := filepath.Join(home, ".claude", "commands", "prospector")
if err := extractEmbedDir(skillsFS, "skills", commandsDir); err != nil {
    fmt.Fprintf(os.Stderr, "Warning: could not install skills: %v\n", err)
} else {
    fmt.Println("Installed skills to", commandsDir)
}
```

Note: `home` is already computed above (used in `registerMCPServer` context). Make sure to declare it at the top of RunE:

```go
home, _ := os.UserHomeDir()
```

If `home` is not already a local variable in RunE, add that line at the start.

- [ ] **Step 7: Show the full updated `newInitCmd` RunE for clarity**

The complete RunE after edits (only showing the body):

```go
RunE: func(cmd *cobra.Command, args []string) error {
    root := prospectorRoot()
    home, _ := os.UserHomeDir()

    dirs := []string{"cv", "cover-letters", "themes", "applications"}
    for _, d := range dirs {
        if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
            return err
        }
    }

    cfgPath := filepath.Join(root, "config.toml")
    if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
        if err := config.Save(cfgPath, config.Config{DashboardPort: 8080, GotenbergURL: "http://localhost:3000"}); err != nil {
            return err
        }
        fmt.Println("Created", cfgPath)
    }

    dbPath := filepath.Join(root, "prospector.db")
    store, err := db.Open(dbPath)
    if err != nil {
        return err
    }
    store.Close()
    fmt.Println("Initialized database at", dbPath)

    // extract default themes (skip any already customised by the user)
    if err := extractEmbedDir(themesFS, "themes", filepath.Join(root, "themes")); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: could not extract default themes: %v\n", err)
    } else {
        fmt.Println("Extracted default themes to", filepath.Join(root, "themes"))
    }

    // install slash command skills to ~/.claude/commands/prospector/
    commandsDir := filepath.Join(home, ".claude", "commands", "prospector")
    if err := extractEmbedDir(skillsFS, "skills", commandsDir); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: could not install skills: %v\n", err)
    } else {
        fmt.Println("Installed skills to", commandsDir)
    }

    // copy docker-compose.yml from next to binary to ~/.config/prospector/
    composeDest := filepath.Join(root, "docker-compose.yml")
    if _, err := os.Stat(composeDest); os.IsNotExist(err) {
        src := filepath.Join(execDir(), "docker-compose.yml")
        data, err := os.ReadFile(src)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Warning: could not copy docker-compose.yml: %v\n", err)
        } else {
            if err := os.WriteFile(composeDest, data, 0644); err != nil {
                fmt.Fprintf(os.Stderr, "Warning: could not write docker-compose.yml: %v\n", err)
            } else {
                fmt.Println("Copied docker-compose.yml to", composeDest)
            }
        }
    }

    if err := registerMCPServer(); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: could not register MCP server automatically: %v\n", err)
        fmt.Println("Add this manually to your Claude Code MCP config:")
        printMCPConfig()
    } else {
        fmt.Println("Registered MCP server with Claude Code.")
    }

    fmt.Println("\nNext steps:")
    fmt.Println("  prospector up       # start Gotenberg")
    fmt.Println("  prospector serve    # start dashboard")
    return nil
},
```

- [ ] **Step 8: Run tests**

```bash
go test ./cmd/prospector/... -v
go test ./internal/assets/... -v
go build ./...
```

Expected:
- `TestExtractEmbedDir` — PASS (extracts minimal.css and modern.css)
- `TestExtractEmbedDirSkipsExisting` — PASS (custom file not overwritten)
- `TestSkillsEmbedded`, `TestThemesEmbedded` — PASS
- `go build ./...` — PASS

- [ ] **Step 9: Verify binary output**

```bash
go build -o prospector ./cmd/prospector/ && ./prospector init --help && rm prospector
```

Expected: shows `init` command description with no errors.

- [ ] **Step 10: Commit**

```bash
git add cmd/prospector/assets.go cmd/prospector/cmd_init.go cmd/prospector/init_assets_test.go
git commit -m "feat: prospector init installs default themes and slash command skills"
```

---

## Task 4: README

**Files:**
- Modify: `README.md`

No tests — verified by reading. The README must include: what Prospector is, prerequisites, installation, first-time setup, slash commands, dashboard, and uninstall.

- [ ] **Step 1: Write `README.md`**

```markdown
# Prospector

CV, cover letter, and job application manager that integrates with Claude Code.

## What it does

- Stores base CV variants and cover letter templates as Markdown in `~/.config/prospector/`
- Extracts experience stats (years per role type) from CV frontmatter into SQLite
- Tracks job applications with status history
- Exports PDFs via Gotenberg
- Provides three slash commands in Claude Code: `/customize-cv`, `/write-cover-letter`, `/apply`
- Dashboard at `http://localhost:8080` for browsing and managing everything

## Prerequisites

- Go 1.22+ (to build from source)
- Docker (for Gotenberg PDF export)
- Claude Code

## Installation

```bash
# 1. Build and install the binary
go install github.com/oxGrad/spicebag/cmd/prospector@latest

# 2. Set up config, database, and install skills
prospector init

# 3. Start Gotenberg (PDF export service)
prospector up
```

`prospector init` creates `~/.config/prospector/`, registers the MCP server with Claude Code, installs default themes, and installs the three slash commands in `~/.claude/commands/prospector/`.

## Slash commands

Use these inside Claude Code:

| Command | What it does |
|---|---|
| `/customize-cv <job post or role>` | Tailors your most relevant base CV for a role and saves a new version |
| `/write-cover-letter <job post>` | Writes a cover letter and saves it to your cover letter library |
| `/apply <job post>` | Full application: tailored CV + cover letter + saved job post in one folder |

Pass a job post as `@file.md`, a URL, or paste the text directly as the argument.

## Dashboard

```bash
prospector serve          # foreground (logs to stdout)
prospector serve -d       # background (logs to ~/.config/prospector/prospector.log)
prospector stop           # stop the background server
```

Dashboard at `http://localhost:8080` — browse applications, CV library, cover letters, experience stats, themes.

## PDF export

The dashboard's "Export PDF" button and the `export_pdf` MCP tool both require Gotenberg:

```bash
prospector up    # starts docker compose with Gotenberg on port 3000
```

## CV frontmatter (for experience stats)

Add structured experience to your CV markdown files:

```yaml
---
experience:
  - role_type: backend
    company: Acme Corp
    start: "2020-01-01"
    end: "2023-06-01"
  - role_type: devops
    company: Acme Corp
    start: "2021-06-01"
    end: "2023-06-01"
---
# Your CV content here
```

Then run `prospector sync` (or click "Sync from CVs" in the dashboard) to refresh stats.

## Themes

Place CSS files in `~/.config/prospector/themes/`. Two defaults are installed by `prospector init`:
- `minimal` — serif typography, clean layout
- `modern` — sans-serif with blue accents

Upload custom themes via the dashboard or drop `.css` files directly into the themes directory.

## Uninstall

```bash
rm $(which prospector)
rm -rf ~/.config/prospector
rm -rf ~/.claude/commands/prospector
# remove from ~/.claude/mcp.json: delete the "prospector" entry under "mcpServers"
```
```

- [ ] **Step 2: Build and run full test suite**

```bash
go test ./... -count=1
go build -ldflags="-s -w" -o prospector ./cmd/prospector/ && rm prospector
```

Expected: all tests pass, release build succeeds.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: README with installation, slash commands, and dashboard usage"
```

---

## What's next

Plan 3 completes the Prospector feature set. Remaining work (out of scope for this plan):

- **GoReleaser**: Configure `.goreleaser.yml` for binary releases + Homebrew tap
- **Marketplace publishing**: Submit `plugin.json` + skill files to Claude Code marketplace
- **Analytics dashboard page**: Application funnel, offer rate, ghost rate queries (spec §8)
