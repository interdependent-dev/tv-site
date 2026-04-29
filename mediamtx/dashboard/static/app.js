/* INTERDEPENDENT // MISSION CONTROL — frontend */

const $ = (id) => document.getElementById(id);
const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

// ── helpers ──────────────────────────────────────────────────────
const fmtBytes = (n) => {
  if (n == null) return '—';
  const u = ['B', 'KB', 'MB', 'GB', 'TB'];
  let i = 0; let v = Number(n);
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
  return v.toFixed(v < 10 ? 2 : v < 100 ? 1 : 0) + ' ' + u[i];
};
const fmtBitrate = (bytesPerSec) => {
  if (bytesPerSec == null) return '—';
  const bps = bytesPerSec * 8;
  if (bps < 1e3) return bps.toFixed(0) + ' bps';
  if (bps < 1e6) return (bps / 1e3).toFixed(1) + ' kbps';
  if (bps < 1e9) return (bps / 1e6).toFixed(2) + ' Mbps';
  return (bps / 1e9).toFixed(2) + ' Gbps';
};
const fmtDuration = (ms) => {
  if (!ms || ms < 0) return '—';
  const s = Math.floor(ms / 1000);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  if (h > 0) return `${h}h ${m}m ${sec}s`;
  if (m > 0) return `${m}m ${sec}s`;
  return `${sec}s`;
};
const fmtAge = (iso) => {
  if (!iso) return '';
  const t = new Date(iso).getTime();
  if (!t) return '';
  return fmtDuration(Date.now() - t) + ' ago';
};
const setText = (id, text) => { const el = $(id); if (el) el.textContent = text; };
const setBar = (id, pct, warn = 75, err = 90) => {
  const el = $(id); if (!el) return;
  el.style.width = Math.max(0, Math.min(100, pct)).toFixed(1) + '%';
  el.classList.toggle('warn', pct >= warn && pct < err);
  el.classList.toggle('err', pct >= err);
};
const lightDot = (id, state) => {
  const el = $(id); if (!el) return;
  el.classList.remove('ok', 'warn', 'err');
  if (state) el.classList.add(state);
};
const showSnack = (msg, ok = true) => {
  const el = $('snack');
  el.textContent = msg;
  el.classList.add('show');
  el.style.color = ok ? 'var(--accent)' : 'var(--err)';
  clearTimeout(showSnack._t);
  showSnack._t = setTimeout(() => el.classList.remove('show'), 2400);
};

// ── clock ────────────────────────────────────────────────────────
function tickClock() {
  const d = new Date();
  const pad = (n) => String(n).padStart(2, '0');
  setText('clock', `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`);
}
setInterval(tickClock, 1000); tickClock();

// ── bitrate sliding window ───────────────────────────────────────
const bitrateState = { lastRX: null, lastTX: null, lastTS: null };
function updateBitrate(rx, tx) {
  const now = Date.now();
  if (bitrateState.lastTS != null && now > bitrateState.lastTS) {
    const dt = (now - bitrateState.lastTS) / 1000;
    const drx = Math.max(0, rx - bitrateState.lastRX) / dt;
    const dtx = Math.max(0, tx - bitrateState.lastTX) / dt;
    setText('hero-rx', fmtBitrate(drx));
    setText('hero-tx', fmtBitrate(dtx));
    setText('n-brin', fmtBitrate(drx));
    setText('n-brout', fmtBitrate(dtx));
  }
  bitrateState.lastRX = rx;
  bitrateState.lastTX = tx;
  bitrateState.lastTS = now;
}

// ── data fetchers ────────────────────────────────────────────────
async function getJSON(url) {
  try {
    const r = await fetch(url, { cache: 'no-store' });
    if (!r.ok) return null;
    return await r.json();
  } catch (e) { return null; }
}

let cfg = null;

async function loadConfig() {
  cfg = await getJSON('/api/config') || {};
}

