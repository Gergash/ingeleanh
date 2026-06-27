'use strict';

(function () {
  const c = document.getElementById('bgCanvas');
  if (!c) return;
  const ctx = c.getContext('2d');
  let W, H, stars = [];
  function resize() {
    W = c.width = innerWidth;
    H = c.height = innerHeight;
    stars = [];
  }
  function mkStar() {
    return {
      x: Math.random() * W,
      y: Math.random() * H,
      r: Math.random() * 0.9 + 0.1,
      a: Math.random(),
      da: Math.random() * 0.004 + 0.001,
      v: Math.random() * 0.08,
    };
  }
  function init() {
    resize();
    for (let i = 0; i < 160; i++) stars.push(mkStar());
  }
  function draw() {
    ctx.clearRect(0, 0, W, H);
    stars.forEach((s) => {
      s.a += s.da;
      if (s.a > 1 || s.a < 0) s.da *= -1;
      s.x -= s.v;
      if (s.x < 0) s.x = W;
      ctx.beginPath();
      ctx.arc(s.x, s.y, s.r, 0, Math.PI * 2);
      ctx.fillStyle = 'rgba(148,163,184,' + s.a * 0.6 + ')';
      ctx.fill();
    });
    requestAnimationFrame(draw);
  }
  window.addEventListener('resize', init);
  init();
  draw();
})();

(function () {
  const canvas = document.getElementById('netCanvas');
  if (!canvas) return;
  const ctx = canvas.getContext('2d');
  let W, H;
  const nodes = [
    { id: 'C2', label: 'C2', x: 0.75, y: 0.22, r: 14, color: '#a78bfa', glow: 'rgba(167,139,250,0.5)', type: 'c2' },
    { id: 'AGG1', label: 'Agregador\nMedellín', x: 0.48, y: 0.45, r: 9, color: '#60a5fa', glow: 'rgba(96,165,250,0.4)', type: 'agg' },
    { id: 'AGG2', label: 'Agregador\nVigil.', x: 0.62, y: 0.72, r: 9, color: '#60a5fa', glow: 'rgba(96,165,250,0.4)', type: 'agg' },
    { id: 'N1', label: 'Laureles', x: 0.15, y: 0.35, r: 7, color: '#22d3ee', glow: 'rgba(34,211,238,0.4)', type: 'src' },
    { id: 'N2', label: 'Poblado', x: 0.28, y: 0.62, r: 7, color: '#22d3ee', glow: 'rgba(34,211,238,0.4)', type: 'src' },
    { id: 'N3', label: 'Envigado', x: 0.36, y: 0.25, r: 7, color: '#22d3ee', glow: 'rgba(34,211,238,0.4)', type: 'src' },
    { id: 'N4', label: 'Peaje\nTrv45', x: 0.55, y: 0.82, r: 7, color: '#f472b6', glow: 'rgba(244,114,182,0.4)', type: 'src' },
    { id: 'N5', label: 'Alcaldía', x: 0.78, y: 0.6, r: 7, color: '#fbbf24', glow: 'rgba(251,191,36,0.4)', type: 'src' },
    { id: 'N6', label: 'Itaguí', x: 0.18, y: 0.75, r: 6, color: '#22d3ee', glow: 'rgba(34,211,238,0.3)', type: 'src' },
  ];
  const edges = [
    ['AGG1', 'C2'], ['AGG2', 'C2'],
    ['N1', 'AGG1'], ['N2', 'AGG1'], ['N3', 'AGG1'],
    ['N4', 'AGG2'], ['N5', 'AGG2'], ['N6', 'AGG1'],
  ];
  let t = 0;
  const pulses = [];
  function addPulse() {
    const e = edges[Math.floor(Math.random() * edges.length)];
    const src = nodes.find((n) => n.id === e[0]);
    const dst = nodes.find((n) => n.id === e[1]);
    if (src && dst) pulses.push({ src, dst, t: 0, speed: 0.015 + Math.random() * 0.01 });
  }
  setInterval(addPulse, 600);
  function resize() {
    W = canvas.offsetWidth;
    H = canvas.offsetHeight;
    canvas.width = W;
    canvas.height = H;
  }
  function px(n) {
    return { x: n.x * W, y: n.y * H };
  }
  function draw() {
    ctx.clearRect(0, 0, W, H);
    edges.forEach(([a, b]) => {
      const na = nodes.find((n) => n.id === a);
      const nb = nodes.find((n) => n.id === b);
      const pa = px(na);
      const pb = px(nb);
      ctx.beginPath();
      ctx.moveTo(pa.x, pa.y);
      ctx.lineTo(pb.x, pb.y);
      ctx.strokeStyle = 'rgba(255,255,255,0.06)';
      ctx.lineWidth = 1;
      ctx.stroke();
    });
    for (let i = pulses.length - 1; i >= 0; i--) {
      const p = pulses[i];
      p.t += p.speed;
      if (p.t > 1) {
        pulses.splice(i, 1);
        continue;
      }
      const pa = px(p.src);
      const pb = px(p.dst);
      const x = pa.x + (pb.x - pa.x) * p.t;
      const y = pa.y + (pb.y - pa.y) * p.t;
      const g = ctx.createRadialGradient(x, y, 0, x, y, 5);
      g.addColorStop(0, 'rgba(167,139,250,0.9)');
      g.addColorStop(1, 'rgba(167,139,250,0)');
      ctx.beginPath();
      ctx.arc(x, y, 5, 0, Math.PI * 2);
      ctx.fillStyle = g;
      ctx.fill();
    }
    nodes.forEach((n) => {
      const { x, y } = px(n);
      const bob = n.type === 'c2' ? Math.sin(t * 0.8) * 3 : 0;
      const g = ctx.createRadialGradient(x, y + bob, 0, x, y + bob, n.r * 3.5);
      g.addColorStop(0, n.glow);
      g.addColorStop(1, 'transparent');
      ctx.beginPath();
      ctx.arc(x, y + bob, n.r * 3.5, 0, Math.PI * 2);
      ctx.fillStyle = g;
      ctx.fill();
      const bg = ctx.createRadialGradient(x - n.r * 0.3, y + bob - n.r * 0.3, 0, x, y + bob, n.r);
      bg.addColorStop(0, '#fff');
      bg.addColorStop(0.4, n.color);
      bg.addColorStop(1, 'rgba(0,0,0,0.5)');
      ctx.beginPath();
      ctx.arc(x, y + bob, n.r, 0, Math.PI * 2);
      ctx.fillStyle = bg;
      ctx.fill();
      if (n.type === 'c2') {
        ctx.beginPath();
        ctx.arc(x, y + bob, n.r + 5 + Math.sin(t) * 2, 0, Math.PI * 2);
        ctx.strokeStyle = 'rgba(167,139,250,' + (0.3 + Math.sin(t * 0.7) * 0.2) + ')';
        ctx.lineWidth = 1;
        ctx.stroke();
      }
      ctx.font = (n.type === 'c2' ? '700 ' : '500 ') + '9px -apple-system,sans-serif';
      ctx.fillStyle = n.type === 'c2' ? '#e2e8f0' : 'rgba(148,163,184,0.9)';
      ctx.textAlign = 'center';
      n.label.split('\n').forEach((line, i) => ctx.fillText(line, x, y + bob + n.r + 10 + i * 10));
    });
    t += 0.02;
    requestAnimationFrame(draw);
  }
  new ResizeObserver(resize).observe(canvas.parentElement);
  resize();
  draw();
})();

