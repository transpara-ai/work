# Inline Agent Definition Panel — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fetch agent `.md` definition files from GitHub and render their content inline in the dot/agent detail panel, below the existing structured fields.

**Architecture:** Client-side fetch from `raw.githubusercontent.com`, parsed with a lightweight `## `-splitter, cached in a JS object for the session. Collapsible sections with Identity, Purpose, What You Produce, and Relationships expanded by default.

**Tech Stack:** Vanilla JS, inline CSS — no dependencies. All changes in `dashboard.html`.

---

### Task 1: Add CSS for collapsible markdown sections

**Files:**
- Modify: `dashboard.html:666` (after the `.panel-body .dot` block, within the `<style>` tag)

- [ ] **Step 1: Add the collapsible section CSS**

Insert after line 668 (after `.panel-body .dot { ... }`) in the `<style>` block:

```css
/* Agent definition (collapsible markdown sections) */
.md-section { border-top: 1px solid var(--border); }
.md-section:first-child { border-top: none; }
.md-section-header {
  display: flex; align-items: center; gap: 0.5rem; padding: 0.5rem 0;
  cursor: pointer; user-select: none;
}
.md-section-header:hover { opacity: 0.8; }
.md-chevron {
  font-size: 10px; color: var(--text-dim); transition: transform 0.15s;
  display: inline-block; width: 12px; text-align: center;
}
.md-section.open .md-chevron { transform: rotate(90deg); }
.md-section-title {
  font-size: 11px; font-weight: 700; text-transform: uppercase;
  letter-spacing: 0.06em; color: var(--text-sec);
}
.md-section-body {
  display: none; padding: 0 0 0.75rem 0;
  font-size: 12px; line-height: 1.6; color: var(--text-sec);
}
.md-section.open .md-section-body { display: block; }
.md-section-body p { margin: 0 0 0.5rem 0; }
.md-section-body p:last-child { margin-bottom: 0; }
.md-section-body ul { margin: 0 0 0.5rem 0; padding-left: 1.25rem; }
.md-section-body li { margin-bottom: 0.25rem; }
.md-section-body code {
  font-family: var(--mono); font-size: 11px;
  background: rgba(255,255,255,0.06); padding: 1px 4px; border-radius: 3px;
}
.md-section-body pre {
  background: rgba(0,0,0,0.3); padding: 0.75rem; border-radius: var(--radius-sm);
  overflow-x: auto; margin: 0.5rem 0;
}
.md-section-body pre code { background: none; padding: 0; }
.md-section-body blockquote {
  border-left: 3px solid var(--border-light); padding-left: 0.75rem;
  color: var(--text-dim); font-style: italic; margin: 0.5rem 0;
}
.md-loading {
  font-size: 11px; color: var(--text-dim); font-style: italic;
  padding: 0.75rem 0;
}
```

- [ ] **Step 2: Verify the file is valid**

Open `dashboard.html` in a browser (or via the work-server). Confirm no visual regressions — the new CSS classes are unused so far and should not affect existing elements.

- [ ] **Step 3: Commit**

```bash
git add dashboard.html
git commit -m "style: add CSS for collapsible agent definition sections"
```

---

### Task 2: Add markdown parser and cache

**Files:**
- Modify: `dashboard.html:974` (after the `agentMdUrl` function, before `var NOISE_TYPES`)

- [ ] **Step 1: Add the `agentMdCache` object and `AGENT_MD_RAW_BASE` constant**

Insert after line 974 (after `agentMdUrl` closing brace), before `var NOISE_TYPES`:

```js
  var AGENT_MD_RAW_BASE = "https://raw.githubusercontent.com/transpara-ai/hive/main/agents/";
  var agentMdCache = {};

  function agentMdRawUrl(role) {
    var key = role.toLowerCase();
    if (!(key in AGENT_MD_NAME) || AGENT_MD_NAME[key] === null) return null;
    return AGENT_MD_RAW_BASE + AGENT_MD_NAME[key] + ".md";
  }
```

- [ ] **Step 2: Add the `parseAgentMd` function**

Insert immediately after:

