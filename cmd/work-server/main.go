// Command work-server is an HTTP REST API server for the Work Graph (Layer 1).
// It exposes task management as signed, auditable events on the shared event graph.
//
// Environment variables:
//
//	WORK_HUMAN                — display name of the human operator (required)
//	WORK_API_KEY              — API key for auth; callers pass Authorization: Bearer <key> (required)
//	WORK_API_TOKEN            — bearer token for workspace-scoped external API; falls back to WORK_API_KEY if unset
//	DATABASE_URL              — Postgres DSN (optional; defaults to in-memory)
//	PORT                      — HTTP port to listen on (optional; defaults to 8080)
//	TELEMETRY_DASHBOARD_PATH  — path to dashboard.html on disk (optional; read at request time for live reload)
//
// Endpoints:
//
//	GET  /                                      read-only dashboard (HTML, no auth required)
//	POST /tasks                                 create a task
//	GET  /tasks                                 list tasks (?open=true, ?priority=high, ?assignee=<actor_id>)
//	GET  /tasks/{id}                            get full task details (title, description, priority, status, assignee, blocked)
//	GET  /tasks/{id}/status                     get task status
//	GET  /tasks/{id}/events                     get audit trail (ordered work.task.* events for this task, including comments)
//	POST /tasks/{id}/assign                     assign task (body: {"assignee":"..."})
//	POST /tasks/{id}/unblock                    mark task blockers resolved (body: {})
//	POST /tasks/{id}/complete                   complete task (body: {"summary":"..."})
//	POST /tasks/{id}/comment                    add a comment (body: {"body":"..."})
//	GET  /tasks/{id}/comments                   list comments for a task
//
// Workspace-scoped routes (authenticated via WORK_API_TOKEN):
//
//	GET  /w/{workspace}                         workspace task dashboard (HTML, no auth required)
//	POST /w/{workspace}/tasks                   create a task in the workspace
//	GET  /w/{workspace}/tasks                   list tasks in the workspace
//	POST /w/{workspace}/tasks/{id}/assign       assign a workspace task
//	POST /w/{workspace}/tasks/{id}/complete     complete a workspace task
//	POST /w/{workspace}/tasks/{id}/comment      add a comment to a workspace task
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/actor/pgactor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/store/pgstore"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"

	"github.com/lovyou-ai/work"
)

