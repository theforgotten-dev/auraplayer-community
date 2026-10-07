const $ = (selector) => document.querySelector(selector);

const ui = {
  open: $('#open-files'), queue: $('#queue'), queueCount: $('#queue-count'), title: $('#track-title'),
  codec: $('#codec-chip'), statusDot: $('#status-dot'), engineLabel: $('#engine-label'), visualizer: $('#visualizer'),
  seek: $('#seek'), position: $('#position'), duration: $('#duration'), volume: $('#volume'), volumeValue: $('#volume-value'),
  toggle: $('#toggle'), stop: $('#stop'), previous: $('#previous'), next: $('#next'), toast: $('#toast'),
  metaTitle: $('#meta-title'), metaCodec: $('#meta-codec'), metaRate: $('#meta-rate'), metaChannels: $('#meta-channels'),
  metaPath: $('#meta-path'), workspace: $('.workspace'), restore: $('#restore-panels'), spectrum: $('#spectrum-canvas'),
  levelLeft: $('#level-left'), levelRight: $('#level-right'), skins: $('#open-skins'), skinDialog: $('#skin-dialog'),
  skinName: $('#skin-name'), importSkin: $('#import-skin'), customCSS: $('#skin-custom-css')
};

const EMPTY_STATE = { state: 'stopped', queue: [], currentIndex: -1, position: 0, volume: .8, error: '' };
const SKIN_STORAGE = 'aura.skin.v1';
const BUILTIN_SKINS = {
  console: { version: 1, name: 'PHOSPHOR_GREEN', variables: {
    '--bg': '#070a09', '--surface': '#0b100e', '--line': '#1d4b38', '--line-hot': '#39d98a', '--text': '#c9f7dd',
    '--muted': '#668c77', '--green': '#53f29c', '--cyan': '#5ee8e2', '--danger': '#ff6b81',
    '--panel-bg': 'rgba(8, 14, 11, .96)', '--control-bg': '#09100d', '--visualizer-bg': '#050806',
    '--channel-left': '#53f29c', '--channel-right': '#5ee8e2'
  }},
  amber: { version: 1, name: 'AMBER_CRT', variables: {
    '--bg': '#0c0903', '--surface': '#171006', '--line': '#624617', '--line-hot': '#ffb52e', '--text': '#ffe5a8',
    '--muted': '#9d7740', '--green': '#ffb52e', '--cyan': '#ffe17a', '--danger': '#ff664d',
    '--panel-bg': 'rgba(20, 13, 4, .97)', '--control-bg': '#171006', '--visualizer-bg': '#090601',
    '--channel-left': '#ff9d24', '--channel-right': '#ffe17a'
  }},
  cyber: { version: 1, name: 'CYBER_VIOLET', variables: {
    '--bg': '#070512', '--surface': '#100b20', '--line': '#483080', '--line-hot': '#bb74ff', '--text': '#eee4ff',
    '--muted': '#8973aa', '--green': '#bb74ff', '--cyan': '#48e8ff', '--danger': '#ff4f91',
    '--panel-bg': 'rgba(13, 8, 28, .97)', '--control-bg': '#100a20', '--visualizer-bg': '#05030d',
    '--channel-left': '#bb74ff', '--channel-right': '#48e8ff'
  }}
};

let state = EMPTY_STATE;
let selectedIndex = 0;
let seeking = false;
let toastTimer;
let queueSignature = '';
let spectrumBusy = false;
let appliedVariables = [];
const spectrumTargetLeft = new Float32Array(16);
const spectrumTargetRight = new Float32Array(16);
const spectrumDisplayLeft = new Float32Array(16);
const spectrumDisplayRight = new Float32Array(16);
const spectrumLabels = ['50', '79', '126', '200', '316', '502', '794', '1.3K', '2.0K', '3.2K', '5.0K', '7.1K', '8.9K', '11K', '14K', '18K'];
let targetLeftLevel = 0;
let targetRightLevel = 0;
let displayLeftLevel = 0;
let displayRightLevel = 0;
let spectrumPalette;

function backend() { return window.go?.main?.App; }

