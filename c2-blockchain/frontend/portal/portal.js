const REGISTRY_AMOY = '0x629238eD79c23267fe502AAd81E5AEfee3908750';
const POLYGONSCAN = 'https://amoy.polygonscan.com/address/' + REGISTRY_AMOY;
const DEMO_GATEWAY_ID = 'a1000000-0000-4000-8000-000000000001';

let token = localStorage.getItem('c2_portal_jwt') || '';
let refreshTimer = null;
let demoGatewayId = '';

function show(id) { document.getElementById(id).classList.remove('hidden'); }
function hide(id) { document.getElementById(id).classList.add('hidden'); }

async function api(path, opts = {}) {
  const headers = { ...(opts.headers || {}) };
  if (token) headers.Authorization = 'Bearer ' + token;
  if (opts.body) headers['Content-Type'] = 'application/json';
  const r = await fetch('/api/v1' + path, { ...opts, headers });
  const data = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(data.error_code || data.message || r.statusText);
  return data;
}

async function login() {
  const user = document.getElementById('login-user').value.trim();
  const pass = document.getElementById('login-pass').value;
  const err = document.getElementById('login-error');
  err.textContent = '';
  try {
    const r = await fetch('/api/v1/operator/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: user, password: pass }),
    });
    const data = await r.json();
    if (!r.ok) throw new Error(data.error_code || 'Login fallido');
    token = data.token;
    localStorage.setItem('c2_portal_jwt', token);
    hide('login-screen');
    show('app-screen');
    await refreshAll();
    startAutoRefresh();
  } catch (e) {
    err.textContent = e.message;
  }
}

function logout() {
  token = '';
  localStorage.removeItem('c2_portal_jwt');
  stopAutoRefresh();
  hide('app-screen');
  show('login-screen');
}

function startAutoRefresh() {
  stopAutoRefresh();
  refreshTimer = setInterval(() => refreshAll(true), 15000);
}

function stopAutoRefresh() {
  if (refreshTimer) clearInterval(refreshTimer);
}

async function refreshAll(silent) {
  if (!silent) document.getElementById('refresh-status').textContent = 'Actualizando…';
  try {
    const info = await api('/portal/info');
    demoGatewayId = info.demo_gateway_id || '';
    document.getElementById('demo-badge').classList.toggle('hidden', !info.demo_mode);

    const agents = await api('/agents');
    const list = agents.agents || [];
    const active = list.filter(a => a.status === 'active').length;
    document.getElementById('stat-agents').textContent = active;
    document.getElementById('stat-agents-sub').textContent = list.length + ' registrados';

    const devices = await api('/devices');
    const devs = devices.devices || [];
    document.getElementById('stat-devices').textContent = devs.length;

    const events = await api('/events?limit=30');
    document.getElementById('stat-events').textContent = events.total || 0;

    const chain = await api('/chain/status');
    document.getElementById('stat-chain').textContent = 'v' + (chain.config_version || 0);
    document.getElementById('chain-bar').innerHTML = buildChainBar(chain);

    renderAgents(list);
    renderDevices(devs);
    renderEvents(events.events || []);
    populateAgentSelect(list);
    document.getElementById('refresh-status').textContent = 'Actualizado ' + new Date().toLocaleTimeString();
  } catch (e) {
    if (!silent) document.getElementById('refresh-status').textContent = 'Error: ' + e.message;
    if (e.message.includes('UNAUTHORIZED') || e.message.includes('401')) logout();
  }
}

function buildChainBar(chain) {
  const reg = chain.contract_address || REGISTRY_AMOY;
  const ver = chain.config_version ?? 0;
  const beacon = chain.beacon_interval_sec ?? '—';
  const hash = (chain.endpoint_hash || '').slice(0, 18) + '…';
  return `
    <span><strong>Polygon Amoy</strong> · chain ${chain.chain_id || 80002}</span>
    <span>Config <strong>v${ver}</strong> · beacon ${beacon}s</span>
    <span class="mono">hash ${hash}</span>
    <span class="mono">registry ${reg.slice(0, 10)}…${reg.slice(-6)}</span>
    <a href="${POLYGONSCAN}" target="_blank" rel="noopener">Ver en Polygonscan ↗</a>
  `;
}

