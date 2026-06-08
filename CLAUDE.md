# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build (full — installs npm deps, builds frontend, then Go binary)
just build

# Build for local dev (skips npm ci, assumes deps installed)
just dev           # builds frontend + Go, then runs spicebag start

# Build steps individually
just build-frontend      # npm ci + vite build → internal/dashboard/ui/
just build-frontend-dev  # vite build only (no npm ci)
just build-go            # go build ./cmd/spicebag/ (requires frontend already built)

# Test
just test                # go test ./...
go test ./internal/mcp/... -v -run TestSomething   # single test

# Clean
just clean               # removes ./spicebag binary and internal/dashboard/ui/
```

`CGO_ENABLED=0` — the SQLite driver is pure Go (`modernc.org/sqlite`); no C toolchain needed.

## Architecture

**Build dependency:** The Vue frontend (`frontend/`) must be built before the Go binary. `vite build` writes its output to `internal/dashboard/ui/`, which the Go binary embeds via `//go:embed ui` in `internal/dashboard/server.go`. If `internal/dashboard/ui/` is missing or stale, the dashboard will error.

**Data directory:** All runtime data lives in `~/.config/spicebag/`. `cmd/spicebag/cmd_init.go:spicebagRoot()` returns this path. Both `start` and `mcp` auto-run init on first launch if `config.toml` is absent.

**Two servers, one binary:**
- `spicebag start` — HTTP dashboard server (`internal/dashboard/`) serving a REST API and the embedded Vue SPA
- `spicebag mcp` — stdio MCP server (`internal/mcp/`) invoked automatically by Claude Code; exposes all tools listed in README

**Internal package layout:**
- `internal/config` — TOML config (`~/.config/spicebag/config.toml`); dashboard port + Gotenberg URL
- `internal/db` — SQLite store; schema: `experience`, `applications`, `application_status_history`, `scraped_jobs`, `scrape_companies`, `scrape_preferences` tables; `db.SetMaxOpenConns(1)` enforces single writer
- `internal/fs` — filesystem CRUD for CVs, cover letters, themes, applications (all stored as `.html` files)
- `internal/mcp` — MCP tool registration; each domain has its own `*_tools.go` file; `server.go` wires them and exposes `CallTool` for in-process testing
- `internal/dashboard` — HTTP handlers split by domain (`handlers_cv.go`, `handlers_apps.go`, etc.); SPA fallback in `server.go:handleSPA`
- `internal/pdf` — thin Gotenberg HTTP client; inlines CSS into HTML before POSTing to `/forms/chromium/convert/html`
- `internal/scrape` — ATS adapters (Greenhouse, Lever, Ashby, SmartRecruiters, Workable, Recruitee, Breezy, BambooHR, Personio, Workday), URL detection/normalization, and the coarse remote pre-filter
- `internal/assets` — embedded default CSS themes (`minimal.css`, `modern.css`)

**Frontend:** Vue 3 + Vue Router + Tailwind CSS, built with Vite. Single-page app; all navigation is client-side. Talks to the Go backend via `frontend/src/api.js`.

**Plugin distribution:** `.claude-plugin/plugin.json` declares the MCP server and slash command paths (`plugins/skills/`). The plugin marketplace installs these; `spicebag init` handles the same setup for direct installs.

**Testing approach:** Go tests use `httptest.NewServer` for dashboard handlers and `mcp.Server.CallTool` (in-process MCP client) for MCP tools. Tests create a temp dir as the data root — no global state.
