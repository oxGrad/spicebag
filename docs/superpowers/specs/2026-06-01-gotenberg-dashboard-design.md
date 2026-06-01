# Gotenberg Dashboard Control — Design Spec

**Date:** 2026-06-01

## Overview

Remove the `spicebag up` CLI command and move Gotenberg start/stop/healthcheck into the web dashboard. Controls appear inline on pages that have a PDF export button (CV view, CL view), so users only see Gotenberg status when they're about to export.

## HTTP Endpoints

Three new endpoints added to `Server` in a new file `internal/dashboard/handlers_gotenberg.go`:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/gotenberg/status` | Healthcheck Gotenberg, return HTML partial |
| `POST` | `/gotenberg/start` | Run `docker compose … up -d`, return updated partial |
| `POST` | `/gotenberg/stop` | Run `docker compose … stop gotenberg`, return updated partial |

**Healthcheck:** `GET <cfg.GotenbergURL>/health` with a 2-second timeout. HTTP 200 = running; any error or non-200 = stopped.

**Start:** `docker compose -f <spicebag-root>/docker-compose.yml up -d`

**Stop:** `docker compose -f <spicebag-root>/docker-compose.yml stop gotenberg`

Both start/stop shell out via `exec.Command`, capture stderr, and return the updated status partial regardless of outcome (so the UI always reflects real state after the action).

## UI / Template Changes

### New partial: `internal/dashboard/templates/gotenberg_status.html`

Defines one template named `gotenberg-status`. Contains:
- Colored dot + label: green "Gotenberg running" or red "Gotenberg stopped"
- Start button (shown when stopped): `hx-post="/gotenberg/start"`, `hx-swap="outerHTML"`, targets the widget
- Stop button (shown when running): `hx-post="/gotenberg/stop"`, `hx-swap="outerHTML"`, targets the widget
- Export button:
  - **Running:** normal submit button, enabled
  - **Stopped:** `disabled` attribute + `title="Start Gotenberg to export PDF"` tooltip

The partial is self-contained — both status and export button live inside it so a single swap updates everything atomically.

### CV view (`cv_view.html`) and CL view (`cl_view.html`)

Replace the static export form/button with:
```html
<div hx-get="/gotenberg/status" hx-trigger="load" hx-swap="outerHTML">
  <!-- loading placeholder -->
</div>
```

This fetches the status partial on page load. After start/stop, the widget swaps itself in place.

## Data Flow

```
Page load
  → hx-get /gotenberg/status
  → handler GETs cfg.GotenbergURL/health (2s timeout)
  → returns gotenberg-status partial (running or stopped state)
  → export button enabled/disabled accordingly

User clicks Start
  → hx-post /gotenberg/start
  → handler runs docker compose up -d
  → handler re-healthchecks
  → returns updated gotenberg-status partial
  → widget swaps in place

User clicks Stop
  → hx-post /gotenberg/stop
  → handler runs docker compose stop gotenberg
  → handler re-healthchecks
  → returns updated gotenberg-status partial
  → widget swaps in place
```

## CLI Change

- Delete `cmd/spicebag/cmd_up.go`
- Remove `newUpCmd()` registration from `cmd/spicebag/main.go`
- No replacement or deprecation notice — command disappears cleanly

## Error Handling

- If docker compose binary is not found: status widget shows stopped + an error message below the dot
- If start/stop exits non-zero: re-healthcheck and render actual state with stderr appended as a small error note
- Healthcheck timeout (2s) treats timeout as stopped

## Testing

- Unit tests in `handlers_gotenberg_test.go` using `httptest` — mock the Gotenberg URL with an `httptest.Server` returning 200 or connection-refused
- Start/stop handlers tested with a fake `docker` binary on PATH (or by injecting the compose command as a field on Server)
- Existing CV/CL view tests remain unchanged since the export widget is fetched via htmx (separate request)