async function invoke(method, ...args) {
  const api = backend();
  if (!api || typeof api[method] !== 'function') throw new Error('El puente con Go todavía no está disponible.');
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
  state = nextState || EMPTY_STATE;
  if (state.currentIndex >= 0) selectedIndex = state.currentIndex;
  const track = state.queue?.[state.currentIndex];
  ui.queueCount.textContent = String(state.queue?.length ?? 0).padStart(2, '0');
  ui.title.textContent = track?.title || 'NO_TRACK_SELECTED';
  ui.codec.textContent = track?.codec || '—';
  ui.metaTitle.textContent = track?.title || '—';
  ui.metaCodec.textContent = track?.codec || '—';
  ui.metaRate.textContent = track ? `${(track.sampleRate / 1000).toFixed(1)} kHz` : '—';
  ui.metaChannels.textContent = track ? (track.channels === 2 ? 'Estéreo' : 'Mono') : '—';
  ui.metaPath.textContent = track?.path || '—';
  ui.metaPath.title = track?.path || '';
  const duration = track?.duration || 0;
  if (!seeking) { ui.seek.max = duration; ui.seek.value = Math.min(state.position || 0, duration); }
  ui.position.textContent = formatTime(state.position || 0);
  ui.duration.textContent = formatTime(duration);
  ui.volume.value = state.volume;
  ui.volumeValue.textContent = `${Math.round(state.volume * 100)}%`;
  const playing = state.state === 'playing';
  ui.toggle.textContent = playing ? 'PAUSE' : 'PLAY';
  ui.toggle.setAttribute('aria-label', playing ? 'Pausar' : 'Reproducir');
  ui.visualizer.classList.toggle('playing', playing);
  ui.statusDot.className = `status-dot ${playing ? 'playing' : state.state === 'error' ? 'error' : ''}`;
  ui.engineLabel.textContent = statusLabel(state.state);
  renderQueue();
  if (state.error) showError(state.error);
}

function renderQueue() {
  const signature = `${state.currentIndex}|${(state.queue || []).map((track) => track.id).join('|')}`;
  if (signature === queueSignature) return;
  queueSignature = signature;
  if (!state.queue?.length) {
    ui.queue.innerHTML = '<div class="empty-state"><div class="empty-icon" aria-hidden="true">[ ]</div><strong>QUEUE_EMPTY</strong><span>LOAD ONE OR MORE MP3 FILES</span></div>';
    return;
  }
  ui.queue.replaceChildren(...state.queue.map((track, index) => {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = `queue-item ${index === state.currentIndex ? 'active' : ''}`;
    button.dataset.index = index;
    button.role = 'option';
    button.setAttribute('aria-selected', index === state.currentIndex ? 'true' : 'false');
    button.innerHTML = `<span class="queue-number">${String(index + 1).padStart(2, '0')}</span><span class="queue-title"></span><span class="queue-duration">${formatTime(track.duration)}</span>`;
    button.querySelector('.queue-title').textContent = track.title;
    button.addEventListener('click', () => playIndex(index));
    return button;
  }));
}

function renderSpectrum(spectrum) {
  if (!spectrum?.frequencies?.length) return;
  const active = state.state === 'playing';
  for (let index = 0; index < 16; index++) {
    spectrumTargetLeft[index] = active ? clamp(spectrum.left?.[index]) : 0;
    spectrumTargetRight[index] = active ? clamp(spectrum.right?.[index]) : 0;
    if (!spectrumLabels[index]) spectrumLabels[index] = formatFrequency(spectrum.frequencies[index]);
  }
  targetLeftLevel = active ? clamp(spectrum.leftLevel) : 0;
  targetRightLevel = active ? clamp(spectrum.rightLevel) : 0;
}

