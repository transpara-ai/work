package main

const telemetryDashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>lovyou.ai — Mission Control</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }

:root {
  --bg: #0f1117;
  --bg-card: #111827;
  --border: #1e293b;
  --border-light: #2d3748;
  --text: #e2e8f0;
  --text-head: #f8fafc;
  --text-sec: #64748b;
  --text-dim: #475569;
  --green: #22c55e;
  --green-dim: #166534;
  --amber: #f59e0b;
  --amber-dim: #78350f;
  --red: #ef4444;
  --red-dim: #7f1d1d;
  --blue: #3b82f6;
  --blue-dim: #1e3a5f;
  --gray: #64748b;
  --purple: #a855f7;
  --font: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  --mono: ui-monospace, "Cascadia Code", "Fira Code", Consolas, monospace;
  --radius: 8px;
  --radius-sm: 4px;
}

body {
  font-family: var(--font);
  background: var(--bg);
  color: var(--text);
  min-height: 100vh;
  font-size: 14px;
  line-height: 1.5;
}

/* ── CONFIG SCREEN ──────────────────────────────── */
#config-screen {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 2rem;
}

.config-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 2rem;
  width: 100%;
  max-width: 440px;
}

.config-card h1 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-head);
  margin-bottom: 0.25rem;
}

.config-subtitle {
  font-size: 12px;
  color: var(--text-sec);
  margin-bottom: 1.5rem;
}

.config-field { margin-bottom: 1rem; }

.config-field label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-sec);
  margin-bottom: 0.375rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.config-field input {
  width: 100%;
  background: var(--bg);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-sm);
  padding: 0.5rem 0.75rem;
  color: var(--text);
  font-family: var(--mono);
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s;
}

.config-field input:focus { border-color: var(--blue); }

.btn-connect {
  width: 100%;
  margin-top: 0.5rem;
  background: var(--blue);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  padding: 0.625rem 1rem;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
}

.btn-connect:hover { opacity: 0.9; }

.config-hint {
  margin-top: 1rem;
  font-size: 11px;
  color: var(--text-dim);
  line-height: 1.6;
}

.config-hint code {
  font-family: var(--mono);
  background: rgba(255,255,255,0.06);
  padding: 1px 4px;
  border-radius: 3px;
}

/* ── DASHBOARD ──────────────────────────────────── */
#dashboard {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

/* ── TOP BAR ────────────────────────────────────── */
#topbar {
  position: sticky;
  top: 0;
  z-index: 100;
  background: rgba(15,17,23,0.95);
  backdrop-filter: blur(8px);
  border-bottom: 1px solid var(--border);
  padding: 0.625rem 1.25rem;
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.topbar-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-head);
  white-space: nowrap;
}

.topbar-sub {
  font-size: 13px;
  font-weight: 400;
  color: var(--text-sec);
  white-space: nowrap;
}

.topbar-sep { color: var(--border-light); }

.conn-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  background: var(--gray);
}

.conn-dot.connected {
  background: var(--green);
  animation: pulse-green 2s ease-in-out infinite;
}

.conn-dot.stale { background: var(--amber); }
.conn-dot.error { background: var(--red); }

@keyframes pulse-green {
  0%, 100% { box-shadow: 0 0 0 0 rgba(34,197,94,0.5); }
  50%       { box-shadow: 0 0 0 5px rgba(34,197,94,0); }
}

.conn-status {
  font-size: 12px;
  color: var(--text-sec);
  display: flex;
  align-items: center;
  gap: 0.4rem;
  white-space: nowrap;
}

.conn-status.error { color: var(--red); }
.conn-status.stale { color: var(--amber); }

.topbar-api {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-dim);
  white-space: nowrap;
}

.topbar-time {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-dim);
  margin-left: auto;
  white-space: nowrap;
}

/* ── MAIN ───────────────────────────────────────── */
.main {
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  max-width: 1600px;
  margin: 0 auto;
  width: 100%;
}

/* ── SECTION ────────────────────────────────────── */
.section {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
}

.section-head {
  padding: 0.625rem 1rem;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.section-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-sec);
}

