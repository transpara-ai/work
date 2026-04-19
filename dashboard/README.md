# lovyou.ai — Summary

Static resources for the Transpara AI hive on nucbuntu.

## Files

| File | Description |
|------|-------------|
| `index.html` | Static architecture poster — complete dependency hierarchy |
| `dashboard.html` | Live mission control dashboard — polls the telemetry API |

---

## Mission Control Dashboard

`dashboard.html` is a standalone HTML file that polls the work-server telemetry API
and renders live hive state. No build step, no framework, no server-side rendering.

### Usage

```
dashboard.html?api=http://nucbuntu:8080&key=YOUR_API_KEY
```

| Parameter | Required | Description |
|-----------|----------|-------------|
| `api` | Yes | Work-server base URL (e.g. `http://nucbuntu:8080`) |
| `key` | Yes | Bearer token for API authentication |

If either parameter is missing, the dashboard shows a configuration screen
with input fields instead of the live view.

### Finding the API URL and key

The work-server runs on nucbuntu. The API URL is `http://nucbuntu:8080`
(or whichever port is configured). The API key is set via the `API_KEY`
environment variable in the work-server's Docker Compose configuration.

### What it shows

Five sections, all on one scrollable page:

- **Connection indicator** — green pulsing dot when live, gray when stale, red on failure
- **Expansion phases** — Phase 0–8 timeline with live status (complete / in_progress / blocked)
- **Agent status** — Cards for each running agent (guardian, sysmon, allocator, strategist,
  planner, implementer). Click any card to expand and see the last LLM message.
- **Hive health** — Active agents, chain length and integrity, event rate, daily cost vs cap
- **Event stream** — Last 50 events, newest first, with color-coded event type badges

### Serving

The dashboard can be served from:

- GitHub Pages: `transpara-ai.github.io/summary/dashboard.html?api=...&key=...`
- A file server on nucbuntu: `python3 -m http.server 9000`
- Opened directly in a browser as a local file (CORS must be open on the work-server, which it is)

The work-server already has `Access-Control-Allow-Origin: *` middleware,
so cross-origin fetch from any of these origins will work.

### Polling

The dashboard polls `GET {api}/telemetry/status` every 10 seconds.
If the hive is offline, the last known state is preserved with a stale indicator —
no fake green lights.
