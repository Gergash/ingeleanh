'use strict';

const REGISTRY_AMOY = '0x629238eD79c23267fe502AAd81E5AEfee3908750';
const POLYGONSCAN = 'https://amoy.polygonscan.com/address/' + REGISTRY_AMOY;
const DEMO_GATEWAY_ID = 'a1000000-0000-4000-8000-000000000001';

/** Canonical API paths — single source of truth for routing. */
const ROUTES = {
  login: () => '/api/v1/operator/login',
  portalInfo: () => '/api/v1/portal/info',
  agents: () => '/api/v1/agents',
  tasks: () => '/api/v1/tasks',
  task: (taskId) => '/api/v1/tasks/' + encodeURIComponent(taskId),
  events: (limit) => '/api/v1/events?limit=' + encodeURIComponent(String(limit)),
  devices: () => '/api/v1/devices',
  deviceCommand: (deviceId) =>
    '/api/v1/devices/' + encodeURIComponent(deviceId) + '/command',
  chainStatus: () => '/api/v1/chain/status',
  demoSeed: () => '/api/v1/demo/seed',
  demoReplayAccess: () => '/api/v1/demo/replay-access',
  demoThreeLayer: () => '/api/v1/demo/three-layer',
};

const AppState = {
  token: localStorage.getItem('c2_portal_jwt') || '',
  demoGatewayId: '',
  refreshTimer: null,
  refreshInFlight: false,
  pollGeneration: 0,
};

function apiBase() {
  const raw = (typeof window !== 'undefined' && window.C2_API_BASE) || '';
  return String(raw).replace(/\/$/, '');
}

function buildUrl(routePath) {
  if (!routePath.startsWith('/api/v1')) {
    throw new Error('Ruta API inválida: ' + routePath);
  }
  const base = apiBase();
  return base ? base + routePath : routePath;
}

function setToken(token) {
  AppState.token = token || '';
  if (AppState.token) {
    localStorage.setItem('c2_portal_jwt', AppState.token);
  } else {
    localStorage.removeItem('c2_portal_jwt');
  }
}

function cancelActivePoll() {
  AppState.pollGeneration += 1;
}

/**
 * Central HTTP client — all portal traffic goes through here.
 * @param {string} routePath - path from ROUTES (must start with /api/v1)
 * @param {{ method?: string, body?: object|string, auth?: boolean }} options
 */
async function request(routePath, options = {}) {
  const method = options.method || 'GET';
  const auth = options.auth !== false;
  const headers = { Accept: 'application/json' };

  if (auth && AppState.token) {
    headers.Authorization = 'Bearer ' + AppState.token;
  }

  let bodyPayload = undefined;
  if (options.body !== undefined && options.body !== null) {
    headers['Content-Type'] = 'application/json';
    bodyPayload =
      typeof options.body === 'string' ? options.body : JSON.stringify(options.body);
  }

  const url = buildUrl(routePath);
  const response = await fetch(url, { method, headers, body: bodyPayload });
  const data = await response.json().catch(() => ({}));

  if (!response.ok) {
    throw new Error(data.error_code || data.message || response.statusText || 'REQUEST_FAILED');
  }
  return data;
}

const API = {
  login: (username, password) =>
    request(ROUTES.login(), {
      method: 'POST',
      body: { username, password },
      auth: false,
    }),
  portalInfo: () => request(ROUTES.portalInfo(), { auth: false }),
  agents: () => request(ROUTES.agents()),
  devices: () => request(ROUTES.devices()),
  events: (limit) => request(ROUTES.events(limit)),
  chainStatus: () => request(ROUTES.chainStatus()),
  createTask: (taskBody) =>
    request(ROUTES.tasks(), { method: 'POST', body: taskBody }),
  getTask: (taskId) => request(ROUTES.task(taskId)),
  lockCommand: (action) =>
    request(ROUTES.deviceCommand('lock-main'), {
      method: 'POST',
      body: { action, duration_sec: 5 },
    }),
  demoSeed: () => request(ROUTES.demoSeed(), { method: 'POST', body: {} }),
  demoReplayAccess: () =>
    request(ROUTES.demoReplayAccess(), { method: 'POST', body: {} }),
  demoThreeLayer: () =>
    request(ROUTES.demoThreeLayer(), { method: 'POST', body: {} }),
};