```js
  function parseAgentMd(raw) {
    var sections = [];
    var parts = raw.split(/^## /m);
    for (var i = 1; i < parts.length; i++) {
      var nlIdx = parts[i].indexOf("\n");
      if (nlIdx === -1) continue;
      var heading = parts[i].substring(0, nlIdx).trim();
      var body = parts[i].substring(nlIdx + 1).trim();
      if (heading && body) sections.push({ heading: heading, body: body });
    }
    return sections;
  }
```

- [ ] **Step 3: Add the `fetchAgentMd` function**

Insert immediately after:

```js
  function fetchAgentMd(role, callback) {
    var key = role.toLowerCase();
    if (agentMdCache[key]) { callback(agentMdCache[key]); return; }
    var url = agentMdRawUrl(key);
    if (!url) { callback(null); return; }
    fetch(url).then(function (res) {
      if (!res.ok) throw new Error("HTTP " + res.status);
      return res.text();
    }).then(function (text) {
      var sections = parseAgentMd(text);
      agentMdCache[key] = sections;
      callback(sections);
    }).catch(function () {
      callback(null);
    });
  }
```

- [ ] **Step 4: Commit**

```bash
git add dashboard.html
git commit -m "feat: add agent markdown parser, cache, and fetch helper"
```

---

### Task 3: Add DOM renderer for parsed markdown sections

**Files:**
- Modify: `dashboard.html` (immediately after `fetchAgentMd`, before `var NOISE_TYPES`)

- [ ] **Step 1: Add the `renderMdBody` helper**

This converts a markdown body string into DOM nodes (paragraphs, lists, code blocks, blockquotes). It extracts fenced code blocks first (they can contain blank lines) then splits remaining text on double-newlines:

```js
  function renderMdBody(body) {
    var frag = document.createDocumentFragment();
    // Extract fenced code blocks before splitting
    var tokens = [];
    var rest = body;
    while (true) {
      var start = rest.indexOf("```");
      if (start === -1) { tokens.push({ type: "text", value: rest }); break; }
      if (start > 0) tokens.push({ type: "text", value: rest.substring(0, start) });
      var end = rest.indexOf("```", start + 3);
      if (end === -1) { tokens.push({ type: "text", value: rest }); break; }
      tokens.push({ type: "code", value: rest.substring(start, end + 3) });
      rest = rest.substring(end + 3);
    }

    tokens.forEach(function (tok) {
      if (tok.type === "code") {
        var code = tok.value.replace(/^```\w*\n?/, "").replace(/\n?```$/, "");
        var pre = el("pre"); pre.appendChild(el("code", { text: code }));
        frag.appendChild(pre);
        return;
      }
      var blocks = tok.value.split(/\n\n+/);
      for (var i = 0; i < blocks.length; i++) {
        var block = blocks[i].trim();
        if (!block) continue;

        // Blockquote
        if (block.charAt(0) === ">") {
          var bq = document.createElement("blockquote");
          bq.textContent = block.replace(/^>\s*/gm, "");
          frag.appendChild(bq);
          continue;
        }

        // Unordered list
        var lines = block.split("\n");
        if (lines[0].match(/^[-*]\s/)) {
          var ul = document.createElement("ul");
          lines.forEach(function (ln) {
            var txt = ln.replace(/^[-*]\s+/, "");
            if (txt) ul.appendChild(el("li", { text: txt }));
          });
          frag.appendChild(ul);
          continue;
        }

        // Numbered list
        if (lines[0].match(/^\d+\.\s/)) {
          var ol = document.createElement("ol");
          lines.forEach(function (ln) {
            var txt = ln.replace(/^\d+\.\s+/, "");
            if (txt) ol.appendChild(el("li", { text: txt }));
          });
          frag.appendChild(ol);
          continue;
        }

        // Paragraph
        var p = document.createElement("p");
        p.textContent = block.replace(/\n/g, " ");
        frag.appendChild(p);
      }
    });
    return frag;
  }
