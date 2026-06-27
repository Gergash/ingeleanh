'use strict';

/** CIR Dashboard — integración con backend C2 Blockchain-Blindado */
const CIR_ROUTES = {
  login: () => '/api/v1/operator/login',
  info: () => '/api/v1/cir/info',
  summary: () => '/api/v1/cir/summary',
  alerts: () => '/api/v1/cir/alerts',
  pulse: () => '/api/v1/cir/pulse',
  simulateAttack: (full) =>
    '/api/v1/cir/simulate-attack' + (full ? '?full=true' : ''),
  chainStatus: () => '/api/v1/chain/status',
};

const CIRState = {
  token: localStorage.getItem('cir_jwt') || '',
  summary: null,
  charts: {},
  refreshTimer: null,
};

function cirApiBase() {
  const raw = (typeof window !== 'undefined' && window.C2_API_BASE) || '';
  return String(raw).replace(/\/$/, '');
}

function cirUrl(path) {
  const base = cirApiBase();
  return base ? base + path : path;
}

async function cirRequest(path, options = {}) {
  const headers = { Accept: 'application/json' };
  if (options.auth !== false && CIRState.token) {
    headers.Authorization = 'Bearer ' + CIRState.token;
  }
  if (options.attack) {
    headers['X-Attack-Token'] = 'compromised';
  }
  let body;
  if (options.body) {
    headers['Content-Type'] = 'application/json';
    body = JSON.stringify(options.body);
  }
  const res = await fetch(cirUrl(path), {
    method: options.method || 'GET',
    headers,
    body,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error_code || res.statusText);
  }
  return data;
}

async function cirLogin(user, pass) {
  const data = await cirRequest(CIR_ROUTES.login(), {
    auth: false,
    method: 'POST',
    body: { username: user, password: pass },
  });
  CIRState.token = data.token;
  localStorage.setItem('cir_jwt', data.token);
  return data.token;
}

async function cirEnsureAuth() {
  if (CIRState.token) return CIRState.token;
  try {
    const info = await cirRequest(CIR_ROUTES.info(), { auth: false });
    return await cirLogin(info.default_user || 'operator', 'lab');
  } catch {
    return await cirLogin('operator', 'lab');
  }
}

function fmtNum(n) {
  return Number(n || 0).toLocaleString('es-CO');
}

function timeAgo(iso) {
  if (!iso) return 'ahora';
  const d = new Date(iso);
  const sec = Math.floor((Date.now() - d.getTime()) / 1000);
  if (sec < 60) return 'hace ' + sec + ' s';
  if (sec < 3600) return 'hace ' + Math.floor(sec / 60) + ' min';
  return 'hace ' + Math.floor(sec / 3600) + ' h';
}

function sevClass(sev) {
  if (sev === 'crit') return 'al-crit';
  if (sev === 'warn') return 'al-warn';
  return 'al-ok';
}

function tipoBadge(tipo) {
  const t = String(tipo || '').toLowerCase();
  if (t.includes('resident')) return 'pb-r';
  if (t.includes('domicil')) return 'pb-d';
  if (t.includes('alert')) return 'pb-a';
  return 'pb-v';
}

function estadoClass(estado) {
  const e = String(estado || '').toLowerCase();
  if (e.includes('bloq')) return 's-block';
  if (e.includes('verif')) return 's-warn';
  return 's-ok';
}

function updateKPIs(data) {
  const sum = data.summary || data;
  if (!sum) return;

  const total = data.records_valid || sum.total_records || 119186;
  const elEv = document.getElementById('kpiEventos');
  const elRec = document.getElementById('tbRecords');
  const elNodos = document.getElementById('kpiNodos');
  const elScore = document.getElementById('kpiScore');
  const elRm = document.getElementById('rmRecords');
  const elDesc = document.getElementById('analysisDesc');

  if (elEv) elEv.textContent = fmtNum(total);
  if (elRec) elRec.textContent = fmtNum(total) + ' registros validados';
  if (elRm) elRm.textContent = fmtNum(total);
  if (elNodos) {
    elNodos.textContent = (sum.active_nodes || 3) + ' / ' + (sum.total_nodes || 3);
  }
  if (elScore) elScore.textContent = (sum.security_score || 94.2).toFixed(1);
  if (elDesc && sum.user_types) {
    const ut = sum.user_types;
    elDesc.textContent =
      fmtNum(total) +
      ' eventos en 30 días. ' +
      (ut.residentes_pct || 73) +
      '% residentes · ' +
      (ut.visitantes_pct || 18) +
      '% visitantes · ' +
      (ut.domicilios_pct || 9) +
      '% domicilios.';
  }

  CIRState.summary = sum;
}

