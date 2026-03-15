'use strict';

// ─── Navigation ───────────────────────────────────────────────────────────────
const navItems = document.querySelectorAll('.nav-item');
const sections = document.querySelectorAll('.section');

navItems.forEach(btn => {
  btn.addEventListener('click', () => {
    navItems.forEach(b => b.classList.remove('active'));
    sections.forEach(s => s.classList.remove('active'));
    btn.classList.add('active');
    document.getElementById('section-' + btn.dataset.section)?.classList.add('active');
  });
});

// ─── Form refs ────────────────────────────────────────────────────────────────
const form          = document.getElementById('searchForm');
const searchBtn     = document.getElementById('searchBtn');
const clearBtn      = document.getElementById('clearBtn');
const loadingOverlay = document.getElementById('loadingOverlay');
const resultsArea   = document.getElementById('resultsArea');
const resultsGrid   = document.getElementById('resultsGrid');
const resultsMeta   = document.getElementById('resultsMeta');
const resultsSummary= document.getElementById('resultsSummary');
const loadingStatus = document.getElementById('loadingStatus');

// ─── Auto-format inputs ───────────────────────────────────────────────────────
document.getElementById('birth_date').addEventListener('input', e => {
  let v = e.target.value.replace(/[^\d]/g, '');
  if (v.length > 2) v = v.slice(0,2) + '.' + v.slice(2);
  if (v.length > 5) v = v.slice(0,5) + '.' + v.slice(5);
  e.target.value = v.slice(0, 10);
});

document.getElementById('inn').addEventListener('input', e => {
  e.target.value = e.target.value.replace(/[^\d]/g, '').slice(0, 12);
});

// ─── Photo upload ─────────────────────────────────────────────────────────────
const photoDrop      = document.getElementById('photoDrop');
const photoFile      = document.getElementById('photoFile');
const photoPreview   = document.getElementById('photoPreview');
const photoPreviewImg= document.getElementById('photoPreviewImg');
const photoDropInner = document.getElementById('photoDropInner');
const photoRemove    = document.getElementById('photoRemove');
const photoUrlInput  = document.getElementById('photo_url');

let photoDataURL = null; // base64 data URL для превью

function showPhotoPreview(src) {
  photoPreviewImg.src = src;
  photoDropInner.style.display = 'none';
  photoPreview.style.display  = 'flex';
}

function clearPhoto() {
  photoDataURL = null;
  photoFile.value = '';
  photoPreviewImg.src = '';
  photoPreview.style.display  = 'none';
  photoDropInner.style.display = 'flex';
}

photoFile.addEventListener('change', e => {
  const file = e.target.files[0];
  if (!file) return;
  if (file.size > 5 * 1024 * 1024) { alert('Файл слишком большой (макс. 5 МБ)'); return; }
  const reader = new FileReader();
  reader.onload = ev => {
    photoDataURL = ev.target.result;
    showPhotoPreview(photoDataURL);
    photoUrlInput.value = ''; // сбрасываем URL если загружен файл
  };
  reader.readAsDataURL(file);
});

photoRemove.addEventListener('click', e => {
  e.stopPropagation();
  clearPhoto();
});

photoUrlInput.addEventListener('input', () => {
  if (photoUrlInput.value.trim()) clearPhoto();
});

// Drag & drop
photoDrop.addEventListener('dragover', e => { e.preventDefault(); photoDrop.classList.add('drag-over'); });
photoDrop.addEventListener('dragleave', () => photoDrop.classList.remove('drag-over'));
photoDrop.addEventListener('drop', e => {
  e.preventDefault();
  photoDrop.classList.remove('drag-over');
  const file = e.dataTransfer.files[0];
  if (file && file.type.startsWith('image/')) {
    photoFile.files = e.dataTransfer.files;
    const evt = new Event('change');
    photoFile.dispatchEvent(evt);
  }
});

// ─── Clear form ───────────────────────────────────────────────────────────────
clearBtn.addEventListener('click', () => {
  form.reset();
  clearPhoto();
  resultsArea.style.display = 'none';
  resultsGrid.innerHTML = '';
});

// ─── Search ───────────────────────────────────────────────────────────────────
form.addEventListener('submit', async e => { e.preventDefault(); await runSearch(); });