```

- [ ] **Step 2: Add the `renderAgentMd` function**

This builds the collapsible sections container:

```js
  var MD_EXPANDED = ["identity", "purpose", "what you produce", "relationships"];

  function renderAgentMd(sections, container) {
    sections.forEach(function (sec) {
      var div = el("div", { cls: "md-section" });
      var isOpen = MD_EXPANDED.indexOf(sec.heading.toLowerCase()) !== -1;
      if (isOpen) div.classList.add("open");

      var header = el("div", { cls: "md-section-header" });
      header.appendChild(el("span", { cls: "md-chevron", text: "\u25B6" }));
      header.appendChild(el("span", { cls: "md-section-title", text: sec.heading }));
      header.addEventListener("click", function () { div.classList.toggle("open"); });
      div.appendChild(header);

      var body = el("div", { cls: "md-section-body" });
      body.appendChild(renderMdBody(sec.body));
      div.appendChild(body);

      container.appendChild(div);
    });
  }
```

- [ ] **Step 3: Commit**

```bash
git add dashboard.html
git commit -m "feat: add markdown-to-DOM renderer with collapsible sections"
```

---

### Task 4: Wire markdown fetch into `showOpsDotPanel`

**Files:**
- Modify: `dashboard.html:1899-1979` (the `showOpsDotPanel` function)

- [ ] **Step 1: Add a loading placeholder and kick off the fetch**

Replace the final block of `showOpsDotPanel` (lines 1973–1978) with:

Old code (lines 1973–1978):
```js
    if (isRefresh && document.body.classList.contains("panel-open")) {
      document.getElementById("panel-title").textContent = role;
      clearEl("panel-body").appendChild(frag);
    } else {
      openArchPanel("ops-dot", role, role, frag);
    }
```

New code:
```js
    // Agent Definition section — loading placeholder
    var mdWrap = el("div", { cls: "p-section" });
    mdWrap.appendChild(el("div", { cls: "p-section-title", text: "Agent Definition" }));
    var mdContent = el("div");
    if (agentMdCache[role]) {
      renderAgentMd(agentMdCache[role], mdContent);
    } else if (agentMdRawUrl(role)) {
      mdContent.appendChild(el("div", { cls: "md-loading", text: "Loading definition\u2026" }));
      fetchAgentMd(role, function (sections) {
        if (!selectedOpsDot || selectedOpsDot.role !== role) return;
        mdContent.textContent = "";
        if (sections && sections.length) {
          renderAgentMd(sections, mdContent);
        } else {
          var fallback = el("div", { cls: "md-loading" });
          fallback.appendChild(document.createTextNode("Definition not available \u2014 "));
          var ghLink = document.createElement("a");
          ghLink.href = agentMdUrl(role) || "#"; ghLink.target = "_blank"; ghLink.rel = "noopener";
          ghLink.textContent = "View on GitHub";
          ghLink.style.cssText = "color:var(--blue);text-decoration:none;font-size:11px";
          fallback.appendChild(ghLink);
          mdContent.appendChild(fallback);
        }
      });
    }
    mdWrap.appendChild(mdContent);
    frag.appendChild(mdWrap);

    if (isRefresh && document.body.classList.contains("panel-open")) {
      document.getElementById("panel-title").textContent = role;
      clearEl("panel-body").appendChild(frag);
    } else {
      openArchPanel("ops-dot", role, role, frag);
    }
```

- [ ] **Step 2: Test in browser**

Open the dashboard connected to the work-server. Click an agent dot (e.g., guardian). Verify:
1. Structured fields appear immediately at top (Status, Phase, Tier, etc.)
2. "Loading definition..." appears briefly below
3. Markdown content replaces the loading text with collapsible sections
4. Identity, Purpose, What You Produce, and Relationships are expanded
5. Other sections (Soul, Authority, etc.) are collapsed — click to expand
6. Click a dot for an agent with `AGENT_MD_NAME[role] === null` (e.g., taskmanager) — no loading placeholder, no fetch
7. On poll refresh, the cached content re-renders instantly (no flicker)

- [ ] **Step 3: Commit**

```bash
git add dashboard.html
git commit -m "feat: show agent definition content in dot detail panel"
```

---

### Task 5: Wire markdown fetch into `showOpsAgentPanel`

**Files:**
- Modify: `dashboard.html:1744-1808` (the `showOpsAgentPanel` function)

- [ ] **Step 1: Add definition section to the agent panel**

Insert before the final panel-open block (before the line `// Open or update panel`). The existing code ends with:

```js
    // Last message
    if (a.last_message) {
      frag.appendChild(makeSection("Last Message", el("pre", { cls: "last-message", text: a.last_message })));
    }
```

Insert after that block, before `// Open or update panel`:

```js
    // Agent Definition section
    var mdWrap = el("div", { cls: "p-section" });
    mdWrap.appendChild(el("div", { cls: "p-section-title", text: "Agent Definition" }));
    var mdContent = el("div");
    var mdRole = (a.role || "").toLowerCase();
    if (agentMdCache[mdRole]) {
      renderAgentMd(agentMdCache[mdRole], mdContent);
    } else if (agentMdRawUrl(mdRole)) {
      mdContent.appendChild(el("div", { cls: "md-loading", text: "Loading definition\u2026" }));
      fetchAgentMd(mdRole, function (sections) {
        if (selectedOpsAgent !== a.role) return;
        mdContent.textContent = "";
        if (sections && sections.length) {
          renderAgentMd(sections, mdContent);
        } else {
          var fallback = el("div", { cls: "md-loading" });
          fallback.appendChild(document.createTextNode("Definition not available \u2014 "));
          var ghLink = document.createElement("a");
          ghLink.href = agentMdUrl(mdRole) || "#"; ghLink.target = "_blank"; ghLink.rel = "noopener";
          ghLink.textContent = "View on GitHub";
          ghLink.style.cssText = "color:var(--blue);text-decoration:none;font-size:11px";
          fallback.appendChild(ghLink);
          mdContent.appendChild(fallback);
        }
      });
    }
    mdWrap.appendChild(mdContent);
    frag.appendChild(mdWrap);
```

- [ ] **Step 2: Test in browser**

Click a running agent card (not a dot). Verify:
1. Runtime status section appears at top (State, Model, Iterations, Cost, etc.)
2. "Agent Definition" section appears below with collapsible markdown sections
3. On poll refresh (every 10s), runtime data updates but definition stays cached (no re-fetch)

- [ ] **Step 3: Commit**

```bash
git add dashboard.html
git commit -m "feat: show agent definition content in running agent panel"
```

---

### Task 6: Final integration test

**Files:**
- No changes — testing only

- [ ] **Step 1: Test dot panel for agent with `.md` file (e.g., guardian)**

Click the guardian dot. Verify:
- Structured fields at top (Status, Phase, Tier, Purpose, Category, etc.)
- "Agent Definition" heading below
- Identity expanded — shows "You are the Guardian of the hive..."
- Purpose expanded — shows "You watch all activity across all agents..."
- What You Produce expanded — shows "HALT signals when invariants are violated..."
- Relationships expanded (if present in guardian.md)
- Soul, Execution Mode, Authority — collapsed, expandable on click

- [ ] **Step 2: Test dot panel for agent without `.md` file (e.g., taskmanager)**

Click the taskmanager dot. Verify:
- Structured fields appear
- No "Loading definition..." text
- No "Agent Definition" section (or section is absent since `AGENT_MD_NAME[taskmanager] === null`)

- [ ] **Step 3: Test running agent panel**

If any agents are running, click an agent card. Verify:
- Runtime status at top
- Agent Definition section below with collapsible markdown

- [ ] **Step 4: Test cache behavior**

Click guardian dot, wait for content to load. Close panel. Click guardian dot again. Verify content appears instantly (no "Loading..." flash).

- [ ] **Step 5: Test network failure**

Temporarily change `AGENT_MD_RAW_BASE` to an invalid URL (e.g., `https://raw.githubusercontent.com/transpara-ai/INVALID/`). Click a dot. Verify:
- "Loading definition..." appears
- Replaced by "Definition not available — View on GitHub" with working link
- Revert the URL change

- [ ] **Step 6: Test panel refresh**

Click a dot, wait for content to load. Wait for the 10s poll cycle. Verify:
- Structured fields update
- Markdown content stays in place (no re-fetch, no flicker)
