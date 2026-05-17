// SSG AI Manager UI — vanilla JS, no build step.

const state = {
  defaultServer: window.AI_DEFAULT_SERVER || 'http://127.0.0.1:8080',
  agents: new Map(),     // id -> summary
  selectedId: null,
  detail: null,          // last detail payload for the selected agent
  detailTab: 'overview',
};

const el = (id) => document.getElementById(id);
const fmt = (v) => (v == null ? '' : String(v));

async function api(path, opts = {}) {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  });
  if (!res.ok) {
    let msg = res.statusText;
    try { const j = await res.json(); msg = j.error || j.message || msg; } catch (_) {}
    throw new Error(msg);
  }
  return res.json();
}

function setConn(label, cls) {
  const c = el('connection');
  c.textContent = label;
  c.className = 'conn ' + (cls || '');
}

function renderAgentList() {
  const list = el('agent-list');
  list.innerHTML = '';
  const ids = [...state.agents.keys()].sort();
  for (const id of ids) {
    const s = state.agents.get(id);
    const li = document.createElement('li');
    if (id === state.selectedId) li.classList.add('selected');
    const title = document.createElement('div');
    title.className = 'title';
    const pill = document.createElement('span');
    pill.className = 'status-pill ' + s.status;
    pill.textContent = s.status;
    title.append(pill, document.createTextNode(`game ${s.game_id.slice(0, 8)} · P${s.player_id}`));
    const sub = document.createElement('div');
    sub.className = 'sub';
    sub.textContent = `turn ${Math.round(s.current_turn)} · ${s.owned_planets}p / ${s.owned_fleets}f`;
    li.append(title, sub);
    li.addEventListener('click', () => selectAgent(id));
    list.append(li);
  }
}

async function selectAgent(id) {
  state.selectedId = id;
  state.detail = null;
  renderAgentList();
  await refreshDetail();
}

async function refreshDetail() {
  const id = state.selectedId;
  const detailEl = el('detail');
  if (!id) {
    detailEl.innerHTML = '<div class="empty">Select or add an agent.</div>';
    return;
  }
  try {
    state.detail = await api(`/api/agents/${id}`);
  } catch (err) {
    detailEl.innerHTML = `<div class="empty">Failed to load: ${err.message}</div>`;
    return;
  }
  renderDetail();
}

function renderDetail() {
  const d = state.detail;
  if (!d) return;
  const summary = d.summary;
  const tabs = ['overview', 'planets', 'commands', 'decisions', 'raw'];
  const pauseLabel = summary.paused ? 'Resume' : 'Pause';
  el('detail').innerHTML = `
    <div class="detail-head">
      <h2>game ${summary.game_id} · player ${summary.player_id}</h2>
      <span class="status-pill ${summary.status}">${summary.status}</span>
      <span class="tick-delay">tick ${summary.tick_delay_ms}ms</span>
      <button id="pause-agent" class="${summary.paused ? 'resume' : 'pause'}">${pauseLabel}</button>
      <button id="stop-agent">Stop</button>
    </div>
    <div class="tabs">
      ${tabs.map(t => `<button data-tab="${t}" class="${t === state.detailTab ? 'active' : ''}">${t}</button>`).join('')}
    </div>
    <div class="tab-body" id="tab-body"></div>
  `;
  el('stop-agent').addEventListener('click', () => stopAgent(summary.id));
  el('pause-agent').addEventListener('click', () => togglePause(summary.id, !summary.paused));
  document.querySelectorAll('.tabs button').forEach(btn => {
    btn.addEventListener('click', () => { state.detailTab = btn.dataset.tab; renderTabBody(); });
  });
  renderTabBody();
}

function renderTabBody() {
  const d = state.detail;
  const body = el('tab-body');
  if (!body || !d) return;
  switch (state.detailTab) {
    case 'overview': body.innerHTML = renderOverview(d); break;
    case 'planets': body.innerHTML = renderPlanets(d); break;
    case 'commands': body.innerHTML = renderLogList(d.log, ['command']); break;
    case 'decisions': body.innerHTML = renderLogList(d.log, ['decision', 'info', 'error']); break;
    case 'raw': body.innerHTML = `<pre class="raw">${escapeHTML(JSON.stringify(d.snapshot, null, 2))}</pre>`; break;
  }
}

function renderOverview(d) {
  const s = d.summary;
  const ts = d.snapshot && d.snapshot.turn_status ? d.snapshot.turn_status : {};
  const research = (d.snapshot && d.snapshot.players || []).find(p => p.id === s.player_id)?.current_research;
  return `
    <dl class="kv">
      <dt>agent id</dt><dd>${s.id}</dd>
      <dt>server</dt><dd>${s.server}</dd>
      <dt>policy</dt><dd>${s.policy}</dd>
      <dt>status</dt><dd>${s.status}${s.last_error ? ' — ' + escapeHTML(s.last_error) : ''}</dd>
      <dt>current turn</dt><dd>${s.current_turn}</dd>
      <dt>ready players</dt><dd>${(ts.ready_player_ids || []).join(', ')}</dd>
      <dt>waiting players</dt><dd>${(ts.waiting_player_ids || []).join(', ')}</dd>
      <dt>owned planets</dt><dd>${s.owned_planets}</dd>
      <dt>owned fleets</dt><dd>${s.owned_fleets}</dd>
      <dt>research</dt><dd>${research ? escapeHTML(research.tech_name || research.subject || '(picking)') : '(none)'}</dd>
    </dl>
  `;
}