function clock() {
  const el = document.getElementById('tbTime');
  if (!el) return;
  el.textContent = new Date().toLocaleTimeString('es-CO', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}
clock();
setInterval(clock, 1000);

window.sw = function (id, btn) {
  document.querySelectorAll('.panel').forEach((p) => p.classList.remove('active'));
  document.querySelectorAll('.tab').forEach((t) => t.classList.remove('active'));
  document.getElementById('p-' + id).classList.add('active');
  btn.classList.add('active');
  if (id === 'predict') initPredict();
  if (id === 'iot') initSensor();
  if (id === 'cyber') initCyber();
  if (id === 'etl') initEtl();
};

const GRID = 'rgba(255,255,255,0.05)';
const TICK = { color: 'rgba(148,163,184,0.7)', font: { size: 10, family: '-apple-system,sans-serif' } };
if (typeof Chart !== 'undefined') {
  Chart.defaults.color = 'rgba(148,163,184,0.7)';
  Chart.defaults.borderColor = 'rgba(255,255,255,0.05)';
}
const baseOpts = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: { legend: { display: false } },
  scales: { x: { ticks: TICK, grid: { color: GRID } }, y: { ticks: TICK, grid: { color: GRID } } },
};

let pI = false, sI = false, cI = false, eI = false;

function initPredict() {
  if (pI || typeof Chart === 'undefined') return;
  pI = true;
  new Chart(document.getElementById('baselineChart'), {
    type: 'bar',
    data: {
      labels: ['06h', '07h', '08h', '09h', '10h', '11h', '12h', '13h', '14h', '15h', '16h', '17h', '18h', '19h'],
      datasets: [
        { label: 'Línea base', data: [12, 45, 88, 62, 34, 28, 20, 22, 18, 24, 30, 42, 85, 70], backgroundColor: 'rgba(96,165,250,0.2)', borderColor: 'rgba(96,165,250,0.4)', borderWidth: 1, borderRadius: 3 },
        { label: 'Hoy', data: [14, 52, 124, 71, 38, 25, 22, 20, 17, 26, 35, 48, 92, 78], backgroundColor: 'rgba(96,165,250,0.6)', borderColor: '#60a5fa', borderWidth: 1, borderRadius: 3 },
      ],
    },
    options: { ...baseOpts, plugins: { legend: { display: true, labels: { color: 'rgba(148,163,184,0.8)', boxWidth: 10, font: { size: 10 } } } } },
  });
  new Chart(document.getElementById('typeChart'), {
    type: 'doughnut',
    data: {
      labels: ['Residentes 73%', 'Visitantes 18%', 'Domicilios 9%'],
      datasets: [{ data: [73, 18, 9], backgroundColor: ['rgba(96,165,250,0.7)', 'rgba(167,139,250,0.7)', 'rgba(251,191,36,0.7)'], borderColor: ['#60a5fa', '#a78bfa', '#fbbf24'], borderWidth: 1.5 }],
    },
    options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { position: 'bottom', labels: { color: 'rgba(148,163,184,0.8)', padding: 12, font: { size: 10 } } } } },
  });
}

