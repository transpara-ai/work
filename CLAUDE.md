# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Repo Is

Static front-end assets for the Transpara AI hive on nucbuntu. No build step, no framework, no server-side rendering — a single HTML file served by the work-server at `/telemetry/`, or opened directly in a browser with query params.

This is one of three repos in the telemetry system:
- **lovyou-ai-hive** — telemetry writer (Go, runs in the hive process)
- **lovyou-ai-work** — telemetry API (Go, runs in the work-server)
- **summary** (this repo) — dashboard and static pages (HTML only)

## Files

- `dashboard.html` — Unified telemetry page with two views: **Mission Control** (polls `GET {api}/telemetry/status` every 10s) and **Architecture** (polls `GET {api}/telemetry/overview` every 30s). Configured via URL query params `?api=...&key=...`.
- `docs/designs/` — Design documents for the telemetry/mission-control system.

## Development

No build, no install, no tests. Open HTML files directly in a browser or serve with:

```
python3 -m http.server 9000
```

Dashboard requires a running work-server instance for live data. Without `api` and `key` query params, it shows a configuration screen.

## Architecture Notes

- `dashboard.html` is fully self-contained — all CSS and JS are inline. No external dependencies, no CDN links.
- Uses vanilla JS fetch to poll the telemetry API. Connection state is tracked with a pulsing indicator (green=live, gray=stale, red=failed).
- Top-level view switcher: Mission Control (ops monitoring) and Architecture (structural understanding). Only the active view's endpoint is polled.
- The work-server has `Access-Control-Allow-Origin: *` so cross-origin fetch works from any serving origin.