function animateSpectrum() {
  const canvas = ui.spectrum;
  const cssWidth = Math.max(1, Math.floor(canvas.clientWidth));
  const cssHeight = Math.max(1, Math.floor(canvas.clientHeight));
  const ratio = Math.min(window.devicePixelRatio || 1, 2);
  const pixelWidth = Math.floor(cssWidth * ratio);
  const pixelHeight = Math.floor(cssHeight * ratio);
  if (canvas.width !== pixelWidth || canvas.height !== pixelHeight) { canvas.width = pixelWidth; canvas.height = pixelHeight; }
  const context = canvas.getContext('2d');
  context.setTransform(ratio, 0, 0, ratio, 0, 0);
  context.clearRect(0, 0, cssWidth, cssHeight);
  if (!spectrumPalette) updateSpectrumPalette();

  const labelHeight = 18;
  const graphHeight = Math.max(1, cssHeight - labelHeight);
  const gap = Math.max(2, Math.min(7, cssWidth * .0045));
  const bandWidth = (cssWidth - gap * 15) / 16;
  context.font = `${Math.max(6, Math.min(8, cssWidth / 100))}px monospace`;
  context.textAlign = 'center';
  for (let index = 0; index < 16; index++) {
    spectrumDisplayLeft[index] += (spectrumTargetLeft[index] - spectrumDisplayLeft[index]) * .22;
    spectrumDisplayRight[index] += (spectrumTargetRight[index] - spectrumDisplayRight[index]) * .22;
    const x = index * (bandWidth + gap);
    const meterGap = 2;
    const meterWidth = (bandWidth - meterGap) / 2;
    drawMeter(context, x, graphHeight, meterWidth, spectrumDisplayLeft[index], spectrumPalette.left);
    drawMeter(context, x + meterWidth + meterGap, graphHeight, meterWidth, spectrumDisplayRight[index], spectrumPalette.right);
    context.fillStyle = spectrumPalette.muted;
    context.fillText(spectrumLabels[index], x + bandWidth / 2, cssHeight - 3);
  }
  displayLeftLevel += (targetLeftLevel - displayLeftLevel) * .22;
  displayRightLevel += (targetRightLevel - displayRightLevel) * .22;
  ui.levelLeft.style.transform = `scaleX(${displayLeftLevel})`;
  ui.levelRight.style.transform = `scaleX(${displayRightLevel})`;
  requestAnimationFrame(animateSpectrum);
}

function drawMeter(context, x, graphHeight, width, level, color) {
  const height = Math.max(1, level * graphHeight);
  context.fillStyle = color;
  context.fillRect(x, graphHeight - height, width, height);
  context.fillStyle = spectrumPalette.mask;
  for (let y = graphHeight - 8; y > graphHeight - height; y -= 10) context.fillRect(x, y, width, 2);
}

function updateSpectrumPalette() {
  const style = getComputedStyle(document.documentElement);
  spectrumPalette = {
    left: style.getPropertyValue('--channel-left').trim(), right: style.getPropertyValue('--channel-right').trim(),
    muted: style.getPropertyValue('--muted').trim(), mask: style.getPropertyValue('--visualizer-bg').trim()
  };
}

function validateSkin(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('El manifiesto de skin debe ser un objeto JSON.');
  if (value.version !== 1) throw new Error('La skin debe declarar "version": 1.');
  if (typeof value.name !== 'string' || !value.name.trim() || value.name.length > 64) throw new Error('La skin necesita un nombre de hasta 64 caracteres.');
  if (value.variables !== undefined && (typeof value.variables !== 'object' || Array.isArray(value.variables))) throw new Error('"variables" debe ser un objeto.');
  if (value.css !== undefined && (typeof value.css !== 'string' || value.css.length > 128000)) throw new Error('"css" debe ser texto de hasta 128 KiB.');
  for (const [key, content] of Object.entries(value.variables || {})) {
    if (!key.startsWith('--') || typeof content !== 'string' || key.length > 80 || content.length > 256) throw new Error(`Variable de skin no válida: ${key}`);
  }
  return value;
}

function applySkin(manifest, persist = true) {
  const skin = validateSkin(manifest);
  appliedVariables.forEach((name) => document.documentElement.style.removeProperty(name));
  appliedVariables = Object.keys(skin.variables || {});
  for (const [name, value] of Object.entries(skin.variables || {})) document.documentElement.style.setProperty(name, value);
  ui.customCSS.textContent = skin.css || '';
  ui.skinName.textContent = skin.name.toUpperCase();
  document.body.dataset.skin = skin.name;
  spectrumPalette = undefined;
  if (persist) localStorage.setItem(SKIN_STORAGE, JSON.stringify(skin));
}

function loadStoredSkin() {
  try {
    const stored = localStorage.getItem(SKIN_STORAGE);
    applySkin(stored ? JSON.parse(stored) : BUILTIN_SKINS.console, false);
  } catch (error) {
    localStorage.removeItem(SKIN_STORAGE);
    applySkin(BUILTIN_SKINS.console, false);
    showError(`Skin descartada: ${error.message}`);
  }
}

