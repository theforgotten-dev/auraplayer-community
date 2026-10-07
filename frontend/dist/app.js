const $ = (selector) => document.querySelector(selector);

const ui = {
  open: $('#open-files'), queue: $('#queue'), queueCount: $('#queue-count'),
  title: $('#track-title'), codec: $('#codec-chip'), statusDot: $('#status-dot'),
  engineLabel: $('#engine-label'), visualizer: $('#visualizer'), seek: $('#seek'),
  position: $('#position'), duration: $('#duration'), volume: $('#volume'),
  volumeValue: $('#volume-value'), toggle: $('#toggle'), stop: $('#stop'),
  previous: $('#previous'), next: $('#next'), toast: $('#toast'),
  metaTitle: $('#meta-title'), metaCodec: $('#meta-codec'), metaRate: $('#meta-rate'),
  metaChannels: $('#meta-channels'), metaPath: $('#meta-path'),
  workspace: $('.workspace'), restore: $('#restore-panels')
};

let state = { state: 'stopped', queue: [], currentIndex: -1, position: 0, volume: .8, error: '' };
let selectedIndex = 0;
let seeking = false;
let toastTimer;

function backend() {
  return window.go?.main?.App;
}

async function invoke(method, ...args) {
  const api = backend();
  if (!api || typeof api[method] !== 'function') {
    throw new Error('El puente con Go todavía no está disponible.');
  }
  try {
    const nextState = await api[method](...args);
    if (nextState) render(nextState);
    return nextState;
  } catch (error) {
    showError(typeof error === 'string' ? error : error?.message || String(error));
    throw error;
  }
}

function render(nextState) {
  state = nextState;
  if (state.currentIndex >= 0) selectedIndex = state.currentIndex;
  const track = state.queue?.[state.currentIndex];
  ui.queueCount.textContent = state.queue?.length ?? 0;
  ui.title.textContent = track?.title || 'Nada seleccionado';
  ui.codec.textContent = track?.codec || '—';
  ui.metaTitle.textContent = track?.title || '—';
  ui.metaCodec.textContent = track?.codec || '—';
  ui.metaRate.textContent = track ? `${(track.sampleRate / 1000).toFixed(1)} kHz` : '—';
  ui.metaChannels.textContent = track ? (track.channels === 2 ? 'Estéreo' : 'Mono') : '—';
  ui.metaPath.textContent = track?.path || '—';
  ui.metaPath.title = track?.path || '';

  const duration = track?.duration || 0;
  if (!seeking) {
    ui.seek.max = duration;
    ui.seek.value = Math.min(state.position || 0, duration);
  }
  ui.position.textContent = formatTime(state.position || 0);
  ui.duration.textContent = formatTime(duration);
  ui.volume.value = state.volume;
  ui.volumeValue.textContent = `${Math.round(state.volume * 100)}%`;

  const playing = state.state === 'playing';
  ui.toggle.textContent = playing ? 'Ⅱ' : '▶';
  ui.toggle.setAttribute('aria-label', playing ? 'Pausar' : 'Reproducir');
  ui.visualizer.classList.toggle('playing', playing);
  ui.statusDot.className = `status-dot ${playing ? 'playing' : state.state === 'error' ? 'error' : ''}`;
  ui.engineLabel.textContent = statusLabel(state.state);
  renderQueue();

  if (state.error) showError(state.error);
}

function renderQueue() {
  if (!state.queue?.length) {
    ui.queue.innerHTML = `<div class="empty-state"><div class="empty-icon" aria-hidden="true">♫</div><strong>Tu cola está vacía</strong><span>Agrega uno o varios archivos MP3.</span></div>`;
    return;
  }
  ui.queue.innerHTML = '';
  state.queue.forEach((track, index) => {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = `queue-item ${index === state.currentIndex ? 'active' : ''}`;
    button.dataset.index = index;
    button.role = 'option';
    button.setAttribute('aria-selected', index === state.currentIndex ? 'true' : 'false');
    button.innerHTML = `<span class="queue-number">${String(index + 1).padStart(2, '0')}</span><span class="queue-title"></span><span class="queue-duration">${formatTime(track.duration)}</span>`;
    button.querySelector('.queue-title').textContent = track.title;
    button.addEventListener('click', () => playIndex(index));
    ui.queue.appendChild(button);
  });
}