.section-meta {
  font-size: 11px;
  color: var(--text-dim);
  margin-left: auto;
}

.section-body { padding: 1rem; }

/* ── PHASE TIMELINE ─────────────────────────────── */
.phase-timeline {
  display: flex;
  align-items: flex-start;
  gap: 0;
  overflow-x: auto;
  padding: 0.5rem 0 0.75rem;
}

.phase-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
  min-width: 80px;
  position: relative;
}

.phase-item:not(:last-child)::after {
  content: '';
  position: absolute;
  top: 8px;
  left: calc(50% + 10px);
  right: calc(-50% + 10px);
  height: 2px;
  background: var(--border-light);
  z-index: 0;
}

.phase-item.complete:not(:last-child)::after { background: var(--green-dim); }
.phase-item.in_progress:not(:last-child)::after { background: var(--blue-dim); }

.phase-dot {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 2px solid var(--border-light);
  background: var(--bg);
  z-index: 1;
  position: relative;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.phase-item.complete .phase-dot   { background: var(--green); border-color: var(--green); }
.phase-item.in_progress .phase-dot {
  background: var(--blue); border-color: var(--blue);
  animation: pulse-blue 2s ease-in-out infinite;
}
.phase-item.blocked .phase-dot { opacity: 0.4; }

@keyframes pulse-blue {
  0%, 100% { box-shadow: 0 0 0 0 rgba(59,130,246,0.5); }
  50%       { box-shadow: 0 0 0 5px rgba(59,130,246,0); }
}

.phase-num {
  font-size: 10px;
  font-weight: 700;
  color: #fff;
  line-height: 1;
}

.phase-info { text-align: center; margin-top: 0.5rem; }

.phase-name {
  font-size: 11px;
  font-weight: 500;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 80px;
}

.phase-item.blocked .phase-name { color: var(--text-sec); }

.phase-ts {
  font-family: var(--mono);
  font-size: 10px;
  color: var(--text-dim);
  margin-top: 2px;
}

.phase-status-badge {
  display: inline-block;
  font-size: 9px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 1px 5px;
  border-radius: 3px;
  margin-top: 3px;
}

.phase-status-badge.complete    { background: var(--green-dim);             color: var(--green); }
.phase-status-badge.in_progress { background: var(--blue-dim);              color: var(--blue); }
.phase-status-badge.blocked     { background: rgba(100,116,139,0.15);       color: var(--gray); }

/* ── AGENT GRID ─────────────────────────────────── */
.agent-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 0.75rem;
}

.agent-card {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
  cursor: pointer;
  transition: border-color 0.15s;
}

.agent-card:hover { border-color: var(--border-light); }
.agent-card.has-errors { border-color: rgba(239,68,68,0.3); }

.agent-card-head {
  padding: 0.625rem 0.75rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.agent-role {
  font-weight: 600;
  font-size: 13px;
  color: var(--text-head);
  flex: 1;
  text-transform: capitalize;
}

.badge {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 2px 7px;
  border-radius: 99px;
  white-space: nowrap;
}

.badge-idle        { background: rgba(34,197,94,0.15);   color: var(--green); }
.badge-processing  { background: rgba(59,130,246,0.15);  color: var(--blue); }
.badge-waiting     { background: rgba(245,158,11,0.15);  color: var(--amber); }
.badge-escalating,
.badge-refusing,
.badge-suspended   { background: rgba(239,68,68,0.15);   color: var(--red); }
.badge-retiring,
.badge-retired     { background: rgba(100,116,139,0.15); color: var(--gray); }
.badge-unknown     { background: rgba(100,116,139,0.15); color: var(--gray); }

.badge-model {
  background: rgba(168,85,247,0.12);
  color: var(--purple);
  font-family: var(--mono);
  font-size: 9px;
  padding: 2px 6px;
}

.agent-card-body {
  padding: 0 0.75rem 0.625rem;
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.agent-row {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.agent-label {
  font-size: 11px;
  color: var(--text-sec);
  flex-shrink: 0;
}

.agent-val {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text);
}

.progress-bar {
  flex: 1;
  height: 4px;
  background: var(--border);
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--blue);
  border-radius: 2px;
  transition: width 0.3s;
}

.progress-fill.warn   { background: var(--amber); }
.progress-fill.danger { background: var(--red); }

.agent-event {
  font-size: 11px;
  color: var(--text-sec);
  padding-top: 0.25rem;
  border-top: 1px solid var(--border);
  display: flex;
  align-items: center;
  gap: 0.375rem;
  flex-wrap: wrap;
}

.event-type-pill {
  font-size: 10px;
  font-weight: 500;
  padding: 1px 6px;
  border-radius: 3px;
  font-family: var(--mono);
}

.agent-expand {
  display: none;
  border-top: 1px solid var(--border);
  padding: 0.625rem 0.75rem;
  flex-direction: column;
  gap: 0.5rem;
}

.agent-card.expanded .agent-expand { display: flex; }

.expand-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-dim);
  margin-bottom: 0.25rem;
}