function renderAgents(agents) {
  const el = document.getElementById('agents-table');
  if (!agents.length) {
    el.innerHTML = '<p class="muted">Sin agentes. Inicia el agente o usa modo demo.</p>';
    return;
  }
  el.innerHTML = '<table><tr><th>ID</th><th>Host</th><th>OS</th><th>Rol</th><th>Beacon</th></tr>' +
    agents.map(a => `<tr>
      <td class="mono">${a.agent_id.slice(0, 8)}…</td>
      <td>${a.hostname || '—'}</td>
      <td>${a.os || '—'}</td>
      <td>${a.agent_role || 'generic'}</td>
      <td>${formatBeacon(a.last_beacon)}</td>
    </tr>`).join('') + '</table>';
}

function formatBeacon(ts) {
  if (!ts) return '—';
  try { return new Date(ts).toLocaleString(); } catch { return ts; }
}

function renderDevices(devs) {
  const el = document.getElementById('devices-grid');
  const catalog = [
    { id: 'sensor-access-gate', label: 'Lector acceso Laureles', icon: '🚪' },
    { id: 'sensor-motion-01', label: 'Sensor movimiento', icon: '📡' },
    { id: 'meter-energy-01', label: 'Medidor energía', icon: '⚡' },
    { id: 'lock-main', label: 'Cerradura principal', icon: '🔐' },
  ];
  const byId = {};
  devs.forEach(d => { byId[d.device_id] = d; });

  el.innerHTML = catalog.map(c => {
    const d = byId[c.id] || {};
    const lock = d.lock_state || (c.id === 'lock-main' ? 'locked' : 'active');
    const isLock = c.id === 'lock-main';
    const stateClass = lock === 'unlocked' ? 'state-ok' : (isLock ? 'state-warn' : 'state-ok');
    let actions = '';
    if (isLock) {
      actions = `
        <button class="btn btn-sm btn-primary" onclick="lockCommand('unlock')">Abrir</button>
        <button class="btn btn-sm btn-secondary" onclick="lockCommand('lock')">Cerrar</button>
      `;
    }
    return `<div class="device-card">
      <div class="dtype">${c.icon} ${d.device_type || c.label}</div>
      <div class="name">${c.id}</div>
      <div class="state ${stateClass}">${isLock ? 'Estado: ' + lock : 'Zona: ' + (d.zone || 'lobby')}</div>
      ${actions}
    </div>`;
  }).join('');
}

function renderEvents(events) {
  const el = document.getElementById('events-list');
  if (!events.length) {
    el.innerHTML = '<p style="color:var(--muted)">Sin eventos. Carga demo o inicia gateway IoT.</p>';
    return;
  }
  el.innerHTML = events.map(e => {
    const summary = e.payload_summary || e.device_id || e.event_type || JSON.stringify(e).slice(0, 60);
    return `<div class="event-row">
      <div class="time">${e.created_at || ''}</div>
      <div><strong>${e.event_type || 'event'}</strong> — ${summary}</div>
    </div>`;
  }).join('');
}

function isLiveC2Agent(a) {
  if (a.agent_id === DEMO_GATEWAY_ID) return false;
  if (a.agent_role === 'iot_gateway') return false;
  if (!a.last_beacon) return false;
  const age = Date.now() - new Date(a.last_beacon).getTime();
  return age < 5 * 60 * 1000;
}

function populateAgentSelect(agents) {
  const sel = document.getElementById('task-agent');
  const hint = document.getElementById('task-agent-hint');
  const live = agents.filter(isLiveC2Agent);
  if (!live.length) {
    sel.innerHTML = '';
    hint.textContent = 'Sin agente C2 en vivo. Ejecuta: go run ./cmd/agent';
    hint.style.color = '#d29922';
    return;
  }
  hint.textContent = 'Agente con beacon reciente (no simulado)';
  hint.style.color = '#8b949e';
  live.sort((a, b) => (b.last_beacon || '').localeCompare(a.last_beacon || ''));
  sel.innerHTML = live.map(a =>
    `<option value="${a.agent_id}">${a.hostname || 'agent'} (${a.agent_id.slice(0, 8)}…)</option>`
  ).join('');
}