async function runSearch() {
  const lastName = document.getElementById('last_name').value.trim();
  if (!lastName) { shakeInput(document.getElementById('last_name')); return; }

  resultsArea.style.display  = 'none';
  loadingOverlay.style.display = 'flex';
  searchBtn.disabled = true;

  const statuses = [
    'Запрос к ФССП...',
    'Запрос к ФНС ЕГРЮЛ/ЕГРИП...',
    'Проверка ИНН...',
    'Запрос к Федресурсу...',
    'Поиск в ВКонтакте...',
    'Формирование ссылок соцсетей...',
    'Обратный поиск по фото...',
    'Сборка результатов...',
  ];
  let si = 0;
  const iv = setInterval(() => { if (si < statuses.length) loadingStatus.textContent = statuses[si++]; }, 500);

  try {
    const t0 = Date.now();
    let resp;

    const hasFile = photoFile.files && photoFile.files[0];
    const hasUrl  = photoUrlInput.value.trim();

    if (hasFile) {
      // Отправляем как multipart/form-data
      const fd = new FormData(form);
      resp = await fetch('/api/search', { method: 'POST', body: fd });
    } else {
      // Отправляем как JSON
      const payload = {
        last_name:   document.getElementById('last_name').value.trim(),
        first_name:  document.getElementById('first_name').value.trim(),
        middle_name: document.getElementById('middle_name').value.trim(),
        birth_date:  document.getElementById('birth_date').value.trim(),
        inn:         document.getElementById('inn').value.trim(),
        region:      document.getElementById('region').value,
        photo_url:   hasUrl,
      };
      resp = await fetch('/api/search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
    }

    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const data = await resp.json();
    const elapsed = ((Date.now() - t0) / 1000).toFixed(2);

    clearInterval(iv);
    loadingOverlay.style.display = 'none';
    renderResults(data, elapsed);

  } catch (err) {
    clearInterval(iv);
    loadingOverlay.style.display = 'none';
    showErrorState(err.message);
  } finally {
    searchBtn.disabled = false;
  }
}

// ─── Render ───────────────────────────────────────────────────────────────────
const SOURCE_ICONS = {
  gavel:           '⚖',
  building:        '🏛',
  'credit-card':   '🪪',
  'alert-triangle':'⚠',
  scale:           '⚖',
  home:            '🏠',
  'external-link': '🔗',
  users:           '👥',
  camera:          '📷',
};

const STATUS_LABELS = {
  found:     'Найдено',
  not_found: 'Не найдено',
  manual:    'Ссылки',
  error:     'Ошибка',
  skip:      'Пропущено',
  pending:   'Ожидание',
};

const ORDER = { found: 0, manual: 1, not_found: 2, error: 3, skip: 4, pending: 5 };

function renderResults(data, elapsed) {
  const results = data.results || [];
  const counts = { found: 0, not_found: 0, manual: 0, error: 0 };
  results.forEach(r => { const k = r.status in counts ? r.status : 'error'; counts[k]++; });

  resultsMeta.textContent = `${results.length} источников · ${elapsed}с`;
  resultsSummary.innerHTML = [
    counts.found     > 0 ? `<div class="summary-chip"><div class="summary-dot dot-found"></div>Найдено: ${counts.found}</div>` : '',
    counts.not_found > 0 ? `<div class="summary-chip"><div class="summary-dot dot-not"></div>Не найдено: ${counts.not_found}</div>` : '',
    counts.manual    > 0 ? `<div class="summary-chip"><div class="summary-dot dot-manual"></div>Ссылки: ${counts.manual}</div>` : '',
    counts.error     > 0 ? `<div class="summary-chip"><div class="summary-dot dot-error"></div>Ошибок: ${counts.error}</div>` : '',
  ].join('');

  const sorted = [...results].sort((a, b) => (ORDER[a.status] || 5) - (ORDER[b.status] || 5));
  resultsGrid.innerHTML = '';
  sorted.forEach(r => resultsGrid.appendChild(buildResultCard(r)));

  resultsArea.style.display = 'block';
  resultsArea.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

function buildResultCard(res) {
  const card = document.createElement('div');
  card.className = `result-card status-${res.status}`;
  if (['found', 'manual'].includes(res.status)) card.classList.add('expanded');

  const icon = SOURCE_ICONS[res.icon] || '📋';
  const pill = `<span class="status-pill pill-${res.status}">${STATUS_LABELS[res.status] || res.status}</span>`;

  card.innerHTML = `
    <div class="result-card-header">
      <div class="result-header-left">
        <span class="result-source-icon">${icon}</span>
        <span class="result-source-name">${escHtml(res.source)}</span>
      </div>
      <div class="result-header-right">
        ${pill}
        <svg class="chevron" viewBox="0 0 16 16" fill="none">
          <path d="M4 6l4 4 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </div>
    </div>
    <div class="result-card-body">${buildCardBody(res)}</div>
  `;

  card.querySelector('.result-card-header').addEventListener('click', () => card.classList.toggle('expanded'));
  return card;
}

function buildCardBody(res) {
  let html = '';

  if (res.source_url) {
    html += `<a class="source-url-link" href="${escAttr(res.source_url)}" target="_blank" rel="noopener">${escHtml(res.source_url)} ↗</a>`;
  }

  if (res.status === 'error') {
    return html + `<div class="error-msg">${escHtml(res.error || 'Неизвестная ошибка')}</div>`;
  }
  if (res.status === 'not_found') {
    return html + `<div class="no-records">Записи не найдены в данном реестре</div>`;
  }
  if (res.status === 'skip') {
    return html + `<div class="no-records">${escHtml(res.error || 'Источник пропущен')}</div>`;
  }
  if (res.status === 'manual' && res.error) {
    html += `<div class="manual-note">ℹ ${escHtml(res.error)}</div>`;
  }

  if (!res.records || res.records.length === 0) {
    if (res.searched_url) {
      return html + `<div class="no-records"><a href="${escAttr(res.searched_url)}" target="_blank" rel="noopener">Открыть в источнике ↗</a></div>`;
    }
    return html + `<div class="no-records">Нет данных</div>`;
  }

  html += '<div class="records-list">';
  res.records.forEach(rec => { html += buildRecord(rec); });
  html += '</div>';
  return html;
}

function buildRecord(rec) {
  const tagsHtml = (rec.tags || []).map(t => `<span class="record-tag">${escHtml(t)}</span>`).join('');

  let fieldsHtml = '';
  (rec.fields || []).forEach(f => {
    if (f.kind === 'link') {
      fieldsHtml += `
        <div class="field-row">
          <span class="field-label">${escHtml(f.label)}</span>
          <a href="${escAttr(f.value)}" target="_blank" rel="noopener" class="field-value link-val">${escHtml(truncate(f.value, 80))}</a>
        </div>`;
    } else {
      const cls = { badge: 'badge-val', money: 'money-val', date: 'mono' }[f.kind] || '';
      fieldsHtml += `
        <div class="field-row">
          <span class="field-label">${escHtml(f.label)}</span>
          <span class="field-value ${cls}">${escHtml(f.value || '—')}</span>
        </div>`;
    }
  });

  const srcLink = rec.source_link
    ? `<div class="record-source-link">
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none"><path d="M7 2h3v3M10 2L5 7M4 3H2v7h7V8" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>
        <a href="${escAttr(rec.source_link)}" target="_blank" rel="noopener">${escHtml(truncate(rec.source_link, 80))}</a>
      </div>`
    : '';

  return `
    <div class="record">
      <div class="record-title">
        <span>${escHtml(truncate(rec.title || 'Запись', 80))}</span>
        <div class="record-tags">${tagsHtml}</div>
      </div>
      <div class="record-fields">${fieldsHtml}</div>
      ${srcLink}
    </div>`;
}

function showErrorState(msg) {
  resultsGrid.innerHTML = `
    <div class="card" style="border-left:3px solid var(--red)">
      <div class="error-msg">Ошибка: ${escHtml(msg)}<br>
        <small style="opacity:.6">Убедитесь что сервер запущен на порту 8080</small>
      </div>
    </div>`;
  resultsArea.style.display = 'block';
  resultsMeta.textContent = 'Ошибка';
  resultsSummary.innerHTML = '';
}

// ─── Utils ────────────────────────────────────────────────────────────────────
function shakeInput(el) {
  el.style.animation = 'none'; el.offsetHeight;
  el.style.animation = 'shake 0.35s ease';
  el.focus();
  el.addEventListener('animationend', () => { el.style.animation = ''; }, { once: true });
}

const shakeCSS = document.createElement('style');
shakeCSS.textContent = `@keyframes shake{0%,100%{transform:translateX(0)}20%{transform:translateX(-6px)}40%{transform:translateX(6px)}60%{transform:translateX(-4px)}80%{transform:translateX(4px)}}`;
document.head.appendChild(shakeCSS);

function escHtml(s) {
  return String(s ?? '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}
function escAttr(s) { return escHtml(s); }
function truncate(s, n) { return s && s.length > n ? s.slice(0, n) + '…' : (s || ''); }

// Ctrl/Cmd+Enter — поиск
document.addEventListener('keydown', e => {
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && !searchBtn.disabled) runSearch();
});