function renderAlerts(alerts) {
  const feed = document.getElementById('alertFeed');
  if (!feed || !alerts || !alerts.length) return;
  feed.innerHTML = alerts
    .slice(0, 8)
    .map(
      (a) =>
        '<div class="alert-row ' +
        sevClass(a.severity) +
        '"><div class="al-dot"></div><div><div class="al-text">' +
        escapeHtml(a.text) +
        '</div><div class="al-time">' +
        timeAgo(a.time) +
        ' · ' +
        escapeHtml(a.source || 'C2') +
        '</div></div></div>'
    )
    .join('');
}

function renderPulse(rows) {
  const tbody = document.getElementById('pulseBody');
  if (!tbody || !rows || !rows.length) return;
  tbody.innerHTML = rows
    .map((r) => {
      const cls = r.alert ? ' class="row-alert"' : '';
      return (
        '<tr' +
        cls +
        '><td>' +
        escapeHtml(r.fecha_hora) +
        '</td><td>' +
        escapeHtml(r.torre_apto) +
        '</td><td><span class="pbadge ' +
        tipoBadge(r.tipo) +
        '">' +
        escapeHtml(r.tipo) +
        '</span></td><td class="pid">' +
        escapeHtml(r.identidad) +
        '</td><td class="' +
        estadoClass(r.estado) +
        '">' +
        escapeHtml(r.estado) +
        '</td></tr>'
      );
    })
    .join('');
}