// ── panels ───────────────────────────────────────────────────────
async function refreshStatus() {
  const s = await getJSON('/api/status');
  if (!s) {
    setText('hero-status', 'API DOWN');
    lightDot('h-mediamtx', 'err');
    setText('h-mediamtx-sub', 'unreachable');
    return;
  }
  // ON AIR badge
  const badge = $('onair-badge');
  badge.classList.toggle('live', !!s.onAir);
  setText('onair-text', s.onAir ? 'ON AIR' : 'OFFLINE');

  setText('hero-status', s.onAir ? 'BROADCASTING' : 'STANDBY');
  setText('hero-pub', s.publishers ?? 0);
  setText('hero-readers', s.readers ?? 0);
  setText('stream-hint', s.onAir ? `${s.publishers} publisher · ${s.readers} viewer(s)` : 'No publisher detected');

  // health
  lightDot('h-mediamtx', s.mediamtxOK && s.launchdMtx ? 'ok' : (s.mediamtxOK ? 'warn' : 'err'));
  setText('h-mediamtx-sub', s.mediamtxOK ? (s.launchdMtx ? 'launchd · running' : 'API ok · no launchd') : 'API offline');
  lightDot('h-cloudflared', s.launchdTun ? 'ok' : 'warn');
  setText('h-cloudflared-sub', s.launchdTun ? 'launchd · running' : 'not running via launchd');
  lightDot('h-api', s.mediamtxOK ? 'ok' : 'err');

  // primary path stats
  const primary = (s.paths || []).find((p) => p.name === (cfg?.autoOpenPath || 'program')) || (s.paths || [])[0];
  if (primary) {
    setText('n-path', primary.name);
    const src = primary.source ? (primary.source.type || '—') : '—';
    setText('n-src', src);
    setText('n-ready', primary.ready ? 'YES' : 'NO');
    setText('n-rdytime', primary.readyTime || '—');
    setText('n-tracks', (primary.tracks || []).join(', ') || '—');
    setText('n-rx', fmtBytes(primary.bytesReceived));
    setText('n-tx', fmtBytes(primary.bytesSent));
    if (primary.ready) {
      const upMs = primary.readyTime ? Date.now() - new Date(primary.readyTime).getTime() : null;
      setText('hero-uptime', upMs ? fmtDuration(upMs) : '—');
    } else {
      setText('hero-uptime', '—');
    }
    updateBitrate(primary.bytesReceived || 0, primary.bytesSent || 0);
  } else {
    ['n-path', 'n-src', 'n-ready', 'n-rdytime', 'n-tracks', 'n-rx', 'n-tx'].forEach((id) => setText(id, '—'));
    setText('hero-uptime', '—');
  }

  setText('n-srt', (s.srtConns || []).length);
  setText('n-hlsmux', (s.hlsMuxers || []).length);
  setText('n-webrtc', (s.webrtcSess || []).length);

  // connections table
  renderConnections(s);
}

function renderConnections(s) {
  const tbody = document.querySelector('#conns-tbl tbody');
  tbody.innerHTML = '';
  let total = 0;

  (s.srtConns || []).forEach((c) => {
    total++;
    tbody.insertAdjacentHTML('beforeend', `
      <tr><td class="proto-srt">SRT</td><td>${c.path || '—'}</td><td>${c.remoteAddr || '—'}</td>
      <td>${c.state || '—'}</td><td>${fmtBytes(c.bytesReceived)}</td><td>${fmtBytes(c.bytesSent)}</td>
      <td>${fmtAge(c.created)}</td></tr>`);
  });
  (s.hlsMuxers || []).forEach((c) => {
    total++;
    tbody.insertAdjacentHTML('beforeend', `
      <tr><td class="proto-hls">HLS</td><td>${c.path || '—'}</td><td>—</td>
      <td>active</td><td>—</td><td>${fmtBytes(c.bytesSent)}</td>
      <td>${fmtAge(c.created)}</td></tr>`);
  });
  (s.webrtcSess || []).forEach((c) => {
    total++;
    tbody.insertAdjacentHTML('beforeend', `
      <tr><td class="proto-webrtc">WebRTC</td><td>${c.path || '—'}</td><td>${c.remoteAddr || '—'}</td>
      <td>${c.state || '—'}</td><td>${fmtBytes(c.bytesReceived)}</td><td>${fmtBytes(c.bytesSent)}</td>
      <td>${fmtAge(c.created)}</td></tr>`);
  });

  if (total === 0) {
    tbody.insertAdjacentHTML('beforeend', `<tr><td class="empty" colspan="7">— no active sessions —</td></tr>`);
  }
  setText('conn-hint', `${total} session${total === 1 ? '' : 's'}`);
}