// dashboardHTML is the read-only monitoring dashboard served at GET /.
// The placeholder {{API_KEY}} is replaced at serve time with the actual key so
// the browser's fetch() calls can authenticate against GET /tasks.
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Work Graph — Live Dashboard</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: system-ui, -apple-system, sans-serif; background: #0f1117; color: #e2e8f0; min-height: 100vh; padding: 2rem; }
h1 { font-size: 1.5rem; font-weight: 600; color: #f8fafc; margin-bottom: 0.25rem; }
.subtitle { font-size: 0.875rem; color: #64748b; margin-bottom: 1.5rem; }
.meta { display: flex; align-items: center; gap: 1rem; margin-bottom: 1.25rem; font-size: 0.8125rem; color: #64748b; }
.dot { width: 8px; height: 8px; border-radius: 50%; background: #22c55e; animation: pulse 2s ease-in-out infinite; }
@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.4; } }
.error { background: #3b1010; color: #fca5a5; padding: 0.75rem 1rem; border-radius: 6px; margin-bottom: 1rem; font-size: 0.875rem; }
.empty { color: #475569; font-size: 0.875rem; padding: 2rem 0; text-align: center; }
table { width: 100%; border-collapse: collapse; font-size: 0.875rem; }
th { text-align: left; padding: 0.5rem 0.75rem; color: #64748b; font-weight: 500; font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; border-bottom: 1px solid #1e293b; }
td { padding: 0.625rem 0.75rem; border-bottom: 1px solid #1e293b; vertical-align: middle; }
tr:hover td { background: #1e293b; }
.title { font-weight: 500; color: #f1f5f9; max-width: 28rem; }
.desc { font-size: 0.75rem; color: #64748b; margin-top: 0.125rem; }
.badge { display: inline-block; padding: 0.2em 0.55em; border-radius: 4px; font-size: 0.72rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.03em; }
.s-open       { background: #1e3a5f; color: #60a5fa; }
.s-in_progress{ background: #3b2400; color: #fb923c; }
.s-completed  { background: #052e16; color: #4ade80; }
.s-blocked    { background: #3b0a0a; color: #f87171; }
.p-high   { background: #3b0a0a; color: #f87171; }
.p-medium { background: #3b2400; color: #fb923c; }
.p-low    { background: #1e293b; color: #94a3b8; }
.blocked-yes { color: #f87171; font-weight: 600; }
.blocked-no  { color: #334155; }
.assignee { font-family: monospace; font-size: 0.75rem; color: #7c3aed; max-width: 12rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.btn { display: inline-block; padding: 0.25em 0.6em; border-radius: 4px; font-size: 0.72rem; font-weight: 600; cursor: pointer; border: none; letter-spacing: 0.02em; transition: opacity 0.15s; }
.btn:hover { opacity: 0.8; }
.btn-assign { background: #1e3a5f; color: #60a5fa; }
.btn-unblock { background: #3b0a0a; color: #f87171; }
.btn-loading { opacity: 0.5; cursor: not-allowed; }
</style>
</head>
<body>
<h1>Work Graph</h1>
<p class="subtitle">Live pipeline dashboard</p>
<div class="meta">
  <span class="dot"></span>
  <span id="status-line">Connecting...</span>
  <span id="countdown"></span>
</div>
<div id="error-box" class="error" style="display:none"></div>
<table id="task-table" style="display:none">
  <thead>
    <tr>
      <th>Task</th>
      <th>Status</th>
      <th>Priority</th>
      <th>Assignee</th>
      <th>Blocked</th>
      <th>Actions</th>
    </tr>
  </thead>
  <tbody id="task-body"></tbody>
</table>
<div id="empty-msg" class="empty" style="display:none">No tasks yet.</div>
<script>
const API_KEY = "{{API_KEY}}";
const REFRESH_MS = 10000;
let timer, countdown, nextAt;

function esc(s) {
  return String(s).replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;");
}

function badge(cls, text) {
  return '<span class="badge ' + cls + '">' + esc(text) + '</span>';
}

function statusBadge(s) {
  const map = { open: "s-open", in_progress: "s-in_progress", completed: "s-completed", blocked: "s-blocked" };
  return badge(map[s] || "s-open", s || "open");
}

function priorityBadge(p) {
  const map = { high: "p-high", medium: "p-medium", low: "p-low" };
  return badge(map[p] || "p-low", p || "");
}

function shortID(id) {
  return id ? id.slice(0, 8) + "\u2026" : "\u2014";
}

async function apiPost(path) {
  const res = await fetch(path, {
    method: "POST",
    headers: { Authorization: "Bearer " + API_KEY, "Content-Type": "application/json" },
    body: "{}",
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error((data.error || "HTTP " + res.status));
  }
  return res.json();
}

async function assignSelf(taskId) {
  try {
    await apiPost("/tasks/" + taskId + "/assign");
    refresh();
  } catch (err) {
    alert("Assign failed: " + err.message);
  }
}

async function unblockTask(taskId) {
  try {
    await apiPost("/tasks/" + taskId + "/unblock");
    refresh();
  } catch (err) {
    alert("Unblock failed: " + err.message);
  }
}

async function refresh() {
  try {
    const res = await fetch("/tasks", { headers: { Authorization: "Bearer " + API_KEY } });
    if (!res.ok) throw new Error("HTTP " + res.status);
    const data = await res.json();
    const tasks = data.tasks || [];
    document.getElementById("error-box").style.display = "none";

    const tbody = document.getElementById("task-body");
    if (tasks.length === 0) {
      document.getElementById("task-table").style.display = "none";
      document.getElementById("empty-msg").style.display = "block";
    } else {
      document.getElementById("task-table").style.display = "table";
      document.getElementById("empty-msg").style.display = "none";
      tbody.innerHTML = tasks.map(t => {
        const blockedCell = t.blocked
          ? '<span class="blocked-yes">\u26a0 blocked</span>'
          : '<span class="blocked-no">\u2014</span>';
        const assigneeCell = t.assignee
          ? '<span class="assignee" title="' + esc(t.assignee) + '">' + esc(shortID(t.assignee)) + '</span>'
          : '<span class="blocked-no">\u2014</span>';
        const assignLabel = t.assignee ? 'Reassign' : 'Assign';
        let actions = '<button class="btn btn-assign" onclick="assignSelf(\'' + esc(t.id) + '\')">' + assignLabel + '</button>';
        if (t.blocked) {
          actions += ' <button class="btn btn-unblock" onclick="unblockTask(\'' + esc(t.id) + '\')">Unblock</button>';
        }
        return '<tr>'
          + '<td><div class="title">' + esc(t.title) + '</div>'
          + (t.description ? '<div class="desc">' + esc(t.description.slice(0, 80)) + (t.description.length > 80 ? "\u2026" : "") + '</div>' : '')
          + '</td>'
          + '<td>' + statusBadge(t.status) + '</td>'
          + '<td>' + priorityBadge(t.priority) + '</td>'
          + '<td>' + assigneeCell + '</td>'
          + '<td>' + blockedCell + '</td>'
          + '<td>' + actions + '</td>'
          + '</tr>';
      }).join("");
    }

    const now = new Date();
    document.getElementById("status-line").textContent =
      "Updated " + now.toLocaleTimeString() + " \u2014 " + tasks.length + " task" + (tasks.length === 1 ? "" : "s");
    nextAt = Date.now() + REFRESH_MS;
  } catch (err) {
    const box = document.getElementById("error-box");
    box.textContent = "Fetch failed: " + err.message;
    box.style.display = "block";
    document.getElementById("status-line").textContent = "Error \u2014 retrying in 10s";
    nextAt = Date.now() + REFRESH_MS;
  }
}

function tick() {
  const secs = Math.max(0, Math.round((nextAt - Date.now()) / 1000));
  document.getElementById("countdown").textContent = secs > 0 ? "(next refresh in " + secs + "s)" : "";
}

refresh();
setInterval(refresh, REFRESH_MS);
setInterval(tick, 1000);
</script>
</body>
</html>`

// workspaceDashboardHTML is the interactive task dashboard served at GET /w/{workspace}.
// Placeholders {{WORKSPACE}} and {{API_TOKEN}} are replaced at serve time.
const workspaceDashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{WORKSPACE}} — Work Graph</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: system-ui, -apple-system, sans-serif; background: #0f1117; color: #e2e8f0; min-height: 100vh; padding: 2rem; }
h1 { font-size: 1.5rem; font-weight: 600; color: #f8fafc; margin-bottom: 0.25rem; }
.subtitle { font-size: 0.875rem; color: #64748b; margin-bottom: 1.5rem; }
.meta { display: flex; align-items: center; gap: 1rem; margin-bottom: 1.25rem; font-size: 0.8125rem; color: #64748b; }
.dot { width: 8px; height: 8px; border-radius: 50%; background: #22c55e; animation: pulse 2s ease-in-out infinite; }
@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.4; } }
.toolbar { display: flex; align-items: center; justify-content: flex-end; margin-bottom: 1.25rem; }
.error { background: #3b1010; color: #fca5a5; padding: 0.75rem 1rem; border-radius: 6px; margin-bottom: 1rem; font-size: 0.875rem; }
.empty { color: #475569; font-size: 0.875rem; padding: 2rem 0; text-align: center; }
table { width: 100%; border-collapse: collapse; font-size: 0.875rem; }
th { text-align: left; padding: 0.5rem 0.75rem; color: #64748b; font-weight: 500; font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; border-bottom: 1px solid #1e293b; }
td { padding: 0.625rem 0.75rem; border-bottom: 1px solid #1e293b; vertical-align: middle; }
tr:hover td { background: #1e293b; }
.title { font-weight: 500; color: #f1f5f9; max-width: 28rem; }
.desc { font-size: 0.75rem; color: #64748b; margin-top: 0.125rem; }
.badge { display: inline-block; padding: 0.2em 0.55em; border-radius: 4px; font-size: 0.72rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.03em; }
.s-open       { background: #1e3a5f; color: #60a5fa; }
.s-in_progress{ background: #3b2400; color: #fb923c; }
.s-completed  { background: #052e16; color: #4ade80; }
.s-blocked    { background: #3b0a0a; color: #f87171; }
.p-high   { background: #3b0a0a; color: #f87171; }
.p-medium { background: #3b2400; color: #fb923c; }
.p-low    { background: #1e293b; color: #94a3b8; }
.blocked-yes { color: #f87171; font-weight: 600; }
.blocked-no  { color: #334155; }
.assignee { font-family: monospace; font-size: 0.75rem; color: #7c3aed; max-width: 12rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.btn { display: inline-block; padding: 0.25em 0.6em; border-radius: 4px; font-size: 0.72rem; font-weight: 600; cursor: pointer; border: none; letter-spacing: 0.02em; transition: opacity 0.15s; margin-right: 0.25rem; }
.btn:hover { opacity: 0.8; }
.btn-primary  { background: #1d4ed8; color: #fff; padding: 0.5em 1em; font-size: 0.8125rem; }
.btn-assign   { background: #1e3a5f; color: #60a5fa; }
.btn-complete { background: #052e16; color: #4ade80; }
.btn-comment  { background: #1e293b; color: #94a3b8; }
.modal-overlay { display: none; position: fixed; inset: 0; background: rgba(0,0,0,0.7); z-index: 100; align-items: center; justify-content: center; }
.modal-overlay.open { display: flex; }
.modal { background: #1e293b; border-radius: 8px; padding: 1.5rem; min-width: 22rem; max-width: 32rem; width: 100%; }
.modal h2 { font-size: 1rem; font-weight: 600; color: #f1f5f9; margin-bottom: 1rem; }
.form-field { margin-bottom: 0.875rem; }
.form-field label { display: block; font-size: 0.8125rem; color: #94a3b8; margin-bottom: 0.25rem; }
.form-field input, .form-field textarea, .form-field select { width: 100%; background: #0f1117; border: 1px solid #334155; border-radius: 4px; color: #e2e8f0; padding: 0.5rem 0.625rem; font-size: 0.875rem; font-family: inherit; }
.form-field textarea { resize: vertical; min-height: 5rem; }
.form-field input:focus, .form-field textarea:focus, .form-field select:focus { outline: none; border-color: #3b82f6; }
.modal-actions { display: flex; gap: 0.5rem; justify-content: flex-end; margin-top: 1rem; }
.btn-cancel { background: #0f1117; color: #94a3b8; border: 1px solid #334155; }
.btn-submit { background: #1d4ed8; color: #fff; }
</style>
</head>
<body>
<h1>{{WORKSPACE}}</h1>
<p class="subtitle">Workspace task board</p>
<div class="meta">
  <span class="dot"></span>
  <span id="status-line">Connecting...</span>
  <span id="countdown"></span>
</div>
<div id="error-box" class="error" style="display:none"></div>
<div class="toolbar">
  <button class="btn btn-primary" onclick="openCreate()">+ New Task</button>
</div>
<table id="task-table" style="display:none">
  <thead>
    <tr>
      <th>Task</th>
      <th>Status</th>
      <th>Priority</th>
      <th>Assignee</th>
      <th>Blocked</th>
      <th>Actions</th>
    </tr>
  </thead>
  <tbody id="task-body"></tbody>
</table>
<div id="empty-msg" class="empty" style="display:none">No tasks yet. Create one above.</div>

<!-- Create Task Modal -->
<div class="modal-overlay" id="create-modal" onclick="if(event.target===this)closeCreate()">
  <div class="modal">
    <h2>New Task</h2>
    <div class="form-field">
      <label>Title *</label>
      <input type="text" id="create-title" placeholder="Task title" />
    </div>
    <div class="form-field">
      <label>Description</label>
      <textarea id="create-desc" placeholder="Optional description"></textarea>
    </div>
    <div class="form-field">
      <label>Priority</label>
      <select id="create-priority">
        <option value="medium">Medium</option>
        <option value="high">High</option>
        <option value="low">Low</option>
      </select>
    </div>
    <div class="modal-actions">
      <button class="btn btn-cancel" onclick="closeCreate()">Cancel</button>
      <button class="btn btn-submit" onclick="submitCreate()">Create</button>
    </div>
  </div>
</div>

<!-- Comment Modal -->
<div class="modal-overlay" id="comment-modal" onclick="if(event.target===this)closeComment()">
  <div class="modal">
    <h2>Add Comment</h2>
    <div class="form-field">
      <label>Comment *</label>
      <textarea id="comment-body" placeholder="Enter your comment"></textarea>
    </div>
    <div class="modal-actions">
      <button class="btn btn-cancel" onclick="closeComment()">Cancel</button>
      <button class="btn btn-submit" onclick="submitComment()">Post</button>
    </div>
  </div>
</div>

<script>
const WORKSPACE = "{{WORKSPACE}}";
const API_TOKEN = "{{API_TOKEN}}";
const BASE = "/w/" + WORKSPACE;
const REFRESH_MS = 10000;
let nextAt;
let pendingCommentTaskId = null;

function esc(s) {
  return String(s).replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;");
}

function badge(cls, text) {
  return '<span class="badge ' + cls + '">' + esc(text) + '</span>';
}

function statusBadge(s) {
  const map = { open: "s-open", in_progress: "s-in_progress", completed: "s-completed", blocked: "s-blocked" };
  return badge(map[s] || "s-open", s || "open");
}

function priorityBadge(p) {
  const map = { high: "p-high", medium: "p-medium", low: "p-low" };
  return badge(map[p] || "p-low", p || "");
}

function shortID(id) {
  return id ? id.slice(0, 8) + "\u2026" : "\u2014";
}

function authHeaders() {
  return { Authorization: "Bearer " + API_TOKEN, "Content-Type": "application/json" };
}

async function apiPost(path, body) {
  const res = await fetch(path, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify(body || {}),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || "HTTP " + res.status);
  return data;
}

function openCreate() {
  document.getElementById("create-title").value = "";
  document.getElementById("create-desc").value = "";
  document.getElementById("create-priority").value = "medium";
  document.getElementById("create-modal").classList.add("open");
  document.getElementById("create-title").focus();
}

function closeCreate() {
  document.getElementById("create-modal").classList.remove("open");
}

async function submitCreate() {
  const title = document.getElementById("create-title").value.trim();
  if (!title) { alert("Title is required"); return; }
  try {
    await apiPost(BASE + "/tasks", {
      title,
      description: document.getElementById("create-desc").value.trim(),
      priority: document.getElementById("create-priority").value,
    });
    closeCreate();
    refresh();
  } catch (err) {
    alert("Create failed: " + err.message);
  }
}

async function assignTask(taskId) {
  try {
    await apiPost(BASE + "/tasks/" + taskId + "/assign", {});
    refresh();
  } catch (err) {
    alert("Assign failed: " + err.message);
  }
}

async function completeTask(taskId) {
  if (!confirm("Mark task as complete?")) return;
  try {
    await apiPost(BASE + "/tasks/" + taskId + "/complete", { summary: "Completed via dashboard" });
    refresh();
  } catch (err) {
    alert("Complete failed: " + err.message);
  }
}

function openComment(taskId) {
  pendingCommentTaskId = taskId;
  document.getElementById("comment-body").value = "";
  document.getElementById("comment-modal").classList.add("open");
  document.getElementById("comment-body").focus();
}

function closeComment() {
  document.getElementById("comment-modal").classList.remove("open");
  pendingCommentTaskId = null;
}

async function submitComment() {
  const body = document.getElementById("comment-body").value.trim();
  if (!body) { alert("Comment is required"); return; }
  try {
    await apiPost(BASE + "/tasks/" + pendingCommentTaskId + "/comment", { body });
    closeComment();
    refresh();
  } catch (err) {
    alert("Comment failed: " + err.message);
  }
}

async function refresh() {
  try {
    const res = await fetch(BASE + "/tasks", { headers: { Authorization: "Bearer " + API_TOKEN } });
    if (!res.ok) throw new Error("HTTP " + res.status);
    const data = await res.json();
    const tasks = data.tasks || [];
    document.getElementById("error-box").style.display = "none";

    const tbody = document.getElementById("task-body");
    if (tasks.length === 0) {
      document.getElementById("task-table").style.display = "none";
      document.getElementById("empty-msg").style.display = "block";
    } else {
      document.getElementById("task-table").style.display = "table";
      document.getElementById("empty-msg").style.display = "none";
      tbody.innerHTML = tasks.map(t => {
        const blockedCell = t.blocked
          ? '<span class="blocked-yes">\u26a0 blocked</span>'
          : '<span class="blocked-no">\u2014</span>';
        const assigneeCell = t.assignee
          ? '<span class="assignee" title="' + esc(t.assignee) + '">' + esc(shortID(t.assignee)) + '</span>'
          : '<span class="blocked-no">\u2014</span>';
        const isCompleted = t.status === "completed";
        let actions = "";
        if (!isCompleted) {
          actions += '<button class="btn btn-assign" onclick="assignTask(\'' + esc(t.id) + '\')">' + (t.assignee ? 'Reassign' : 'Assign') + '</button>';
          actions += '<button class="btn btn-complete" onclick="completeTask(\'' + esc(t.id) + '\')">Complete</button>';
        }
        actions += '<button class="btn btn-comment" onclick="openComment(\'' + esc(t.id) + '\')">Comment</button>';
        return '<tr>'
          + '<td><div class="title">' + esc(t.title) + '</div>'
          + (t.description ? '<div class="desc">' + esc(t.description.slice(0, 80)) + (t.description.length > 80 ? "\u2026" : "") + '</div>' : '')
          + '</td>'
          + '<td>' + statusBadge(t.status) + '</td>'
          + '<td>' + priorityBadge(t.priority) + '</td>'
          + '<td>' + assigneeCell + '</td>'
          + '<td>' + blockedCell + '</td>'
          + '<td>' + actions + '</td>'
          + '</tr>';
      }).join("");
    }

    const now = new Date();
    document.getElementById("status-line").textContent =
      "Updated " + now.toLocaleTimeString() + " \u2014 " + tasks.length + " task" + (tasks.length === 1 ? "" : "s");
    nextAt = Date.now() + REFRESH_MS;
  } catch (err) {
    const box = document.getElementById("error-box");
    box.textContent = "Fetch failed: " + err.message;
    box.style.display = "block";
    document.getElementById("status-line").textContent = "Error \u2014 retrying in 10s";
    nextAt = Date.now() + REFRESH_MS;
  }
}

function tick() {
  const secs = Math.max(0, Math.round((nextAt - Date.now()) / 1000));
  document.getElementById("countdown").textContent = secs > 0 ? "(next refresh in " + secs + "s)" : "";
}

document.addEventListener("keydown", e => {
  if (e.key === "Escape") { closeCreate(); closeComment(); }
});

refresh();
setInterval(refresh, REFRESH_MS);
setInterval(tick, 1000);
</script>
</body>
</html>`

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	humanName := os.Getenv("WORK_HUMAN")
	if humanName == "" {
		return fmt.Errorf("WORK_HUMAN env var is required (display name of the human operator)")
	}
	apiKey := os.Getenv("WORK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("WORK_API_KEY env var is required")
	}
	apiToken := os.Getenv("WORK_API_TOKEN")
	if apiToken == "" {
		apiToken = apiKey
	}
	dsn := os.Getenv("DATABASE_URL")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Open shared pool for Postgres, or nil for in-memory.
	var pool *pgxpool.Pool
	if dsn != "" {
		fmt.Fprintf(os.Stderr, "Postgres: %s\n", dsn)
		poolCfg, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			return fmt.Errorf("postgres config: %w", err)
		}
		// The pool is shared between the event store (which holds connections
		// during advisory-locked writes) and telemetry read queries. Default
		// MaxConns (≈4 in containers) is too small: just a few concurrent
		// event writes exhaust the pool and starve telemetry reads.
		poolCfg.MaxConns = 20
		poolCfg.MinConns = 2
		poolCfg.MaxConnLifetime = 30 * time.Minute
		poolCfg.MaxConnIdleTime = 5 * time.Minute
		poolCfg.HealthCheckPeriod = 30 * time.Second
		poolCfg.ConnConfig.ConnectTimeout = 5 * time.Second
		pool, err = pgxpool.NewWithConfig(ctx, poolCfg)
		if err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
		defer pool.Close()
	}

	s, err := openStore(ctx, pool)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "store close: %v\n", err)
		}
	}()

	actors, err := openActorStore(ctx, pool)
	if err != nil {
		return fmt.Errorf("actor store: %w", err)
	}

	// Bootstrap human actor — same key-derivation pattern as cmd/hive.
	if pool != nil {
		fmt.Fprintln(os.Stderr, "WARNING: CLI key derivation is insecure for persistent Postgres stores.")
		fmt.Fprintln(os.Stderr, "         Production should use Google auth. Proceeding for development.")
	}
	humanID, err := registerHuman(actors, humanName)
	if err != nil {
		return fmt.Errorf("register human: %w", err)
	}

	// Register work event type unmarshalers before any store reads —
	// Head() deserializes the latest event which may be a work type.
	// Enable raw fallback so unknown event types (hive, agent, membrane)
	// don't break deserialization when sharing a Postgres store.
	event.SetFallbackUnmarshaler(event.RawFallback)
	work.RegisterEventTypes()

	// Bootstrap the event graph if it has no genesis event.
	if err := bootstrapGraph(s, humanID); err != nil {
		return fmt.Errorf("bootstrap graph: %w", err)
	}

	// Build factory and signer for work events.
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	factory := event.NewEventFactory(registry)
	signer := deriveSignerFromID(humanID)

	ts := work.NewTaskStore(s, factory, signer)

	srv := &server{
		ts:       ts,
		store:    s,
		humanID:  humanID,
		apiKey:   apiKey,
		apiToken: apiToken,
		pool:     pool,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", srv.dashboard)
	mux.HandleFunc("GET /health", srv.health)
	mux.HandleFunc("POST /tasks", srv.auth(srv.createTask))
	mux.HandleFunc("GET /tasks", srv.auth(srv.listTasks))
	mux.HandleFunc("GET /tasks/{id}", srv.auth(srv.getTask))
	mux.HandleFunc("GET /tasks/{id}/status", srv.auth(srv.getTaskStatus))
	mux.HandleFunc("GET /tasks/{id}/events", srv.auth(srv.getTaskEvents))
	mux.HandleFunc("POST /tasks/{id}/assign", srv.auth(srv.assignTask))
	mux.HandleFunc("POST /tasks/{id}/unblock", srv.auth(srv.unblockTask))
	mux.HandleFunc("POST /tasks/{id}/complete", srv.auth(srv.completeTask))
	mux.HandleFunc("POST /tasks/{id}/comment", srv.auth(srv.addComment))
	mux.HandleFunc("GET /tasks/{id}/comments", srv.auth(srv.listComments))

	// Telemetry routes — reads from hive-postgres via the shared pool.
	mux.HandleFunc("GET /telemetry/status", srv.auth(srv.telemetryStatus))
	mux.HandleFunc("GET /telemetry/agents", srv.auth(srv.telemetryAgents))
	mux.HandleFunc("GET /telemetry/agents/{role}", srv.auth(srv.telemetryAgentDetail))
	mux.HandleFunc("GET /telemetry/stream", srv.auth(srv.telemetryStream))
	mux.HandleFunc("GET /telemetry/phases", srv.auth(srv.telemetryPhases))
	mux.HandleFunc("POST /telemetry/phases/{phase}", srv.auth(srv.updatePhase))
	mux.HandleFunc("GET /telemetry/health", srv.auth(srv.telemetryHealth))
	mux.HandleFunc("GET /telemetry/sse", srv.auth(srv.telemetrySSE))
	mux.HandleFunc("GET /telemetry/", func(w http.ResponseWriter, r *http.Request) {
		// Set a session cookie so the dashboard can poll without an Authorization
		// header, avoiding Chrome Private-Network-Access preflight blocks.
		http.SetCookie(w, &http.Cookie{
			Name:     "ws_key",
			Value:    srv.apiKey,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if path := os.Getenv("TELEMETRY_DASHBOARD_PATH"); path != "" {
			if html, err := os.ReadFile(path); err == nil {
				w.Write(html)
				return
			}
		}
		fmt.Fprint(w, telemetryDashboardHTML)
	})

	// Workspace-scoped routes — isolated namespace per team, auth via WORK_API_TOKEN.
	mux.HandleFunc("GET /w/{workspace}", srv.workspaceDashboard)
	mux.HandleFunc("POST /w/{workspace}/tasks", srv.tokenAuth(srv.createWorkspaceTask))
	mux.HandleFunc("GET /w/{workspace}/tasks", srv.tokenAuth(srv.listWorkspaceTasks))
	mux.HandleFunc("POST /w/{workspace}/tasks/{id}/assign", srv.tokenAuth(srv.assignTask))
	mux.HandleFunc("POST /w/{workspace}/tasks/{id}/complete", srv.tokenAuth(srv.completeTask))
	mux.HandleFunc("POST /w/{workspace}/tasks/{id}/comment", srv.tokenAuth(srv.addComment))

	addr := ":" + port
	fmt.Fprintf(os.Stderr, "work-server listening on %s\n", addr)
	httpSrv := &http.Server{Addr: addr, Handler: corsMiddleware(mux)}
	go func() {
		<-ctx.Done()
		httpSrv.Shutdown(context.Background()) //nolint:errcheck
	}()
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

// server holds shared dependencies for HTTP handlers.
type server struct {
	ts       *work.TaskStore
	store    store.Store
	humanID  types.ActorID
	apiKey   string
	apiToken string
	pool     *pgxpool.Pool // nil when running in-memory; telemetry handlers check this
}

// dashboard handles GET / — serves the read-only HTML monitoring dashboard.
// No auth required; the API key is injected into the page so the browser's
// fetch() calls can authenticate against GET /tasks.
func (sv *server) dashboard(w http.ResponseWriter, r *http.Request) {
	html := strings.ReplaceAll(dashboardHTML, "{{API_KEY}}", jsEscapeKey(sv.apiKey))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// workspaceDashboard handles GET /w/{workspace} — serves the interactive workspace task dashboard.
// No auth required on the GET; the API token is injected into the page for browser fetch() calls.
func (sv *server) workspaceDashboard(w http.ResponseWriter, r *http.Request) {
	workspace := r.PathValue("workspace")
	if workspace == "" {
		writeErr(w, http.StatusBadRequest, "workspace is required")
		return
	}
	html := strings.ReplaceAll(workspaceDashboardHTML, "{{WORKSPACE}}", jsEscapeKey(workspace))
	html = strings.ReplaceAll(html, "{{API_TOKEN}}", jsEscapeKey(sv.apiToken))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// health handles GET /health — used by Fly.io and load balancers to check liveness.
func (sv *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// jsEscapeKey returns s with characters that are dangerous inside a <script>
// JSON string literal escaped as Unicode escapes. This prevents the HTML parser
// from interpreting characters like < as tag openers before the JS engine runs.
func jsEscapeKey(s string) string {
	b, _ := json.Marshal(s) // produces "\"…\"" with \u003c etc.
	// Strip surrounding quotes — callers already embed in "{{API_KEY}}".
	return string(b[1 : len(b)-1])
}

// corsMiddleware adds CORS and Private Network Access headers for browsers.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// auth is middleware that validates the API key via Bearer header or session cookie.
func (sv *server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try Authorization header first (external API callers).
		if token, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); found && token == sv.apiKey {
			next(w, r)
			return
		}
		// Fall back to session cookie (inline dashboard avoids custom headers
		// that trigger Chrome Private-Network-Access preflight blocks).
		if c, err := r.Cookie("ws_key"); err == nil && c.Value == sv.apiKey {
			next(w, r)
			return
		}
		writeErr(w, http.StatusUnauthorized, "invalid or missing API key")
	}
}

// tokenAuth is middleware for workspace routes that validates the WORK_API_TOKEN bearer token.
func (sv *server) tokenAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !found || token != sv.apiToken {
			writeErr(w, http.StatusUnauthorized, "invalid or missing API token")
			return
		}
		next(w, r)
	}
}

// createTask handles POST /tasks
// Body: {"title":"...", "description":"...", "priority":"high"}
func (sv *server) createTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Title == "" {
		writeErr(w, http.StatusBadRequest, "title is required")
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	task, err := sv.ts.Create(sv.humanID, body.Title, body.Description, causes, convID, work.TaskPriority(body.Priority))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "create task: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          task.ID.Value(),
		"title":       task.Title,
		"description": task.Description,
		"priority":    string(task.Priority),
		"created_by":  task.CreatedBy.Value(),
	})
}

// listTasks handles GET /tasks
// Query params: ?open=true, ?priority=high, ?assignee=<actor_id>
func (sv *server) listTasks(w http.ResponseWriter, r *http.Request) {
	openOnly := r.URL.Query().Get("open") == "true"
	priorityFilter := r.URL.Query().Get("priority")
	assigneeFilter := r.URL.Query().Get("assignee")

	summaries, err := sv.ts.ListSummaries(100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list tasks: "+err.Error())
		return
	}

	if openOnly {
		filtered := make([]work.TaskSummary, 0, len(summaries))
		for _, s := range summaries {
			if s.Status != work.StatusCompleted && !s.Blocked {
				filtered = append(filtered, s)
			}
		}
		summaries = filtered
	}

	if assigneeFilter != "" {
		aid, err := types.NewActorID(assigneeFilter)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid assignee: "+err.Error())
			return
		}
		filtered := make([]work.TaskSummary, 0, len(summaries))
		for _, s := range summaries {
			if s.Assignee == aid {
				filtered = append(filtered, s)
			}
		}
		summaries = filtered
	}

	if priorityFilter != "" {
		p := work.TaskPriority(priorityFilter)
		filtered := make([]work.TaskSummary, 0, len(summaries))
		for _, s := range summaries {
			if s.Task.Priority == p {
				filtered = append(filtered, s)
			}
		}
		summaries = filtered
	}

	items := make([]map[string]any, 0, len(summaries))
	for _, s := range summaries {
		items = append(items, map[string]any{
			"id":          s.Task.ID.Value(),
			"title":       s.Task.Title,
			"description": s.Task.Description,
			"priority":    string(s.Task.Priority),
			"created_by":  s.Task.CreatedBy.Value(),
			"status":      string(s.Status),
			"assignee":    s.Assignee.Value(),
			"blocked":     s.Blocked,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": items})
}

// getTaskStatus handles GET /tasks/{id}/status
func (sv *server) getTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	status, err := sv.ts.GetStatus(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get status: "+err.Error())
		return
	}
	priority, err := sv.ts.GetPriority(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get priority: "+err.Error())
		return
	}
	blocked, err := sv.ts.IsBlocked(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "blocked check: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       taskID.Value(),
		"status":   string(status),
		"priority": string(priority),
		"blocked":  blocked,
	})
}

// assignTask handles POST /tasks/{id}/assign
// Body: {"assignee":"actor_id"} — omit assignee to assign to the human operator.
func (sv *server) assignTask(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	var body struct {
		Assignee string `json:"assignee"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	assignee := sv.humanID
	if body.Assignee != "" {
		aid, err := types.NewActorID(body.Assignee)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid assignee: "+err.Error())
			return
		}
		assignee = aid
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.ts.Assign(sv.humanID, taskID, assignee, causes, convID); err != nil {
		writeErr(w, http.StatusInternalServerError, "assign: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"task_id":  taskID.Value(),
		"assignee": assignee.Value(),
	})
}

// unblockTask handles POST /tasks/{id}/unblock
// Emits a work.task.unblocked event, explicitly marking the task's blockers resolved.
// Body: {} (no fields required; actor is the authenticated human operator)
func (sv *server) unblockTask(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.ts.UnblockTask(sv.humanID, taskID, causes, convID); err != nil {
		writeErr(w, http.StatusInternalServerError, "unblock: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"task_id": taskID.Value(),
		"blocked": false,
	})
}

// completeTask handles POST /tasks/{id}/complete
// Body: {"summary":"..."}
func (sv *server) completeTask(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	var body struct {
		Summary string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.ts.Complete(sv.humanID, taskID, body.Summary, causes, convID); err != nil {
		writeErr(w, http.StatusInternalServerError, "complete: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"task_id": taskID.Value(),
		"status":  "completed",
	})
}

// addComment handles POST /tasks/{id}/comment
// Body: {"body":"..."}
func (sv *server) addComment(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Body == "" {
		writeErr(w, http.StatusBadRequest, "body is required")
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	if err := sv.ts.AddComment(taskID, body.Body, sv.humanID, causes, convID); err != nil {
		writeErr(w, http.StatusInternalServerError, "add comment: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"task_id": taskID.Value(),
		"status":  "commented",
	})
}

// listComments handles GET /tasks/{id}/comments
// Returns all comments for the task in chronological order.
func (sv *server) listComments(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}
	comments, err := sv.ts.ListComments(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list comments: "+err.Error())
		return
	}
	items := make([]map[string]any, 0, len(comments))
	for _, c := range comments {
		items = append(items, map[string]any{
			"id":        c.ID.Value(),
			"task_id":   c.TaskID.Value(),
			"body":      c.Body,
			"author_id": c.AuthorID.Value(),
			"timestamp": c.Timestamp.String(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"task_id": taskID.Value(), "comments": items})
}

// getTask handles GET /tasks/{id}
// Returns full task details: title, description, priority, status, assignee, blocked.
func (sv *server) getTask(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	// Fetch the creation event for base task fields.
	ev, err := sv.store.Get(taskID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "task not found: "+err.Error())
		return
	}
	c, ok := ev.Content().(work.TaskCreatedContent)
	if !ok {
		writeErr(w, http.StatusNotFound, "event is not a task")
		return
	}

	status, err := sv.ts.GetStatus(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get status: "+err.Error())
		return
	}
	priority, err := sv.ts.GetPriority(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get priority: "+err.Error())
		return
	}
	blocked, err := sv.ts.IsBlocked(taskID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "blocked check: "+err.Error())
		return
	}

	// Find current assignee: assigned events are returned newest-first, so the first match wins.
	var assignee string
	assignedPage, err := sv.store.ByType(work.EventTypeTaskAssigned, 1000, types.None[types.Cursor]())
	if err == nil {
		for _, ae := range assignedPage.Items() {
			ac, ok := ae.Content().(work.TaskAssignedContent)
			if ok && ac.TaskID == taskID {
				assignee = ac.AssignedTo.Value()
				break
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":          taskID.Value(),
		"title":       c.Title,
		"description": c.Description,
		"priority":    string(priority),
		"status":      string(status),
		"created_by":  c.CreatedBy.Value(),
		"assignee":    assignee,
		"blocked":     blocked,
	})
}

// getTaskEvents handles GET /tasks/{id}/events
// Returns the ordered audit trail of all work.task.* events causally linked to this task.
func (sv *server) getTaskEvents(w http.ResponseWriter, r *http.Request) {
	taskID, ok := parseTaskID(w, r)
	if !ok {
		return
	}

	var collected []event.Event

	// Include the task creation event itself.
	if ev, err := sv.store.Get(taskID); err == nil {
		if _, ok := ev.Content().(work.TaskCreatedContent); ok {
			collected = append(collected, ev)
		}
	}

	// Scan all other work event types for events that reference this task.
	for _, et := range []types.EventType{
		work.EventTypeTaskAssigned,
		work.EventTypeTaskCompleted,
		work.EventTypeTaskDependencyAdded,
		work.EventTypeTaskPrioritySet,
		work.EventTypeTaskComment,
		work.EventTypeTaskUnblocked,
	} {
		page, err := sv.store.ByType(et, 1000, types.None[types.Cursor]())
		if err != nil {
			continue
		}
		for _, ev := range page.Items() {
			if taskIDFromContent(ev.Content()) == taskID {
				collected = append(collected, ev)
			}
		}
	}

	// Sort chronologically (oldest first) for a readable audit trail.
	sort.Slice(collected, func(i, j int) bool {
		return collected[i].Timestamp().Value().Before(collected[j].Timestamp().Value())
	})

	items := make([]map[string]any, 0, len(collected))
	for _, ev := range collected {
		items = append(items, map[string]any{
			"id":        ev.ID().Value(),
			"type":      ev.Type().Value(),
			"source":    ev.Source().Value(),
			"timestamp": ev.Timestamp().String(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"task_id": taskID.Value(), "events": items})
}

// taskIDFromContent extracts the TaskID field from a work event content struct.
// Returns zero value EventID if the content type does not reference a task ID.
func taskIDFromContent(content any) types.EventID {
	switch c := content.(type) {
	case work.TaskAssignedContent:
		return c.TaskID
	case work.TaskCompletedContent:
		return c.TaskID
	case work.TaskDependencyContent:
		return c.TaskID
	case work.TaskPrioritySetContent:
		return c.TaskID
	case work.CommentContent:
		return c.TaskID
	case work.TaskUnblockedContent:
		return c.TaskID
	}
	return types.EventID{}
}

// createWorkspaceTask handles POST /w/{workspace}/tasks
// Creates a task scoped to the given workspace namespace.
// Body: {"title":"...", "description":"...", "priority":"high"}
func (sv *server) createWorkspaceTask(w http.ResponseWriter, r *http.Request) {
	workspace := r.PathValue("workspace")
	if workspace == "" {
		writeErr(w, http.StatusBadRequest, "workspace is required")
		return
	}
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Title == "" {
		writeErr(w, http.StatusBadRequest, "title is required")
		return
	}
	causes, err := sv.currentCauses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "get causes: "+err.Error())
		return
	}
	convID, err := newConversationID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "conversation id: "+err.Error())
		return
	}
	task, err := sv.ts.CreateInWorkspace(sv.humanID, body.Title, body.Description, workspace, causes, convID, work.TaskPriority(body.Priority))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "create task: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          task.ID.Value(),
		"title":       task.Title,
		"description": task.Description,
		"priority":    string(task.Priority),
		"workspace":   task.Workspace,
		"created_by":  task.CreatedBy.Value(),
	})
}

// listWorkspaceTasks handles GET /w/{workspace}/tasks
// Lists tasks scoped to the given workspace namespace.
func (sv *server) listWorkspaceTasks(w http.ResponseWriter, r *http.Request) {
	workspace := r.PathValue("workspace")
	if workspace == "" {
		writeErr(w, http.StatusBadRequest, "workspace is required")
		return
	}
	summaries, err := sv.ts.ListSummariesByWorkspace(workspace, 100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list tasks: "+err.Error())
		return
	}
	items := make([]map[string]any, 0, len(summaries))
	for _, s := range summaries {
		items = append(items, map[string]any{
			"id":          s.Task.ID.Value(),
			"title":       s.Task.Title,
			"description": s.Task.Description,
			"priority":    string(s.Task.Priority),
			"workspace":   s.Task.Workspace,
			"created_by":  s.Task.CreatedBy.Value(),
			"status":      string(s.Status),
			"assignee":    s.Assignee.Value(),
			"blocked":     s.Blocked,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspace": workspace, "tasks": items})
}

// --- Helpers ---

// currentCauses fetches the current graph head to use as a cause for new events.
func (sv *server) currentCauses() ([]types.EventID, error) {
	head, err := sv.store.Head()
	if err != nil {
		return nil, err
	}
	if head.IsSome() {
		return []types.EventID{head.Unwrap().ID()}, nil
	}
	return nil, nil
}

// parseTaskID extracts and validates the {id} path parameter from the request.
func parseTaskID(w http.ResponseWriter, r *http.Request) (types.EventID, bool) {
	idStr := r.PathValue("id")
	if idStr == "" {
		writeErr(w, http.StatusBadRequest, "task id is required")
		return types.EventID{}, false
	}
	taskID, err := types.NewEventID(idStr)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid task id: "+err.Error())
		return types.EventID{}, false
	}
	return taskID, true
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// writeErr writes a JSON error response.
func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// --- Infrastructure helpers (mirror of cmd/work patterns) ---

func openStore(ctx context.Context, pool *pgxpool.Pool) (store.Store, error) {
	if pool == nil {
		fmt.Fprintln(os.Stderr, "Store: in-memory")
		return store.NewInMemoryStore(), nil
	}
	fmt.Fprintln(os.Stderr, "Store: postgres")
	return pgstore.NewPostgresStoreFromPool(ctx, pool)
}

func openActorStore(ctx context.Context, pool *pgxpool.Pool) (actor.IActorStore, error) {
	if pool == nil {
		fmt.Fprintln(os.Stderr, "Actor store: in-memory")
		return actor.NewInMemoryActorStore(), nil
	}
	fmt.Fprintln(os.Stderr, "Actor store: postgres")
	return pgactor.NewPostgresActorStoreFromPool(ctx, pool)
}

// bootstrapGraph emits the genesis event if the store is empty. Idempotent.
func bootstrapGraph(s store.Store, humanID types.ActorID) error {
	head, err := s.Head()
	if err != nil {
		return fmt.Errorf("check head: %w", err)
	}
	if head.IsSome() {
		return nil // already bootstrapped
	}
	fmt.Fprintln(os.Stderr, "Bootstrapping event graph...")
	registry := event.DefaultRegistry()
	bsFactory := event.NewBootstrapFactory(registry)
	bsSigner := &bootstrapSigner{humanID: humanID}
	bootstrap, err := bsFactory.Init(humanID, bsSigner)
	if err != nil {
		return fmt.Errorf("create genesis event: %w", err)
	}
	if _, err := s.Append(bootstrap); err != nil {
		return fmt.Errorf("append genesis event: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Event graph bootstrapped.")
	return nil
}

// bootstrapSigner provides a minimal Signer for the genesis event.
type bootstrapSigner struct {
	humanID types.ActorID
}

func (b *bootstrapSigner) Sign(data []byte) (types.Signature, error) {
	h := sha256.Sum256([]byte("signer:" + b.humanID.Value()))
	priv := ed25519.NewKeyFromSeed(h[:])
	sig := ed25519.Sign(priv, data)
	return types.NewSignature(sig)
}

// registerHuman bootstraps a human operator in the actor store.
// WARNING: derives key from display name — insecure for production persistent stores.
// Mirrors cmd/hive registerHuman exactly so the same name produces the same ActorID.
func registerHuman(actors actor.IActorStore, displayName string) (types.ActorID, error) {
	h := sha256.Sum256([]byte("human:" + displayName))
	priv := ed25519.NewKeyFromSeed(h[:])
	pub := priv.Public().(ed25519.PublicKey)
	pk, err := types.NewPublicKey([]byte(pub))
	if err != nil {
		return types.ActorID{}, fmt.Errorf("public key: %w", err)
	}
	a, err := actors.Register(pk, displayName, event.ActorTypeHuman)
	if err != nil {
		return types.ActorID{}, err
	}
	return a.ID(), nil
}

// ed25519Signer implements event.Signer for work-emitted events.
type ed25519Signer struct {
	key ed25519.PrivateKey
}

func (s *ed25519Signer) Sign(data []byte) (types.Signature, error) {
	sig := ed25519.Sign(s.key, data)
	return types.NewSignature(sig)
}

// deriveSignerFromID creates a deterministic Ed25519 signer from an ActorID.
// Stable across restarts — the same humanID always produces the same key.
func deriveSignerFromID(id types.ActorID) *ed25519Signer {
	h := sha256.Sum256([]byte("signer:" + id.Value()))
	priv := ed25519.NewKeyFromSeed(h[:])
	return &ed25519Signer{key: priv}
}

// newConversationID generates a unique ConversationID for this HTTP request.
func newConversationID() (types.ConversationID, error) {
	id, err := types.NewEventIDFromNew()
	if err != nil {
		return types.ConversationID{}, err
	}
	return types.NewConversationID("work-server-" + id.Value())
}