function renderPlanets(d) {
  const s = d.summary;
  if (!d.snapshot || !d.snapshot.galaxy) return '<div class="empty">(no snapshot yet)</div>';
  const owned = [];
  for (const star of d.snapshot.galaxy.stars || []) {
    for (const p of (star.planets || [])) {
      if (p.owner === s.player_id) owned.push({ star, planet: p });
    }
  }
  if (!owned.length) return '<div class="empty">(no owned planets yet)</div>';
  return `<div class="planet-grid">${owned.map(({ star, planet }) => renderPlanetCard(star, planet)).join('')}</div>`;
}

function renderPlanetCard(star, p) {
  const food = p.food || {};
  const ind = p.industry || {};
  const sci = p.science || {};
  const inc = p.income || {};
  const totalAllocated = (food.farmers || 0) + (ind.workers || 0) + (sci.scientists || 0);
  const foodOut = (food.farmers || 0) * (food.produced_per_farmer || 0) + (food.produced_per_building || 0);
  const indOut = (ind.workers || 0) * (ind.produced_per_worker || 0) + (ind.produced_per_building || 0) + (p.population || 0) * (ind.produced_per_colonist || 0);
  const sciOut = (sci.scientists || 0) * (sci.produced_per_scientist || 0) + (sci.produced_per_building || 0);
  const buildings = (p.buildings || []).slice().sort();
  const queue = p.build_queue || [];
  const project = p.current_project ? `${escapeHTML(p.current_project)} (${p.build_bucket || 0} bc banked)` : '(idle)';
  return `
    <div class="planet-card">
      <div class="planet-head">
        <div class="planet-name">${escapeHTML(p.name || '?')}</div>
        <div class="planet-sub">${escapeHTML(star.name || 'star ' + star.id)} · pop ${p.population}/${p.max_population} (Δ${p.last_turn_pop_growth || 0})</div>
      </div>
      <div class="planet-section">
        <div class="planet-section-head">Labor</div>
        <table class="kvt">
          <tr><td>Farmers</td><td>${food.farmers || 0}</td><td>food ${foodOut} ${foodOut < p.population ? '⚠' : ''}</td></tr>
          <tr><td>Workers</td><td>${ind.workers || 0}</td><td>ind ${indOut}</td></tr>
          <tr><td>Scientists</td><td>${sci.scientists || 0}</td><td>sci ${sciOut}</td></tr>
          <tr><td>Total alloc</td><td>${totalAllocated}</td><td>of ${p.population}</td></tr>
        </table>
      </div>
      <div class="planet-section">
        <div class="planet-section-head">Building</div>
        <div>Project: ${project}</div>
        ${queue.length ? `<div>Queue: ${queue.map(escapeHTML).join(' → ')}</div>` : ''}
      </div>
      <div class="planet-section">
        <div class="planet-section-head">Buildings (${buildings.length})</div>
        <ul class="building-list">${buildings.map(b => `<li>${escapeHTML(b)}</li>`).join('') || '<li class="muted">(none)</li>'}</ul>
      </div>
      <div class="planet-section muted small">
        ${escapeHTML(stringifyEnv(p))} · gravity ${p.gravity} · minerals ${p.mineral_class} · income ${inc.total || 0}
      </div>
    </div>
  `;
}

function stringifyEnv(p) {
  // PlanetEnvironment/Mineral/Size enums marshal as numbers; show their raw
  // values — the operator can look them up in the model package if needed.
  return `env ${p.environment} · size ${p.size}`;
}

function renderLogList(entries, kinds) {
  if (!entries || !entries.length) return '<div class="empty">(no entries yet)</div>';
  const filtered = entries.filter(e => kinds.includes(e.kind));
  if (!filtered.length) return '<div class="empty">(no entries yet)</div>';
  const rows = filtered.slice().reverse().map(e => {
    const meta = `<div class="meta">${new Date(e.timestamp).toLocaleTimeString()} · turn ${e.turn} · ${e.kind}${e.advanced ? ' · advanced' : ''}</div>`;
    let extra = '';
    if (e.request) extra += `<details><summary>request</summary><pre>${escapeHTML(prettyJSON(e.request))}</pre></details>`;
    if (e.response) extra += `<details><summary>response</summary><pre>${escapeHTML(prettyJSON(e.response))}</pre></details>`;
    if (e.error) extra += `<pre>${escapeHTML(e.error)}</pre>`;
    const message = e.message ? `<div>${escapeHTML(e.message)}</div>` : '';
    return `<li class="${e.kind}">${meta}${message}${extra}</li>`;
  }).join('');
  return `<ul class="log-list">${rows}</ul>`;
}