function initSensor() {
  if (sI || typeof Chart === 'undefined') return;
  sI = true;
  new Chart(document.getElementById('sensorChart'), {
    type: 'bar',
    data: {
      labels: ['Movimiento', 'Temperatura', 'Consumo', 'Cámaras', 'Acceso'],
      datasets: [{ data: [24, 12, 8, 16, 6], backgroundColor: ['rgba(96,165,250,0.5)', 'rgba(34,211,238,0.5)', 'rgba(251,191,36,0.5)', 'rgba(167,139,250,0.5)', 'rgba(248,113,113,0.5)'], borderColor: ['#60a5fa', '#22d3ee', '#fbbf24', '#a78bfa', '#f87171'], borderWidth: 1, borderRadius: 3 }],
    },
    options: { indexAxis: 'y', ...baseOpts },
  });
}

function initCyber() {
  if (cI || typeof Chart === 'undefined') return;
  cI = true;
  new Chart(document.getElementById('vulnChart'), {
    type: 'radar',
    data: {
      labels: ['Phishing', 'API exploit', 'Inyección', 'Fuerza bruta', 'MitM', 'DoS'],
      datasets: [{ label: 'Nivel riesgo', data: [72, 58, 44, 35, 20, 48], backgroundColor: 'rgba(248,113,113,0.12)', borderColor: '#f87171', pointBackgroundColor: '#f87171', pointRadius: 4, borderWidth: 1.5 }],
    },
    options: {
      responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } },
      scales: { r: { ticks: { color: 'rgba(148,163,184,0.6)', backdropColor: 'transparent', stepSize: 20 }, grid: { color: 'rgba(255,255,255,0.06)' }, pointLabels: { color: 'rgba(148,163,184,0.8)', font: { size: 10 } }, angleLines: { color: 'rgba(255,255,255,0.06)' } } },
    },
  });
  new Chart(document.getElementById('secChart'), {
    type: 'line',
    data: {
      labels: Array.from({ length: 30 }, (_, i) => 'D' + (i + 1)),
      datasets: [{ data: [2, 1, 3, 0, 2, 4, 1, 2, 5, 2, 1, 3, 2, 6, 4, 2, 3, 1, 2, 4, 3, 5, 2, 7, 3, 2, 4, 6, 5, 7], borderColor: '#f87171', backgroundColor: 'rgba(248,113,113,0.08)', borderWidth: 1.5, pointRadius: 0, fill: true, tension: 0.4 }],
    },
    options: { ...baseOpts, plugins: { legend: { display: false } }, scales: { x: { ticks: { ...TICK, maxTicksLimit: 8 }, grid: { color: GRID } }, y: { ticks: TICK, grid: { color: GRID } } } },
  });
}

function initEtl() {
  if (eI || typeof Chart === 'undefined') return;
  eI = true;
  new Chart(document.getElementById('reportChart'), {
    type: 'bar',
    data: {
      labels: ['Semana 1', 'Semana 2', 'Semana 3', 'Semana 4'],
      datasets: [
        { label: 'Residentes', data: [21840, 28920, 24480, 31620], backgroundColor: 'rgba(96,165,250,0.6)', borderColor: '#60a5fa', borderWidth: 1, borderRadius: 3 },
        { label: 'Visitantes', data: [5040, 6720, 5760, 7560], backgroundColor: 'rgba(167,139,250,0.6)', borderColor: '#a78bfa', borderWidth: 1, borderRadius: 3 },
        { label: 'Domicilios', data: [2520, 3360, 2880, 3780], backgroundColor: 'rgba(251,191,36,0.6)', borderColor: '#fbbf24', borderWidth: 1, borderRadius: 3 },
      ],
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      plugins: { legend: { display: true, labels: { color: 'rgba(148,163,184,0.8)', boxWidth: 10, font: { size: 10 } } } },
      scales: { x: { stacked: true, ticks: TICK, grid: { color: GRID } }, y: { stacked: true, ticks: TICK, grid: { color: GRID } } },
    },
  });
}

setTimeout(initPredict, 200);