async function refreshSystem() {
  const s = await getJSON('/api/system');
  if (!s) return;
  setText('s-host', s.hostname || '—');
  setText('s-os', `${s.os || ''} / ${s.arch || ''} · ${s.numCPU} cores`);
  const cpuUsed = (s.cpuUserPct || 0) + (s.cpuSysPct || 0);
  setText('s-cpu', `${cpuUsed.toFixed(1)}% used  (${(s.cpuIdlePct || 0).toFixed(1)}% idle)`);
  setBar('s-cpu-bar', cpuUsed);
  setText('s-load', (s.loadAvg || []).map((v) => v.toFixed(2)).join('  '));
  const memPct = s.memTotalBytes ? (100 * s.memUsedBytes / s.memTotalBytes) : 0;
  setText('s-mem', `${fmtBytes(s.memUsedBytes)} / ${fmtBytes(s.memTotalBytes)} (${memPct.toFixed(0)}%)`);
  setBar('s-mem-bar', memPct);
  const dpct = s.disk?.totalBytes ? (100 * s.disk.usedBytes / s.disk.totalBytes) : 0;
  setText('s-disk', `${fmtBytes(s.disk?.usedBytes)} / ${fmtBytes(s.disk?.totalBytes)} (${dpct.toFixed(0)}%)`);
  setBar('s-disk-bar', dpct);
  setText('s-uptime', s.uptime || '—');
}

async function refreshNetwork() {
  const n = await getJSON('/api/network');
  if (!n) return;
  setText('net-lan', (n.lanIPs || []).join(', ') || '—');
  setText('net-public', n.publicIP || '—');

  const ifaces = (n.interfaces || []).filter((i) => i.up && (i.addrs || []).some((a) => /^\d+\.\d+\.\d+\.\d+/.test(a)));
  const primary = ifaces[0];
  if (primary) {
    setText('net-iface', primary.name);
    setText('net-link', primary.media || '—');
    setText('net-mtu', primary.mtu);
  }
  const dns = n.dnsLookup || {};
  setText('net-dns', dns.error ? `error: ${dns.error}` : `${dns.host} → ${(dns.addrs || []).join(', ')} (${dns.ms}ms)`);

  const pingsEl = $('net-pings'); pingsEl.innerHTML = '';
  (n.pings || []).forEach((p) => {
    if (!p.target) return;
    pingsEl.insertAdjacentHTML('beforeend',
      `<li><span class="dot ${p.ok ? 'ok' : 'err'}"></span><span>${p.target}</span><span class="v">${p.ok ? p.latencyMs.toFixed(1) + ' ms' : 'unreachable'}${p.loss ? ' · ' + p.loss + '% loss' : ''}</span></li>`);
  });

  const portsEl = $('net-ports'); portsEl.innerHTML = '';
  (n.ports || []).forEach((p) => {
    const proto = p.proto || 'tcp';
    const detail = p.open
      ? (proto === 'udp' ? 'listening · udp' : 'open · ' + p.latencyMs + ' ms')
      : (proto === 'udp' ? 'no listener · udp' : 'closed');
    portsEl.insertAdjacentHTML('beforeend',
      `<li><span class="dot ${p.open ? 'ok' : 'err'}"></span><span>${p.name} <span style="color:var(--txt-mute)">${p.host}:${p.port}/${proto}</span></span><span class="v">${detail}</span></li>`);
  });

  // health side panel update
  const publicReachable = (n.pings || []).find((p) => p.target === cfg?.publicHost);
  lightDot('h-public', publicReachable?.ok ? 'ok' : 'err');
  setText('h-public-sub', cfg?.publicHost || '—');
}

async function refreshCloudflare() {
  const c = await getJSON('/api/cloudflare');
  if (!c) return;
  setText('cf-platform', c.platformStatus || '—');
  if (c.platformIndicator && c.platformIndicator !== 'none') {
    document.getElementById('cf-platform').style.color = c.platformIndicator === 'critical' ? 'var(--err)' : 'var(--warn)';
  }
  setText('cf-zone', c.zoneError ? `error: ${c.zoneError}` : `${c.zoneName || '—'} ${c.zoneStatus ? '· ' + c.zoneStatus : ''}`);
  setText('cf-req', c.requests24h != null ? c.requests24h.toLocaleString() : (c.tokenSet ? 'no data' : 'set CF_API_TOKEN'));
  setText('cf-bw', c.bandwidth24h != null ? fmtBytes(c.bandwidth24h) : '—');
  setText('cf-cache', c.cacheHitPct != null ? c.cacheHitPct.toFixed(1) + '%' : '—');
  setText('cf-threat', c.threats24h ?? '—');
  if (c.tlsExpiry) {
    const t = new Date(c.tlsExpiry);
    const days = Math.floor((t.getTime() - Date.now()) / 86400000);
    setText('cf-tls', `${t.toISOString().slice(0, 10)} (${days}d)`);
    lightDot('h-tls', days > 14 ? 'ok' : days > 0 ? 'warn' : 'err');
    setText('h-tls-sub', `${days}d remaining`);
  } else if (c.tlsError) {
    setText('cf-tls', 'error');
    lightDot('h-tls', 'err');
    setText('h-tls-sub', c.tlsError);
  }
  setText('cf-issuer', c.tlsIssuer || '—');
  setText('cf-hint', c.tokenSet ? 'API token: set' : 'CF_API_TOKEN not set — public status only');

  const ul = $('cf-incidents'); ul.innerHTML = '';
  if (!c.incidents || c.incidents.length === 0) {
    ul.innerHTML = '<li class="muted">No active incidents.</li>';
  } else {
    c.incidents.forEach((inc) => {
      ul.insertAdjacentHTML('beforeend',
        `<li class="${inc.impact === 'critical' || inc.impact === 'major' ? 'major' : ''}">
        <strong>${inc.name}</strong> — ${inc.status} (${inc.impact})
        ${inc.shortlink ? ` <a href="${inc.shortlink}" target="_blank">↗</a>` : ''}</li>`);
    });
  }
}