function prettyJSON(v) {
  if (v == null) return '';
  if (typeof v === 'object') return JSON.stringify(v, null, 2);
  try { return JSON.stringify(JSON.parse(v), null, 2); } catch (_) { return String(v); }
}

function escapeHTML(s) {
  return String(s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

async function togglePause(id, pause) {
  try {
    const updated = await api(`/api/agents/${id}/${pause ? 'pause' : 'resume'}`, { method: 'POST' });
    state.agents.set(id, updated);
    renderAgentList();
    if (state.selectedId === id) await refreshDetail();
  } catch (err) {
    alert((pause ? 'Pause' : 'Resume') + ' failed: ' + err.message);
  }
}

async function stopAgent(id) {
  try {
    await api(`/api/agents/${id}`, { method: 'DELETE' });
    state.agents.delete(id);
    if (state.selectedId === id) state.selectedId = null;
    renderAgentList();
    await refreshDetail();
  } catch (err) {
    alert('Stop failed: ' + err.message);
  }
}

// --- Add agent dialog ---

el('add-agent').addEventListener('click', () => openAddDialog());
el('add-cancel').addEventListener('click', () => el('add-dialog').close());
el('add-form').addEventListener('submit', (e) => { e.preventDefault(); submitAddForm(); });
el('form-server').addEventListener('change', loadGamesForForm);
el('form-game').addEventListener('change', loadPlayersForForm);

async function openAddDialog() {
  el('add-error').textContent = '';
  el('form-server').value = state.defaultServer;
  await loadGamesForForm();
  el('add-dialog').showModal();
}

async function loadGamesForForm() {
  const server = el('form-server').value.trim();
  const gameSel = el('form-game');
  gameSel.innerHTML = '<option>loading…</option>';
  try {
    const r = await api(`/api/games?server=${encodeURIComponent(server)}`);
    const games = (r.game_list || []);
    if (!games.length) { gameSel.innerHTML = '<option value="">(no games)</option>'; el('form-player').innerHTML = ''; return; }
    gameSel.innerHTML = games.map(g => {
      const players = (g.players || []).map(p => `${p.name} (P${p.id})`).join(', ');
      return `<option value="${g.game_id}" data-players='${JSON.stringify(g.players || [])}'>${g.game_id.slice(0, 8)} — turn ${Math.round(g.current_turn)} — ${players}</option>`;
    }).join('');
    loadPlayersForForm();
  } catch (err) {
    gameSel.innerHTML = '';
    el('add-error').textContent = 'Could not list games: ' + err.message;
  }
}

function loadPlayersForForm() {
  const gameSel = el('form-game');
  const opt = gameSel.selectedOptions[0];
  const playerSel = el('form-player');
  if (!opt) { playerSel.innerHTML = ''; return; }
  let players = [];
  try { players = JSON.parse(opt.dataset.players || '[]'); } catch (_) {}
  playerSel.innerHTML = players.map(p => `<option value="${p.id}">${p.name} (P${p.id})</option>`).join('');
}

async function submitAddForm() {
  const body = {
    server: el('form-server').value.trim(),
    game_id: el('form-game').value,
    player_id: parseInt(el('form-player').value, 10),
  };
  try {
    const a = await api('/api/agents', { method: 'POST', body: JSON.stringify(body) });
    el('add-dialog').close();
    state.agents.set(a.id, a);
    renderAgentList();
    selectAgent(a.id);
  } catch (err) {
    el('add-error').textContent = err.message;
  }
}

// --- Live updates over WS ---

function openWS() {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  const ws = new WebSocket(`${proto}://${location.host}/ws`);
  ws.onopen = () => setConn('connected', 'ok');
  ws.onclose = () => { setConn('disconnected — retrying', 'bad'); setTimeout(openWS, 1500); };
  ws.onerror = () => setConn('error', 'bad');
  ws.onmessage = (e) => {
    let ev; try { ev = JSON.parse(e.data); } catch (_) { return; }
    handleEvent(ev);
  };
}

function handleEvent(ev) {
  switch (ev.kind) {
    case 'agent_changed':
      if (ev.summary) state.agents.set(ev.agent_id, ev.summary);
      renderAgentList();
      if (state.selectedId === ev.agent_id) refreshDetail();
      break;
    case 'agent_removed':
      state.agents.delete(ev.agent_id);
      if (state.selectedId === ev.agent_id) { state.selectedId = null; state.detail = null; }
      renderAgentList();
      refreshDetail();
      break;
    case 'agent_log':
      if (state.selectedId === ev.agent_id && state.detail) {
        state.detail.log = (state.detail.log || []).concat([ev.entry]);
        if (state.detail.log.length > 1000) state.detail.log.splice(0, state.detail.log.length - 1000);
        renderTabBody();
      }
      break;
  }
}

async function initial() {
  try {
    const r = await api('/api/agents');
    for (const s of (r.agents || [])) state.agents.set(s.id, s);
    renderAgentList();
  } catch (_) {}
  openWS();
}

initial();
