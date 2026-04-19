# Inline Agent Definition Content in Dot Panel

**Date**: 2026-04-15
**Status**: Approved
**Scope**: `dashboard.html` only — no server-side changes

## Problem

Clicking an agent dot opens a detail panel showing structured fields from the overview API (status, phase, tier, model, config, dependencies, watch patterns), but none of the rich narrative content from the agent's `.md` definition file. Users must click through to GitHub to understand what an agent is, what it produces, and how it relates to other agents.

## Solution

Fetch the agent's `.md` file from GitHub at click time, parse it into sections, and render the content inline below the existing structured fields in the dot detail panel.

## Data Source

```
https://raw.githubusercontent.com/transpara-ai/hive/main/agents/{role}.md
```

- Role name mapped via the existing `AGENT_MD_NAME` lookup table.
- Agents where `AGENT_MD_NAME[role] === null` (no `.md` file exists) skip the fetch entirely and show "Definition not yet created" (existing behavior).

## Caching

A JS `Map` (`agentMdCache`) stores parsed results keyed by role name. First click fetches and caches; subsequent clicks for the same role use the cache. Cache lives for the browser session — no persistence needed.

## Markdown Parsing

Lightweight, purpose-built parser — no external library:

1. Split file content on `## ` headings.
2. Each split produces a `{ heading, body }` pair.
3. Body text is rendered as paragraphs (split on double newlines).
4. Lines starting with `- ` are rendered as `<ul><li>` lists.
5. Inline backticks rendered as `<code>`.
6. Fenced code blocks (triple backtick) rendered as `<pre><code>`.
7. Everything else rendered as plain text.

This covers the actual formatting used in the agent `.md` files without needing a full markdown engine.

## Panel Layout

Top section (unchanged):
- Existing structured fields from the overview API: Status, Phase, Tier, Purpose, Category, Origin, Has prompt, Has persona, Graduated, Definition link, Configuration, Dependencies, Watch Patterns.

New section below:
- Heading: "Agent Definition"
- Loading state: "Loading definition..." placeholder text
- Error state: "Definition not available" with link to GitHub page
- Success state: parsed `.md` sections as collapsible blocks

## Section Visibility

Expanded by default:
- **Identity**
- **Purpose**
- **What You Produce**
- **Relationships**

Collapsed by default (click header to expand):
- Soul
- Execution Mode
- What You Watch
- Observation Context
- Institutional Knowledge
- Authority
- Techniques / Quality Criteria
- Channel Protocol
- Anti-patterns

Sections not present in a given `.md` file are simply omitted.

## Collapsible Section UI

Each section is a `<div>` with:
- A clickable header showing the section name and a chevron indicator (right when collapsed, down when expanded).
- A body `<div>` toggled via `display: none` / `display: block`.
- CSS transition on the chevron rotation.
- Styled consistently with existing panel sections (uses `makeSection` pattern where possible).

## Loading UX

1. Panel opens immediately with structured fields (from overview API — already available).
2. Below the structured fields, a "Loading definition..." text appears.
3. Fetch fires asynchronously.
4. On success: placeholder replaced with parsed collapsible sections.
5. On failure (404, network error, timeout): placeholder replaced with "Definition not available" and a "View on GitHub" link (using existing `agentMdUrl()`).
6. On cache hit: content renders synchronously — no loading state shown.

## Refresh Behavior

When the panel is refreshed during a poll cycle (`isRefresh === true`):
- Structured fields update as they do today.
- Markdown content is NOT re-fetched — the cached version persists.
- If the cache is empty (fetch was still in-flight or failed), the loading/error state remains.

## Implementation Scope

All changes are in `dashboard.html`:
- New `agentMdCache` variable (JS `Map` or plain object).
- New `fetchAgentMd(role, callback)` function — fetches, parses, caches.
- New `parseAgentMd(rawText)` function — splits on `## ` headings, returns array of `{ heading, body }`.
- New `renderAgentMd(sections, container)` function — builds collapsible DOM.
- New CSS for collapsible sections (`.md-section`, `.md-section-header`, `.md-section-body`, `.md-chevron`).
- Modified `showOpsDotPanel` — after building structured fields, appends loading placeholder and kicks off fetch.
- Modified `showOpsAgentPanel` — same treatment for running agents (they also benefit from definition context).

## Out of Scope

- Full markdown rendering (tables, images, nested blockquotes).
- Server-side changes to the work-server.
- Editing or updating agent `.md` files from the dashboard.
- Caching across browser sessions (localStorage).