async function runWhoami() {
  const agentId = document.getElementById('task-agent').value;
  const out = document.getElementById('task-result');
  if (!agentId) {
    out.textContent = 'Selecciona un agente C2 en vivo (go run ./cmd/agent).';
    return;
  }
  out.textContent = 'Enviando tarea whoami…';
  try {
    const created = await api('/tasks', {
      method: 'POST',
      body: JSON.stringify({
        agent_id: agentId,
        command_type: 'shell',
        payload: { argv: ['whoami'] },
      }),
    });
    if (!created.task_id) {
      out.textContent = 'Error: respuesta sin task_id — ' + JSON.stringify(created);
      return;
    }
    out.textContent = 'pending task_id=' + created.task_id + ' — esperando agente (~30s)…';
    pollTask(created.task_id, 0);
  } catch (e) {
    out.textContent = 'Error: ' + e.message;
  }
}

async function pollTask(taskId, attempt) {
  if (attempt > 12) {
    document.getElementById('task-result').textContent =
      'Tiempo agotado. Verifica que el agente esté corriendo (go run ./cmd/agent).';
    return;
  }
  try {
    const t = await api('/tasks/' + taskId);
    const out = document.getElementById('task-result');
    if (t.status === 'completed' || t.status === 'failed') {
      out.textContent = JSON.stringify(t, null, 2);
      return;
    }
    setTimeout(() => pollTask(taskId, attempt + 1), 2500);
  } catch (e) {
    if (e.message === 'NOT_FOUND' && attempt < 3) {
      setTimeout(() => pollTask(taskId, attempt + 1), 1000);
      return;
    }
    document.getElementById('task-result').textContent =
      'Error poll: ' + e.message + ' — reinicia server+agente si persiste.';
  }
}

async function lockCommand(action) {
  try {
    const res = await api('/devices/lock-main/command', {
      method: 'POST',
      body: JSON.stringify({ action, duration_sec: 5 }),
    });
    document.getElementById('task-result').textContent = 'Cerradura: ' + JSON.stringify(res);
    await refreshAll(true);
  } catch (e) {
    document.getElementById('task-result').textContent = 'Error cerradura: ' + e.message;
  }
}

async function runThreeLayer() {
  const out = document.getElementById('task-result');
  out.textContent = 'Ejecutando DEMO-011 (IoT → C2 → blockchain)…';
  try {
    const res = await api('/demo/three-layer', { method: 'POST', body: '{}' });
    const lines = (res.steps || []).map(s =>
      `[${s.layer}] ${s.title}: ${s.detail || ''}`
    );
    out.textContent = res.narrative + '\n\n' + lines.join('\n') +
      '\n\nchain: ' + JSON.stringify(res.chain, null, 2);
    await refreshAll(true);
  } catch (e) {
    out.textContent = 'DEMO-011: ' + e.message;
  }
}

async function replayAccess() {
  try {
    const res = await api('/demo/replay-access', { method: 'POST', body: '{}' });
    document.getElementById('task-result').textContent =
      'Acceso Laureles simulado: ' + (res.event && res.event.payload_summary || JSON.stringify(res));
    await refreshAll(true);
  } catch (e) {
    document.getElementById('task-result').textContent = 'Replay CSV: ' + e.message;
  }
}

async function seedDemo() {
  try {
    const res = await api('/demo/seed', { method: 'POST', body: '{}' });
    alert(res.message || 'Demo cargado');
    await refreshAll();
  } catch (e) {
    alert('Seed: ' + e.message);
  }
}

document.addEventListener('DOMContentLoaded', () => {
  if (token) {
    hide('login-screen');
    show('app-screen');
    refreshAll();
    startAutoRefresh();
  }
});