async function playIndex(index) { selectedIndex = index; await invoke('Play', index); }
function clamp(value) { return Math.max(0, Math.min(1, Number(value) || 0)); }
function formatFrequency(value) { return value >= 1000 ? `${(value / 1000).toFixed(value < 10000 ? 1 : 0)}K` : String(Math.round(value)); }
function formatTime(seconds) {
  if (!Number.isFinite(seconds) || seconds < 0) return '0:00';
  const rounded = Math.floor(seconds);
  return `${Math.floor(rounded / 60)}:${String(rounded % 60).padStart(2, '0')}`;
}
function statusLabel(value) { return ({ playing: 'ENGINE_PLAYING', paused: 'ENGINE_PAUSED', loading: 'ENGINE_LOADING', error: 'ENGINE_ERROR', stopped: 'ENGINE_READY' })[value] || 'ENGINE_READY'; }
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
ui.seek.addEventListener('change', async () => { try { await invoke('Seek', Number(ui.seek.value)); } catch (_) { /* reported */ } finally { seeking = false; } });
ui.volume.addEventListener('input', () => { ui.volumeValue.textContent = `${Math.round(Number(ui.volume.value) * 100)}%`; });
ui.volume.addEventListener('change', () => invoke('SetVolume', Number(ui.volume.value)).catch(() => {}));
document.querySelectorAll('.panel-toggle').forEach((button) => button.addEventListener('click', () => setPanelHidden(button.dataset.target, true)));
ui.restore.addEventListener('click', () => ['playlist', 'metadata'].forEach((name) => setPanelHidden(name, false)));
ui.skins.addEventListener('click', () => ui.skinDialog.showModal());
document.querySelectorAll('[data-skin]').forEach((button) => button.addEventListener('click', () => applySkin(BUILTIN_SKINS[button.dataset.skin])));
ui.importSkin.addEventListener('click', async () => {
  try { const contents = await backend()?.OpenSkin(); if (contents) applySkin(JSON.parse(contents)); }
  catch (error) { showError(typeof error === 'string' ? error : error?.message || String(error)); }
});
ui.skinDialog.addEventListener('click', (event) => { if (event.target === ui.skinDialog) ui.skinDialog.close(); });

function setPanelHidden(name, hidden) {
  document.querySelector(`[data-panel="${name}"]`)?.classList.toggle('hidden', hidden);
  ui.workspace.classList.toggle(`hide-${name}`, hidden);
  localStorage.setItem(`aura.panel.${name}`, hidden ? 'hidden' : 'visible');
}

document.addEventListener('keydown', (event) => {
  if (event.target.matches('input, button')) return;
  if (event.code === 'Space') { event.preventDefault(); invoke('Toggle').catch(() => {}); }
  if (event.key === 'j' && state.queue.length) selectedIndex = Math.min(state.queue.length - 1, selectedIndex + 1);
  if (event.key === 'k' && state.queue.length) selectedIndex = Math.max(0, selectedIndex - 1);
  if (event.key === 'Enter' && state.queue.length) playIndex(selectedIndex).catch(() => {});
  if (event.ctrlKey && event.key.toLowerCase() === 'o') { event.preventDefault(); invoke('OpenFiles').catch(() => {}); }
  ui.queue.querySelector(`[data-index="${selectedIndex}"]`)?.focus();
});

async function refresh() {
  const api = backend();
  if (!api) return;
  try { render(await api.GetState()); } catch (error) { showError(typeof error === 'string' ? error : String(error)); }
}

async function refreshSpectrum() {
  const api = backend();
  if (!api || spectrumBusy || state.state !== 'playing') {
    if (state.state !== 'playing') {
      spectrumTargetLeft.fill(0); spectrumTargetRight.fill(0); targetLeftLevel = 0; targetRightLevel = 0;
    }
    return;
  }
  spectrumBusy = true;
  try { renderSpectrum(await api.GetSpectrum()); } catch (_) { /* regular refresh reports errors */ } finally { spectrumBusy = false; }
}

loadStoredSkin();
['playlist', 'metadata'].forEach((name) => setPanelHidden(name, localStorage.getItem(`aura.panel.${name}`) === 'hidden'));
setTimeout(refresh, 150);
setInterval(refresh, 500);
setInterval(refreshSpectrum, 120);
requestAnimationFrame(animateSpectrum);