.last-message {
  background: rgba(0,0,0,0.3);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 0.5rem;
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-sec);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 200px;
  overflow-y: auto;
  line-height: 1.5;
}

.expand-meta {
  display: flex;
  gap: 1.5rem;
  flex-wrap: wrap;
}

.expand-meta-item {
  font-size: 11px;
  color: var(--text-sec);
}

.data-empty {
  color: var(--text-sec);
  font-size: 13px;
  padding: 1.5rem;
  text-align: center;
}

/* ── HIVE HEALTH ────────────────────────────────── */
.hive-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 0.75rem;
}

.hive-card {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 0.75rem;
}

.hive-card-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-dim);
  margin-bottom: 0.375rem;
}

.hive-card-val {
  font-family: var(--mono);
  font-size: 20px;
  font-weight: 700;
  color: var(--text-head);
  line-height: 1.2;
}

.hive-card-val-sub {
  font-family: var(--mono);
  font-size: 13px;
  font-weight: 400;
  color: var(--text-sec);
}

.hive-card-sub {
  font-size: 11px;
  color: var(--text-sec);
  margin-top: 0.25rem;
}

.severity-badge {
  display: inline-block;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding: 3px 10px;
  border-radius: 99px;
  margin-top: 0.375rem;
}

.severity-ok       { background: rgba(34,197,94,0.15);  color: var(--green); }
.severity-warning  { background: rgba(245,158,11,0.15); color: var(--amber); }
.severity-critical { background: rgba(239,68,68,0.15);  color: var(--red); }

/* ── EVENT STREAM ───────────────────────────────── */
.event-stream {
  max-height: 380px;
  overflow-y: auto;
}

.event-row {
  display: flex;
  align-items: baseline;
  gap: 0.625rem;
  padding: 0.375rem 1rem;
  border-bottom: 1px solid rgba(30,41,59,0.5);
  font-size: 12px;
  overflow: hidden;
}

.event-row:last-child { border-bottom: none; }

.event-ts {
  font-family: var(--mono);
  font-size: 10px;
  color: var(--text-dim);
  white-space: nowrap;
  flex-shrink: 0;
  min-width: 52px;
}

.event-actor {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 3px;
  background: rgba(100,116,139,0.15);
  color: var(--text-sec);
  white-space: nowrap;
  flex-shrink: 0;
  text-transform: capitalize;
}

.event-type {
  font-family: var(--mono);
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 3px;
  white-space: nowrap;
  flex-shrink: 0;
}

.evt-health  { background: rgba(59,130,246,0.15);  color: var(--blue); }
.evt-budget  { background: rgba(245,158,11,0.15);  color: var(--amber); }
.evt-work    { background: rgba(34,197,94,0.15);   color: var(--green); }
.evt-state   { background: rgba(100,116,139,0.15); color: var(--gray); }
.evt-hive    { background: rgba(168,85,247,0.12);  color: var(--purple); }
.evt-default { background: rgba(100,116,139,0.1);  color: var(--text-sec); }

.event-summary {
  color: var(--text-sec);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

/* ── SCROLLBAR ──────────────────────────────────── */
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--border-light); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: var(--text-dim); }