function show(id) {
  document.getElementById(id).classList.remove('hidden');
}
function hide(id) {
  document.getElementById(id).classList.add('hidden');
}

function taskResultEl() {
  return document.getElementById('task-result');
}

async function login() {
  const user = document.getElementById('login-user').value.trim();
  const pass = document.getElementById('login-pass').value;
  const err = document.getElementById('login-error');
  err.textContent = '';
  try {
    const data = await API.login(user, pass);
    if (!data.token) {
      throw new Error('Respuesta sin token');
    }
    setToken(data.token);
    hide('login-screen');
    show('app-screen');
    await refreshAll();
    startAutoRefresh();
  } catch (e) {
    err.textContent = e.message;
  }
}

function logout() {
  cancelActivePoll();
  setToken('');
  AppState.demoGatewayId = '';
  stopAutoRefresh();
  hide('app-screen');
  show('login-screen');
}

function startAutoRefresh() {
  stopAutoRefresh();
  AppState.refreshTimer = setInterval(() => refreshAll(true), 15000);
}

function stopAutoRefresh() {
  if (AppState.refreshTimer) {
    clearInterval(AppState.refreshTimer);
    AppState.refreshTimer = null;
  }
}

async function refreshAll(silent) {
  if (AppState.refreshInFlight) return;
  AppState.refreshInFlight = true;
  if (!silent) document.getElementById('refresh-status').textContent = 'Actualizando…';
  try {
    const info = await API.portalInfo();
    AppState.demoGatewayId = info.demo_gateway_id || '';
    document.getElementById('demo-badge').classList.toggle('hidden', !info.demo_mode);

    const agents = await API.agents();
    const list = agents.agents || [];
    const active = list.filter((a) => a.status === 'active').length;
    document.getElementById('stat-agents').textContent = active;
    document.getElementById('stat-agents-sub').textContent = list.length + ' registrados';

    const devices = await API.devices();
    const devs = devices.devices || [];
    document.getElementById('stat-devices').textContent = devs.length;

    const events = await API.events(30);
    document.getElementById('stat-events').textContent = events.total || 0;

    const chain = await API.chainStatus();
    document.getElementById('stat-chain').textContent = 'v' + (chain.config_version || 0);
    document.getElementById('chain-bar').innerHTML = buildChainBar(chain);

    renderAgents(list);
    renderDevices(devs);
    renderEvents(events.events || []);
    populateAgentSelect(list);
    document.getElementById('refresh-status').textContent =
      'Actualizado ' + new Date().toLocaleTimeString();
  } catch (e) {
    if (!silent) {
      document.getElementById('refresh-status').textContent = 'Error: ' + e.message;
    }
    if (e.message.includes('UNAUTHORIZED') || e.message.includes('401')) {
      logout();
    }
  } finally {
    AppState.refreshInFlight = false;
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
  el.innerHTML =
    '<table><tr><th>ID</th><th>Host</th><th>OS</th><th>Rol</th><th>Beacon</th></tr>' +
    agents
      .map(
        (a) =>
          `<tr>
      <td class="mono">${a.agent_id.slice(0, 8)}…</td>
      <td>${a.hostname || '—'}</td>
      <td>${a.os || '—'}</td>
      <td>${a.agent_role || 'generic'}</td>
      <td>${formatBeacon(a.last_beacon)}</td>
    </tr>`
      )
      .join('') +
    '</table>';
}

function formatBeacon(ts) {
  if (!ts) return '—';
  try {
    return new Date(ts).toLocaleString();
  } catch {
    return ts;
  }
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
  devs.forEach((d) => {
    byId[d.device_id] = d;
  });

  el.innerHTML = catalog
    .map((c) => {
      const d = byId[c.id] || {};
      const lock = d.lock_state || (c.id === 'lock-main' ? 'locked' : 'active');
      const isLock = c.id === 'lock-main';
      const stateClass = lock === 'unlocked' ? 'state-ok' : isLock ? 'state-warn' : 'state-ok';
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
    })
    .join('');
}

function renderEvents(events) {
  const el = document.getElementById('events-list');
  if (!events.length) {
    el.innerHTML =
      '<p style="color:var(--muted)">Sin eventos. Carga demo o inicia gateway IoT.</p>';
    return;
  }
  el.innerHTML = events
    .map((e) => {
      const summary =
        e.payload_summary || e.device_id || e.event_type || JSON.stringify(e).slice(0, 60);
      return `<div class="event-row">
      <div class="time">${e.created_at || ''}</div>
      <div><strong>${e.event_type || 'event'}</strong> — ${summary}</div>
    </div>`;
    })
    .join('');
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
  sel.innerHTML = live
    .map(
      (a) =>
        `<option value="${a.agent_id}">${a.hostname || 'agent'} (${a.agent_id.slice(0, 8)}…)</option>`
    )
    .join('');
}

async function runWhoami() {
  cancelActivePoll();
  const pollGen = AppState.pollGeneration;

  const agentId = document.getElementById('task-agent').value;
  const out = taskResultEl();
  if (!agentId) {
    out.textContent = 'Selecciona un agente C2 en vivo (go run ./cmd/agent).';
    return;
  }
  out.textContent = 'Enviando tarea whoami…';
  try {
    const created = await API.createTask({
      agent_id: agentId,
      command_type: 'shell',
      payload: { argv: ['whoami'] },
    });
    if (!created.task_id) {
      out.textContent = 'Error: respuesta sin task_id — ' + JSON.stringify(created);
      return;
    }
    out.textContent =
      'pending task_id=' + created.task_id + ' — esperando agente (~30s)…';
    pollTask(created.task_id, 0, pollGen);
  } catch (e) {
    out.textContent = 'Error: ' + e.message;
  }
}

function pollTask(taskId, attempt, pollGen) {
  if (pollGen !== AppState.pollGeneration) return;

  if (attempt > 12) {
    taskResultEl().textContent =
      'Tiempo agotado. Verifica que el agente esté corriendo (go run ./cmd/agent).';
    return;
  }

  API.getTask(taskId)
    .then((t) => {
      if (pollGen !== AppState.pollGeneration) return;
      const out = taskResultEl();
      if (t.status === 'completed' || t.status === 'failed') {
        out.textContent = JSON.stringify(t, null, 2);
        return;
      }
      setTimeout(() => pollTask(taskId, attempt + 1, pollGen), 2500);
    })
    .catch((e) => {
      if (pollGen !== AppState.pollGeneration) return;
      if (e.message === 'NOT_FOUND' && attempt < 3) {
        setTimeout(() => pollTask(taskId, attempt + 1, pollGen), 1000);
        return;
      }
      taskResultEl().textContent =
        'Error poll: ' + e.message + ' — reinicia server+agente si persiste.';
    });
}

async function lockCommand(action) {
  const out = taskResultEl();
  try {
    const res = await API.lockCommand(action);
    out.textContent = 'Cerradura: ' + JSON.stringify(res);
    await refreshAll(true);
  } catch (e) {
    out.textContent = 'Error cerradura: ' + e.message;
  }
}

async function runThreeLayer() {
  const out = taskResultEl();
  out.textContent = 'Ejecutando DEMO-011 (IoT → C2 → blockchain)…';
  try {
    const res = await API.demoThreeLayer();
    const lines = (res.steps || []).map(
      (s) => '[' + s.layer + '] ' + s.title + ': ' + (s.detail || '')
    );
    out.textContent =
      res.narrative + '\n\n' + lines.join('\n') +
      '\n\nchain: ' + JSON.stringify(res.chain, null, 2);
    await refreshAll(true);
  } catch (e) {
    out.textContent = 'DEMO-011: ' + e.message;
  }
}

async function replayAccess() {
  const out = taskResultEl();
  try {
    const res = await API.demoReplayAccess();
    out.textContent =
      'Acceso Laureles simulado: ' +
      ((res.event && res.event.payload_summary) || JSON.stringify(res));
    await refreshAll(true);
  } catch (e) {
    out.textContent = 'Replay CSV: ' + e.message;
  }
}

async function seedDemo() {
  try {
    const res = await API.demoSeed();
    alert(res.message || 'Demo cargado');
    await refreshAll();
  } catch (e) {
    alert('Seed: ' + e.message);
  }
}

document.addEventListener('DOMContentLoaded', () => {
  if (AppState.token) {
    hide('login-screen');
    show('app-screen');
    refreshAll();
    startAutoRefresh();
  }
});