async function refreshStreamHealth() {
  const r = await getJSON('/api/health/stream');
  if (!r) return;
  const overall = $('sh-overall');
  overall.classList.remove('ok', 'warn', 'err');
  overall.classList.add(r.overall || 'ok');
  setText('sh-text',
    r.overall === 'ok' ? '✓ All systems nominal' :
    r.overall === 'warn' ? '⚠ Degraded — see issues below' :
    '✕ Errors detected');

  const ul = $('sh-incidents');
  ul.innerHTML = '';
  if (!r.incidents || r.incidents.length === 0) {
    ul.innerHTML = '<li class="muted">No issues in the last 15 min.</li>';
    setText('sh-hint', 'looking back 15 min');
    return;
  }
  setText('sh-hint', `${r.incidents.length} active issue${r.incidents.length === 1 ? '' : 's'}`);
  r.incidents.forEach((inc) => {
    const ago = fmtAge(inc.lastSeen);
    ul.insertAdjacentHTML('beforeend',
      `<li class="${inc.severity || 'info'}">
        <div class="ihead">
          <div><span class="icat">${escapeHTML(inc.category || '')}</span><span class="ititle">${escapeHTML(inc.title || '')}</span></div>
          <span class="icount">×${inc.count} · ${ago}</span>
        </div>
        ${inc.detail ? `<div class="idetail">${escapeHTML(inc.detail)}</div>` : ''}
        ${inc.remediation ? `<div class="iremedy">${escapeHTML(inc.remediation)}</div>` : ''}
      </li>`);
  });
}

async function refreshGitHub() {
  const g = await getJSON('/api/github');
  if (!g) return;
  setText('gh-hint', g.repo || '');
  const ul = $('gh-runs'); ul.innerHTML = '';
  if (g.error) {
    ul.innerHTML = `<li class="muted">${g.error}</li>`;
    return;
  }
  if (!g.runs || g.runs.length === 0) {
    ul.innerHTML = '<li class="muted">No runs.</li>';
    return;
  }
  g.runs.forEach((r) => {
    let cls = 'warn';
    if (r.status === 'in_progress' || r.status === 'queued') cls = 'run';
    else if (r.conclusion === 'success') cls = 'ok';
    else if (r.conclusion === 'failure' || r.conclusion === 'cancelled') cls = 'err';
    ul.insertAdjacentHTML('beforeend',
      `<li>
        <span class="dot ${cls}"></span>
        <div class="title">
          <span class="t1">${escapeHTML(r.displayTitle || r.name || '—')}</span>
          <span class="t2">${r.event || ''} · ${r.headBranch || ''} · ${(r.headSha || '').slice(0, 7)}${r.url ? ` · <a href="${r.url}" target="_blank">↗ open</a>` : ''}</span>
        </div>
        <span class="age">${fmtAge(r.updatedAt || r.createdAt)}</span>
      </li>`);
  });
}