/* ── RESPONSIVE ─────────────────────────────────── */
@media (max-width: 900px) {
  .agent-grid { grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); }
  .hive-grid  { grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); }
}

@media (max-width: 768px) {
  .main { padding: 0.75rem; gap: 0.75rem; }
  .topbar-time { display: none; }
}
</style>
</head>
<body>

<!-- Config screen (shown when URL params are missing) -->
<div id="config-screen" style="display:none">
  <div class="config-card">
    <h1>lovyou.ai Mission Control</h1>
    <p class="config-subtitle">Enter the work-server URL and API key to connect</p>
    <div class="config-field">
      <label for="cfg-api">API URL</label>
      <input id="cfg-api" type="url" placeholder="http://nucbuntu:8080" autocomplete="off" spellcheck="false">
    </div>
    <div class="config-field">
      <label for="cfg-key">API Key</label>
      <input id="cfg-key" type="password" placeholder="Bearer token" autocomplete="off" spellcheck="false">
    </div>
    <button class="btn-connect" id="cfg-btn">Connect</button>
    <p class="config-hint">
      Opens: <code>dashboard.html?api=URL&amp;key=KEY</code><br>
      Find the API URL and key in the work-server config on nucbuntu.
    </p>
  </div>
</div>

<!-- Dashboard (shown when URL params are present) -->
<div id="dashboard" style="display:none">
  <div id="topbar">
    <span class="topbar-title">lovyou.ai</span>
    <span class="topbar-sep">·</span>
    <span class="topbar-sub">Mission Control</span>
    <div class="conn-status" id="conn-status">
      <div class="conn-dot" id="conn-dot"></div>
      <span id="conn-text">Connecting…</span>
    </div>
    <span class="topbar-api" id="topbar-api"></span>
    <span class="topbar-time" id="topbar-time"></span>
  </div>

  <div class="main">
    <!-- Expansion Phases -->
    <div class="section">
      <div class="section-head">
        <span class="section-label">Expansion Phases</span>
      </div>
      <div class="section-body">
        <div class="phase-timeline" id="phase-timeline">
          <div class="data-empty">Awaiting telemetry data…</div>
        </div>
      </div>
    </div>

    <!-- Agent Status -->
    <div class="section">
      <div class="section-head">
        <span class="section-label">Agent Status</span>
        <span class="section-meta" id="agent-count"></span>
      </div>
      <div class="section-body">
        <div class="agent-grid" id="agent-grid">
          <div class="data-empty">Awaiting telemetry data…</div>
        </div>
      </div>
    </div>

    <!-- Hive Health -->
    <div class="section">
      <div class="section-head">
        <span class="section-label">Hive Health</span>
      </div>
      <div class="section-body">
        <div class="hive-grid" id="hive-grid">
          <div class="data-empty">Awaiting telemetry data…</div>
        </div>
      </div>
    </div>

    <!-- Event Stream -->
    <div class="section">
      <div class="section-head">
        <span class="section-label">Event Stream</span>
        <span class="section-meta" id="event-count"></span>
      </div>
      <div class="event-stream" id="event-stream" onscroll="onStreamScroll()">
        <div class="data-empty">No recent events</div>
      </div>
    </div>
  </div>
</div>

