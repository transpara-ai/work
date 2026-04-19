# Per-Agent Task Pane — Design Spec

## Problem

The dashboard has no task visibility. 24 tasks exist in the work-server but operators can't see them. Showing all tasks in a flat list would be noisy and unactionable. Operators need per-agent task context when investigating a specific agent.

## Solution

A slide-in right panel triggered by clicking an agent card header. Shows only tasks assigned to the clicked agent. The rest of the dashboard shifts left to accommodate.

## Interaction

1. **No card selected (default):** Dashboard renders full-width, no pane visible.
2. **Click agent card header:** Pane slides in from the right (~380px wide). Dashboard content shrinks via right padding. Clicked card gets a `var(--blue)` border highlight.
3. **Click same card header or X button:** Pane closes, dashboard returns to full width.
4. **Click different card header:** Pane content switches to the new agent's tasks (no close/reopen animation). Previous card loses highlight, new card gains it.
5. **Card body click:** Existing expand/collapse behavior is preserved. Only the `.agent-card-head` element triggers the task pane — not the card body.
6. **Missing actor_id:** If an agent has no `actor_id`, the card header click is a no-op (no pane opens).

## Data Source

- **Endpoint:** `GET /tasks?assignee={actor_id}` on the work-server.
- **Auth:** Uses the same conditional as status polling: when `USE_COOKIE_AUTH` is true (embedded mode), send `{ credentials: "same-origin" }`. Otherwise send `{ headers: { Authorization: "Bearer " + API_KEY } }`.
- **Fetch timing:** On card header click only. No auto-refresh or polling. Re-clicking the active card re-fetches.
- **AbortController:** Cancel any in-flight task fetch when a new card is clicked (same pattern as status polling).

## Task API Response Shape

```json
{
  "tasks": [
    {
      "id": "019d63ca-59c2-...",
      "title": "Scrape loop and main wiring",
      "description": "Implement the main scrape loop...",
      "status": "assigned",
      "priority": "high",
      "assignee": "actor_13c4fb4af6dc7d3def47139827601e53",
      "created_by": "actor_0be01847499b9ba7ff5a27524a2eab36",
      "blocked": false
    }
  ]
}
```

Fields used by the pane: `title`, `status`, `priority`, `description`.

## Pane Layout

```
+---------------------------------------+
| [X]  {Role} — Tasks (N)              |
+---------------------------------------+
| +-----------------------------------+ |
| | Title of task                     | |
| | [assigned] [high]                 | |
| | First line of description...      | |
| +-----------------------------------+ |
| +-----------------------------------+ |
| | Another task title                | |
| | [pending] [medium]               | |
| | First line of description...      | |
| +-----------------------------------+ |
|                                       |
| (click task card to expand desc)      |
+---------------------------------------+
```

- **Header:** Close button (X), agent role capitalized, "Tasks (N)" count.
- **Task card:** Title (bold), status badge (colored: green=completed, blue=assigned, gray=pending), priority badge, single-line description truncated with ellipsis.
- **Expand:** Click a task card to expand its full description. Click again to collapse.
- **Empty state:** "No tasks assigned" centered in the pane body.
- **Loading state:** Simple "Loading..." text while fetch is in-flight.
- **Error state:** "Failed to load tasks" with a retry link.

## CSS

- Pane: `position: fixed; right: 0; top: 0; bottom: 0; width: 380px; background: var(--bg); border-left: 1px solid var(--border); z-index: 10; overflow-y: auto; transform: translateX(100%); transition: transform 0.2s ease;`
- Pane open: `transform: translateX(0);`
- Dashboard shift: `#dashboard { transition: padding-right 0.2s ease; }` — set `padding-right: 390px` via JS when pane is open. This works with the existing flex column layout by shrinking the content area rather than offsetting it.
- Selected card: `border-color: var(--blue);`
- Responsive: On narrow viewports, pane overlays the content with `box-shadow` instead of shifting. Use `@media (max-width: 768px)` to match the existing responsive breakpoint at line 769.

## Status Badges

| Task status | Color | Badge text |
|---|---|---|
| assigned | `var(--blue)` / `#3b82f6` | assigned |
| pending | `var(--text-dim)` | pending |
| completed | `var(--green)` | done |

## Priority Badges

| Priority | Style |
|---|---|
| high | Red text |
| medium | Amber text |
| low | Dim text |
| (missing) | No badge |

## Implementation Constraints

- All CSS and JS inline in `dashboard.html` — no external files (per CLAUDE.md).
- Vanilla JS only — no frameworks.
- Reuse existing `el()` helper for DOM construction.
- Reuse existing CSS variables for colors/spacing.
- Add pane HTML structure as a sibling to `#dashboard`, not inside it.
- Click target is `.agent-card-head` (not the whole card) to avoid conflict with the existing card expand/collapse click handler on the card body.
- Task fetch uses the same auth conditional (`USE_COOKIE_AUTH`) as status polling.

## Files Modified

- `dashboard.html` — CSS additions (~60 lines), HTML pane container, JS click handlers + fetch + render (~180 lines).

## Out of Scope

- Tasks created by the agent (only assigned-to).
- Auto-refresh / polling of tasks.
- Task mutation (create, assign, complete) from the dashboard.
- Showing tasks in the main dashboard area.