function escapeHTML(s) {
  return String(s).replace(/[&<>"']/g, (c) => ({ '&':'&amp;', '<':'&lt;', '>':'&gt;', '"':'&quot;', "'":'&#39;' }[c]));
}

// ── Logs (SSE) ───────────────────────────────────────────────────
const logState = {
  paused: false,
  filterText: '',
  sources: { mediamtx: true, cloudflared: true, dashboard: false },
  levels: { ERR: true, WRN: true, INF: true, DBG: false },
  lines: [],
};

function shouldShow(line) {
  if (!logState.sources[line.source]) return false;
  if (!logState.levels[line.level]) return false;
  if (logState.filterText && !line.text.toLowerCase().includes(logState.filterText)) return false;
  return true;
}

function appendLogLine(line) {
  if (!shouldShow(line)) return;
  const stream = $('log-stream');
  const wasAtBottom = stream.scrollHeight - stream.scrollTop - stream.clientHeight < 40;
  const t = new Date(line.time);
  const pad = (n) => String(n).padStart(2, '0');
  const ts = `${pad(t.getHours())}:${pad(t.getMinutes())}:${pad(t.getSeconds())}`;
  const div = document.createElement('div');
  div.className = `log-line lvl-${line.level}`;
  div.innerHTML = `<span class="lt">${ts}</span><span class="ll">${line.level}</span><span class="ls ${line.source}">${line.source}</span><span class="lx">${escapeHTML(line.text)}</span>`;
  stream.appendChild(div);
  while (stream.children.length > 1500) stream.removeChild(stream.firstChild);
  if (wasAtBottom && !logState.paused) stream.scrollTop = stream.scrollHeight;
}

function rerenderLogs() {
  const stream = $('log-stream');
  stream.innerHTML = '';
  logState.lines.forEach(appendLogLine);
}

async function loadLogHistory() {
  const lines = await getJSON('/api/logs/history') || [];
  lines.sort((a, b) => new Date(a.time) - new Date(b.time));
  logState.lines = lines;
  rerenderLogs();
}

function startLogStream() {
  const es = new EventSource('/api/logs/stream');
  es.onopen = () => setText('footer-conn', 'logs · connected');
  es.onerror = () => setText('footer-conn', 'logs · reconnecting…');
  es.onmessage = (ev) => {
    if (logState.paused) return;
    try {
      const line = JSON.parse(ev.data);
      logState.lines.push(line);
      if (logState.lines.length > 2000) logState.lines.shift();
      appendLogLine(line);
    } catch (e) {}
  };
}

// ── Actions ──────────────────────────────────────────────────────
async function triggerAction(name) {
  showSnack(`running: ${name}…`);
  const r = await fetch(`/api/action/${name}`, { method: 'POST' });
  const j = await r.json().catch(() => ({}));
  showSnack(j.ok ? `✓ ${name}` : `✗ ${name}: ${j.error || 'failed'}`, j.ok);
}

function setupUI() {
  $$('.act[data-action]').forEach((b) => {
    b.addEventListener('click', () => triggerAction(b.dataset.action));
  });

  $('act-copy-obs').addEventListener('click', async () => {
    const ip = $('net-lan').textContent.split(',')[0].trim() || '127.0.0.1';
    const url = `srt://${ip}:8890?streamid=publish:program&pkt_size=1316&latency=200000`;
    try { await navigator.clipboard.writeText(url); showSnack('OBS URL copied'); }
    catch { showSnack('copy failed: ' + url, false); }
  });
  $('act-copy-public').addEventListener('click', async () => {
    const url = `https://${cfg?.publicHost || 'live.interdependent.dev'}/program/`;
    try { await navigator.clipboard.writeText(url); showSnack('Public URL copied'); }
    catch { showSnack('copy failed: ' + url, false); }
  });

  // Log toolbar
  $$('.log-toolbar input[data-src]').forEach((cb) => {
    logState.sources[cb.dataset.src] = cb.checked;
    cb.addEventListener('change', () => { logState.sources[cb.dataset.src] = cb.checked; rerenderLogs(); });
  });
  $$('.log-toolbar input[data-level]').forEach((cb) => {
    logState.levels[cb.dataset.level] = cb.checked;
    cb.addEventListener('change', () => { logState.levels[cb.dataset.level] = cb.checked; rerenderLogs(); });
  });
  $('log-search').addEventListener('input', (e) => { logState.filterText = e.target.value.toLowerCase(); rerenderLogs(); });
  $('log-pause').addEventListener('click', () => {
    logState.paused = !logState.paused;
    $('log-pause').classList.toggle('paused', logState.paused);
    $('log-pause').textContent = logState.paused ? 'RESUME' : 'PAUSE';
  });
  $('log-clear').addEventListener('click', () => { logState.lines = []; $('log-stream').innerHTML = ''; });
}

// ── Boot ─────────────────────────────────────────────────────────
async function boot() {
  await loadConfig();
  setupUI();
  await Promise.all([refreshStatus(), refreshSystem(), refreshNetwork(), refreshCloudflare(), refreshGitHub(), refreshStreamHealth(), loadLogHistory()]);
  startLogStream();

  setInterval(refreshStatus, 2000);
  setInterval(refreshStreamHealth, 4000);
  setInterval(refreshSystem, 5000);
  setInterval(refreshNetwork, 10000);
  setInterval(refreshCloudflare, 60000);
  setInterval(refreshGitHub, 30000);
}

boot();