<script>
(function () {
  "use strict";

  // ── CONFIGURATION ──────────────────────────────────
  var API_BASE = "";             // served from same origin — relative URLs
  var API_KEY  = "{{API_KEY}}";  // injected by work-server at serve time

  document.getElementById("dashboard").style.display = "flex";
  document.getElementById("topbar-api").textContent = window.location.host;
  init();

  function connect() {
    var api = document.getElementById("cfg-api").value.trim().replace(/\/$/, "");
    var key = document.getElementById("cfg-key").value.trim();
    if (!api || !key) { alert("Both API URL and API Key are required."); return; }
    var url = new URL(window.location.href.split("?")[0]);
    url.searchParams.set("api", api);
    url.searchParams.set("key", key);
    window.location.href = url.toString();
  }

  // ── POLLING STATE ──────────────────────────────────
  var REFRESH_MS     = 10000;
  var lastSuccess    = null;
  var isUserScrolled = false;

  function init() {
    refresh();
    setInterval(refresh, REFRESH_MS);
    setInterval(tickClock, 1000);
  }

  // ── FETCH ──────────────────────────────────────────
  function refresh() {
    fetch(API_BASE + "/telemetry/status", {
      headers: { Authorization: "Bearer " + API_KEY }
    })
    .then(function (res) {
      if (res.status === 503) {
        setConnStatus("stale", "Telemetry not initialized");
        return null;
      }
      if (!res.ok) throw new Error("HTTP " + res.status);
      return res.json();
    })
    .then(function (data) {
      if (!data) return;
      lastSuccess = Date.now();
      setConnStatus("connected", "Live");
      renderPhases(data.phases || []);
      renderAgents(data.agents || []);
      renderHive(data.hive || null);
      renderEvents(data.recent_events || []);
    })
    .catch(function (err) {
      var msg = (err.message || "").indexOf("fetch") !== -1
        ? "Cannot reach API — check URL and network"
        : "Connection lost";
      setConnStatus("error", msg);
      console.error("Poll failed:", err);
    });
  }

  // ── CLOCK ──────────────────────────────────────────
  function tickClock() {
    var el = document.getElementById("topbar-time");
    if (el) el.textContent = new Date().toLocaleTimeString();

    if (lastSuccess === null) return;
    var ago = Math.floor((Date.now() - lastSuccess) / 1000);

    if (ago < 15)       setConnStatus("connected", "Live");
    else if (ago < 60)  setConnStatus("stale",     "Last update: " + ago + "s ago");
    else                setConnStatus("error",      "Connection lost (" + ago + "s ago)");
  }

  function setConnStatus(state, text) {
    document.getElementById("conn-dot").className  = "conn-dot " + state;
    document.getElementById("conn-status").className = "conn-status " + (state === "connected" ? "" : state);
    document.getElementById("conn-text").textContent = text;
  }

  // ── DOM HELPERS ────────────────────────────────────
  // Build an element with optional class, optional text content, optional style object
  function el(tag, opts) {
    var e = document.createElement(tag);
    if (opts) {
      if (opts.cls)   e.className = opts.cls;
      if (opts.text != null) e.textContent = String(opts.text);
      if (opts.title) e.title = String(opts.title);
      if (opts.style) Object.assign(e.style, opts.style);
    }
    return e;
  }

  function append(parent /*, ...children */) {
    for (var i = 1; i < arguments.length; i++) {
      if (arguments[i]) parent.appendChild(arguments[i]);
    }
    return parent;
  }

  function clearEl(id) {
    var e = document.getElementById(id);
    while (e.firstChild) e.removeChild(e.firstChild);
    return e;
  }

  // ── PHASES ────────────────────────────────────────
  function renderPhases(phases) {
    var container = clearEl("phase-timeline");

    if (!phases.length) {
      container.appendChild(el("div", { cls: "data-empty", text: "Awaiting telemetry data…" }));
      return;
    }

    phases.forEach(function (p) {
      var status = p.status || "blocked";
      var item   = el("div", { cls: "phase-item " + status });

      var dot    = el("div", { cls: "phase-dot" });
      dot.appendChild(el("span", { cls: "phase-num", text: p.phase }));
      item.appendChild(dot);

      var info   = el("div", { cls: "phase-info" });
      var name   = el("div", { cls: "phase-name", text: p.label || "Phase " + p.phase });
      name.title = p.label || "";
      info.appendChild(name);

      var ts = "";
      if (status === "complete"    && p.completed_at) ts = fmtDate(p.completed_at);
      if (status === "in_progress" && p.started_at)   ts = "since " + fmtDate(p.started_at);
      if (ts) info.appendChild(el("div", { cls: "phase-ts", text: ts }));

      var badge = el("span", { cls: "phase-status-badge " + status, text: status.replace("_", " ") });
      info.appendChild(badge);
      item.appendChild(info);
      container.appendChild(item);
    });
  }

  // ── AGENTS ────────────────────────────────────────
  function renderAgents(agents) {
    var grid  = clearEl("agent-grid");
    var count = document.getElementById("agent-count");

    if (!agents.length) {
      grid.appendChild(el("div", { cls: "data-empty", text: "No agent data — hive may be offline" }));
      count.textContent = "";
      return;
    }

    count.textContent = agents.length + " agent" + (agents.length !== 1 ? "s" : "");

    var sorted = agents.slice().sort(function (a, b) {
      var ae = a.errors || 0, be = b.errors || 0;
      if (ae !== be) return be - ae;
      return (a.role || "").localeCompare(b.role || "");
    });

    sorted.forEach(function (a) {
      grid.appendChild(buildAgentCard(a));
    });
  }

  function buildAgentCard(a) {
    var state      = (a.state || "unknown").toLowerCase();
    var model      = shortModel(a.model || "");
    var iter       = a.iteration || 0;
    var maxIter    = a.max_iterations || 1;
    var pct        = Math.min(100, Math.round((iter / maxIter) * 100));
    var fillCls    = pct >= 90 ? "danger" : pct >= 70 ? "warn" : "";
    var cost       = fmtCost(a.cost_usd);
    var trust      = a.trust_score != null ? Math.round(a.trust_score * 100) + "%" : "—";
    var errors     = a.errors || 0;
    var hasErrors  = errors > 0;
    var lastEvType = a.last_event_type || "";
    var lastEvAt   = a.last_event_at ? relTime(a.last_event_at) : "";

    var card = el("div", { cls: "agent-card" + (hasErrors ? " has-errors" : "") });
    card.addEventListener("click", function () { card.classList.toggle("expanded"); });

    // Head
    var head = el("div", { cls: "agent-card-head" });
    head.appendChild(el("span", { cls: "agent-role", text: a.role || "unknown" }));
    head.appendChild(el("span", { cls: "badge badge-" + state, text: a.state || "Unknown" }));
    if (model) {
      var mb = el("span", { cls: "badge badge-model", text: model });
      head.appendChild(mb);
    }
    card.appendChild(head);

    // Body
    var body = el("div", { cls: "agent-card-body" });

    // Iter row with progress bar
    var iterRow = el("div", { cls: "agent-row" });
    iterRow.appendChild(el("span", { cls: "agent-label", text: "Iter" }));
    var bar  = el("div", { cls: "progress-bar" });
    var fill = el("div", { cls: "progress-fill " + fillCls });
    fill.style.width = pct + "%";
    bar.appendChild(fill);
    iterRow.appendChild(bar);
    iterRow.appendChild(el("span", { cls: "agent-val", text: iter + "/" + maxIter }));
    body.appendChild(iterRow);

    // Cost / Trust row
    var costRow = el("div", { cls: "agent-row" });
    costRow.appendChild(el("span", { cls: "agent-label", text: "Cost" }));
    costRow.appendChild(el("span", { cls: "agent-val", text: cost }));
    var spacer = el("span"); spacer.style.flex = "1";
    costRow.appendChild(spacer);
    costRow.appendChild(el("span", { cls: "agent-label", text: "Trust" }));
    costRow.appendChild(el("span", { cls: "agent-val", text: trust }));
    body.appendChild(costRow);

    // Last event row
    if (lastEvType || lastEvAt) {
      var evRow = el("div", { cls: "agent-event" });
      if (lastEvType) {
        var pill = el("span", { cls: "event-type-pill " + evtClass(lastEvType), text: lastEvType });
        evRow.appendChild(pill);
      }
      if (lastEvAt) {
        var dim = el("span", { text: lastEvAt });
        dim.style.color = "var(--text-dim)";
        evRow.appendChild(dim);
      }
      if (hasErrors) {
        var errSpan = el("span", { text: errors + " err" });
        errSpan.style.cssText = "color:var(--red);margin-left:auto";
        evRow.appendChild(errSpan);
      }
      body.appendChild(evRow);
    }

    card.appendChild(body);

    // Expand section
    var expand = el("div", { cls: "agent-expand" });

    var msgLabel = el("div", { cls: "expand-label", text: "Last Message" });
    var msgPre   = el("pre", { cls: "last-message", text: a.last_message || "(no message recorded)" });
    expand.appendChild(msgLabel);
    expand.appendChild(msgPre);

    var meta = el("div", { cls: "expand-meta" });

    var tokItem = el("div", { cls: "expand-meta-item" });
    tokItem.appendChild(document.createTextNode("Tokens "));
    var tokStrong = el("strong", { text: (a.tokens_used || 0).toLocaleString() });
    tokItem.appendChild(tokStrong);
    meta.appendChild(tokItem);

    var errItem = el("div", { cls: "expand-meta-item" });
    errItem.appendChild(document.createTextNode("Errors "));
    var errStrong = el("strong", { text: errors });
    errStrong.style.color = hasErrors ? "var(--red)" : "var(--text)";
    errItem.appendChild(errStrong);
    meta.appendChild(errItem);

    if (a.actor_id) {
      var idItem = el("div", { cls: "expand-meta-item",
        text: a.actor_id.slice(0, 20) + (a.actor_id.length > 20 ? "…" : "") });
      idItem.style.cssText = "font-family:var(--mono);font-size:10px;color:var(--text-dim)";
      meta.appendChild(idItem);
    }

    expand.appendChild(meta);
    card.appendChild(expand);
    return card;
  }

  // ── HIVE HEALTH ────────────────────────────────────
  function renderHive(hive) {
    var grid = clearEl("hive-grid");

    if (!hive) {
      grid.appendChild(el("div", { cls: "data-empty", text: "Awaiting telemetry data…" }));
      return;
    }

    // Agents card
    grid.appendChild(buildHiveCard("Agents", function (card) {
      var valEl = el("div", { cls: "hive-card-val", text: hive.active_agents != null ? hive.active_agents : "—" });
      var sub   = el("span", { cls: "hive-card-val-sub", text: " / " + (hive.total_actors != null ? hive.total_actors : "—") });
      valEl.appendChild(sub);
      card.appendChild(valEl);
      card.appendChild(el("div", { cls: "hive-card-sub", text: "active / total" }));
    }));

    // Chain card
    grid.appendChild(buildHiveCard("Chain", function (card) {
      var chainOk = !!hive.chain_ok;
      var valEl   = el("div", { cls: "hive-card-val",
        text: (hive.chain_length || 0).toLocaleString() + " " });
      var icon    = el("span", { text: chainOk ? "✓" : "✗" });
      icon.style.cssText = "font-size:16px;font-weight:700;color:" + (chainOk ? "var(--green)" : "var(--red)");
      valEl.appendChild(icon);
      card.appendChild(valEl);
      card.appendChild(el("div", { cls: "hive-card-sub", text: chainOk ? "integrity ok" : "integrity failed" }));
    }));

    // Event rate card
    grid.appendChild(buildHiveCard("Event Rate", function (card) {
      var rate = hive.event_rate != null ? hive.event_rate + "/min" : "—";
      card.appendChild(el("div", { cls: "hive-card-val", text: rate }));
      card.appendChild(el("div", { cls: "hive-card-sub", text: "events / min" }));
    }));

    // Daily cost card
    grid.appendChild(buildHiveCard("Daily Cost", function (card) {
      if (hive.daily_cost != null) {
        card.appendChild(el("div", { cls: "hive-card-val", text: fmtCost(hive.daily_cost) }));
        if (hive.daily_cap != null && hive.daily_cap > 0) {
          var pct     = Math.min(100, Math.round((hive.daily_cost / hive.daily_cap) * 100));
          var fillCls = pct >= 90 ? "danger" : pct >= 70 ? "warn" : "";
          var barWrap = el("div", { cls: "progress-bar" });
          barWrap.style.cssText = "margin:0.375rem 0 0.25rem";
          var fill = el("div", { cls: "progress-fill " + fillCls });
          fill.style.width = pct + "%";
          barWrap.appendChild(fill);
          card.appendChild(barWrap);
          card.appendChild(el("div", { cls: "hive-card-sub",
            text: pct + "% of " + fmtCost(hive.daily_cap) + " cap" }));
        } else {
          card.appendChild(el("div", { cls: "hive-card-sub", text: "no cap set" }));
        }
      } else {
        card.appendChild(el("div", { cls: "hive-card-val", text: "—" }));
        card.appendChild(el("div", { cls: "hive-card-sub", text: "no data" }));
      }
    }));

    // Severity card
    grid.appendChild(buildHiveCard("Severity", function (card) {
      var sev     = (hive.severity || "ok").toLowerCase();
      var sevCls  = sev === "ok" ? "severity-ok" : sev === "warning" ? "severity-warning" : "severity-critical";
      var badge   = el("span", { cls: "severity-badge " + sevCls, text: sev });
      card.appendChild(badge);
    }));
  }

  function buildHiveCard(label, fillFn) {
    var card = el("div", { cls: "hive-card" });
    card.appendChild(el("div", { cls: "hive-card-label", text: label }));
    fillFn(card);
    return card;
  }

  // ── EVENT STREAM ───────────────────────────────────
  function onStreamScroll() {
    var el2 = document.getElementById("event-stream");
    isUserScrolled = el2.scrollTop > 40;
  }
  // expose for onscroll attribute
  window.onStreamScroll = onStreamScroll;

  function renderEvents(events) {
    var container = clearEl("event-stream");
    var count     = document.getElementById("event-count");

    if (!events.length) {
      container.appendChild(el("div", { cls: "data-empty", text: "No recent events" }));
      count.textContent = "";
      return;
    }

    count.textContent = events.length + " event" + (events.length !== 1 ? "s" : "");

    events.forEach(function (ev) {
      var row = el("div", { cls: "event-row" });

      row.appendChild(el("span", { cls: "event-ts",    text: ev.at ? relTime(ev.at) : "—" }));
      row.appendChild(el("span", { cls: "event-actor", text: ev.actor_role || "—" }));

      var evType = ev.event_type || "";
      var typeBadge = el("span", { cls: "event-type " + evtClass(evType), text: evType || "—" });
      row.appendChild(typeBadge);

      var summary = el("span", { cls: "event-summary", text: ev.summary || "" });
      if (ev.summary) summary.title = ev.summary;
      row.appendChild(summary);

      container.appendChild(row);
    });

    if (!isUserScrolled) container.scrollTop = 0;
  }

  // ── FORMATTERS ─────────────────────────────────────
  function shortModel(m) {
    if (!m) return "";
    if (m.indexOf("haiku")  !== -1) return "haiku";
    if (m.indexOf("sonnet") !== -1) return "sonnet";
    if (m.indexOf("opus")   !== -1) return "opus";
    return m.split("-").slice(0, 2).join("-");
  }

  function fmtCost(v) {
    if (v == null) return "—";
    return "$" + Number(v).toFixed(3);
  }

  function fmtDate(iso) {
    if (!iso) return "";
    try {
      return new Date(iso).toLocaleDateString(undefined,
        { month: "short", day: "numeric", year: "numeric" });
    } catch (e) { return iso; }
  }

  function relTime(iso) {
    if (!iso) return "";
    try {
      var secs = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
      if (secs <  5)    return "just now";
      if (secs < 60)    return secs + "s ago";
      if (secs < 3600)  return Math.floor(secs / 60)   + "m ago";
      if (secs < 86400) return Math.floor(secs / 3600)  + "h ago";
      return Math.floor(secs / 86400) + "d ago";
    } catch (e) { return iso; }
  }

  function evtClass(type) {
    if (!type)                                return "evt-default";
    if (type.indexOf("health.")        === 0) return "evt-health";
    if (type.indexOf("agent.budget.")  === 0) return "evt-budget";
    if (type.indexOf("work.task.")     === 0) return "evt-work";
    if (type.indexOf("agent.state.")   === 0) return "evt-state";
    if (type.indexOf("hive.")          === 0) return "evt-hive";
    return "evt-default";
  }

}());
</script>
</body>
</html>
`