function escapeHtml(s) {
  return String(s || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

function updateChartsFromSummary(sum) {
  if (!sum || !window.Chart) return;

  const labels = (sum.hourly_baseline || []).map((h) =>
    String(h.hour).padStart(2, '0') + 'h'
  );
  const baseline = (sum.hourly_baseline || []).map((h) => h.count);
  const today = (sum.hourly_today || []).map((h) => h.count);

  if (CIRState.charts.baseline) {
    CIRState.charts.baseline.data.labels = labels;
    CIRState.charts.baseline.data.datasets[0].data = baseline;
    CIRState.charts.baseline.data.datasets[1].data = today;
    CIRState.charts.baseline.update();
  }

  const ut = sum.user_types || {};
  if (CIRState.charts.type) {
    CIRState.charts.type.data.datasets[0].data = [
      ut.residentes_pct || 73,
      ut.visitantes_pct || 18,
      ut.domicilios_pct || 9,
    ];
    CIRState.charts.type.update();
  }

  const sc = sum.sensor_counts || {};
  if (CIRState.charts.sensor) {
    CIRState.charts.sensor.data.datasets[0].data = [
      sc.Movimiento || 24,
      sc.Temperatura || 12,
      sc.Consumo || 8,
      sc['Cámaras'] || 16,
      sc.Acceso || 6,
    ];
    CIRState.charts.sensor.update();
  }

  const vp = sum.vuln_patterns || {};
  if (CIRState.charts.vuln) {
    CIRState.charts.vuln.data.datasets[0].data = [
      vp.Phishing || 72,
      vp['API exploit'] || 58,
      vp.Inyección || 44,
      vp['Fuerza bruta'] || 35,
      vp.MitM || 20,
      vp.DoS || 48,
    ];
    CIRState.charts.vuln.update();
  }

  if (CIRState.charts.sec && sum.sec_events_30d) {
    CIRState.charts.sec.data.datasets[0].data = sum.sec_events_30d;
    CIRState.charts.sec.update();
  }

  const wr = sum.weekly_report || [];
  if (CIRState.charts.report && wr.length) {
    CIRState.charts.report.data.datasets[0].data = wr.map((w) => w.residentes);
    CIRState.charts.report.data.datasets[1].data = wr.map((w) => w.visitantes);
    CIRState.charts.report.data.datasets[2].data = wr.map((w) => w.domicilios);
    CIRState.charts.report.update();
  }
}

function registerCharts() {
  const bc = document.getElementById('baselineChart');
  if (bc && window.Chart && !CIRState.charts.baseline) {
    CIRState.charts.baseline = Chart.getChart(bc);
  }
  const tc = document.getElementById('typeChart');
  if (tc && window.Chart && !CIRState.charts.type) {
    CIRState.charts.type = Chart.getChart(tc);
  }
  const sc = document.getElementById('sensorChart');
  if (sc && window.Chart && !CIRState.charts.sensor) {
    CIRState.charts.sensor = Chart.getChart(sc);
  }
  const vc = document.getElementById('vulnChart');
  if (vc && window.Chart && !CIRState.charts.vuln) {
    CIRState.charts.vuln = Chart.getChart(vc);
  }
  const sec = document.getElementById('secChart');
  if (sec && window.Chart && !CIRState.charts.sec) {
    CIRState.charts.sec = Chart.getChart(sec);
  }
  const rc = document.getElementById('reportChart');
  if (rc && window.Chart && !CIRState.charts.report) {
    CIRState.charts.report = Chart.getChart(rc);
  }
}

async function cirRefresh() {
  try {
    await cirEnsureAuth();
    const [summary, alerts, pulse] = await Promise.all([
      cirRequest(CIR_ROUTES.summary()),
      cirRequest(CIR_ROUTES.alerts()),
      cirRequest(CIR_ROUTES.pulse()),
    ]);
    updateKPIs(summary);
    renderAlerts(alerts.alerts);
    renderPulse(pulse.pulse);
    registerCharts();
    updateChartsFromSummary(summary.summary);
  } catch (e) {
    console.warn('CIR refresh:', e.message);
    const badge = document.querySelector('.live-badge');
    if (badge) {
      badge.innerHTML =
        '<div class="live-dot" style="background:var(--amber)"></div>Reconectando…';
    }
  }
}

async function cirSimulateAttack(full) {
  try {
    const data = await cirRequest(CIR_ROUTES.simulateAttack(full), {
      auth: false,
      method: 'POST',
      attack: true,
    });
    const kpi = document.getElementById('kpiAlertas');
    if (kpi && data.kpi_alerts) kpi.textContent = data.kpi_alerts;
    const feed = document.getElementById('alertFeed');
    if (feed && data.alert_text) {
      const el = document.createElement('div');
      el.className = 'alert-row al-crit';
      el.innerHTML =
        '<div class="al-dot"></div><div><div class="al-text">' +
        escapeHtml(data.alert_text) +
        '</div><div class="al-time">ahora · Motor defensa C2</div></div>';
      feed.insertBefore(el, feed.firstChild);
    }
    if (full) {
      const step4 = document.getElementById('step4');
      if (step4) step4.style.display = 'flex';
      const b = document.getElementById('simBtn');
      if (b) {
        b.innerHTML =
          '<i class="ti ti-shield-check"></i>Simulación completada — defensa activada';
        b.className = 'run-btn rb-ok';
      }
    }
    return data;
  } catch (e) {
    console.error('Simulate attack:', e);
  }
}

window.simulateAttack = function () {
  cirSimulateAttack(false);
};

window.runSim = function () {
  cirSimulateAttack(true);
};

function cirStartPolling() {
  if (CIRState.refreshTimer) clearInterval(CIRState.refreshTimer);
  cirRefresh();
  CIRState.refreshTimer = setInterval(cirRefresh, 15000);
}

document.addEventListener('DOMContentLoaded', function () {
  setTimeout(function () {
    cirStartPolling();
  }, 500);
});

// Re-registrar charts tras init de pestañas
const _origSw = window.sw;
window.sw = function (id, btn) {
  if (_origSw) _origSw(id, btn);
  setTimeout(function () {
    registerCharts();
    if (CIRState.summary) updateChartsFromSummary(CIRState.summary);
  }, 300);
};