async function playIndex(index) {
  selectedIndex = index;
  await invoke('Play', index);
}

function formatTime(seconds) {
  if (!Number.isFinite(seconds) || seconds < 0) return '0:00';
  const rounded = Math.floor(seconds);
  return `${Math.floor(rounded / 60)}:${String(rounded % 60).padStart(2, '0')}`;
}

function statusLabel(value) {
  return ({ playing: 'Reproduciendo', paused: 'En pausa', loading: 'Cargando', error: 'Revisar error', stopped: 'Motor listo' })[value] || 'Motor listo';
}

function showError(message) {
  if (!message) return;
  ui.toast.textContent = message;
  ui.toast.classList.add('visible');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => ui.toast.classList.remove('visible'), 4800);
}

ui.open.addEventListener('click', () => invoke('OpenFiles').catch(() => {}));
ui.toggle.addEventListener('click', () => invoke('Toggle').catch(() => {}));
ui.stop.addEventListener('click', () => invoke('Stop').catch(() => {}));
ui.previous.addEventListener('click', () => invoke('Previous').catch(() => {}));
ui.next.addEventListener('click', () => invoke('Next').catch(() => {}));

ui.seek.addEventListener('pointerdown', () => { seeking = true; });
ui.seek.addEventListener('input', () => { ui.position.textContent = formatTime(Number(ui.seek.value)); });
ui.seek.addEventListener('change', async () => {
  try { await invoke('Seek', Number(ui.seek.value)); }
  catch (_) { /* shown by invoke */ }
  finally { seeking = false; }
});

ui.volume.addEventListener('input', () => {
  ui.volumeValue.textContent = `${Math.round(Number(ui.volume.value) * 100)}%`;
});
ui.volume.addEventListener('change', () => invoke('SetVolume', Number(ui.volume.value)).catch(() => {}));

document.querySelectorAll('.panel-toggle').forEach((button) => {
  button.addEventListener('click', () => setPanelHidden(button.dataset.target, true));
});
ui.restore.addEventListener('click', () => {
  ['playlist', 'metadata'].forEach((name) => setPanelHidden(name, false));
});

function setPanelHidden(name, hidden) {
  const panel = document.querySelector(`[data-panel="${name}"]`);
  panel?.classList.toggle('hidden', hidden);
  ui.workspace.classList.toggle(`hide-${name}`, hidden);
  localStorage.setItem(`aura.panel.${name}`, hidden ? 'hidden' : 'visible');
}

document.addEventListener('keydown', (event) => {
  if (event.target.matches('input')) return;
  if (event.code === 'Space') { event.preventDefault(); invoke('Toggle').catch(() => {}); }
  if (event.key === 'j' && state.queue.length) selectedIndex = Math.min(state.queue.length - 1, selectedIndex + 1);
  if (event.key === 'k' && state.queue.length) selectedIndex = Math.max(0, selectedIndex - 1);
  if (event.key === 'Enter' && state.queue.length) playIndex(selectedIndex).catch(() => {});
  if (event.ctrlKey && event.key.toLowerCase() === 'o') { event.preventDefault(); invoke('OpenFiles').catch(() => {}); }
  const selected = ui.queue.querySelector(`[data-index="${selectedIndex}"]`);
  selected?.focus();
});

['playlist', 'metadata'].forEach((name) => {
  setPanelHidden(name, localStorage.getItem(`aura.panel.${name}`) === 'hidden');
});

async function refresh() {
  const api = backend();
  if (!api) return;
  try { render(await api.GetState()); }
  catch (error) { showError(typeof error === 'string' ? error : String(error)); }
}

setTimeout(refresh, 150);
setInterval(refresh, 350);
