'use strict';

// ─── Navigation ───────────────────────────────────────────────────────────────
const navItems = document.querySelectorAll('.nav-item');
const sections = document.querySelectorAll('.section');

navItems.forEach(btn => {
  btn.addEventListener('click', () => {
    navItems.forEach(b => b.classList.remove('active'));
    sections.forEach(s => s.classList.remove('active'));
    btn.classList.add('active');
    const sectionId = 'section-' + btn.dataset.section;
    document.getElementById(sectionId)?.classList.add('active');
  });
});

// ─── Form ─────────────────────────────────────────────────────────────────────
const form = document.getElementById('searchForm');
const searchBtn = document.getElementById('searchBtn');
const clearBtn = document.getElementById('clearBtn');
const loadingOverlay = document.getElementById('loadingOverlay');
const resultsArea = document.getElementById('resultsArea');
const resultsGrid = document.getElementById('resultsGrid');
const resultsMeta = document.getElementById('resultsMeta');
const resultsSummary = document.getElementById('resultsSummary');
const loadingStatus = document.getElementById('loadingStatus');

clearBtn.addEventListener('click', () => {
  form.reset();
  resultsArea.style.display = 'none';
  resultsGrid.innerHTML = '';
});

// Авто-форматирование даты
const bdInput = document.getElementById('birth_date');
bdInput.addEventListener('input', (e) => {
  let val = e.target.value.replace(/[^\d]/g, '');
  if (val.length > 2) val = val.slice(0,2) + '.' + val.slice(2);
  if (val.length > 5) val = val.slice(0,5) + '.' + val.slice(5);
  if (val.length > 10) val = val.slice(0,10);
  e.target.value = val;
});

// Авто-форматирование ИНН
const innInput = document.getElementById('inn');
innInput.addEventListener('input', (e) => {
  e.target.value = e.target.value.replace(/[^\d]/g, '').slice(0, 12);
});

// ─── Search ───────────────────────────────────────────────────────────────────
form.addEventListener('submit', async (e) => {
  e.preventDefault();
  await runSearch();
});

async function runSearch() {
  const data = collectFormData();

  if (!data.last_name) {
    shakeInput(document.getElementById('last_name'));
    return;
  }

  // Show loading
  resultsArea.style.display = 'none';
  loadingOverlay.style.display = 'flex';
  searchBtn.disabled = true;

  const statuses = [
    'Запрос к ФССП...',
    'Запрос к ФНС ЕГРЮЛ/ЕГРИП...',
    'Проверка ИНН...',
    'Запрос к Федресурсу...',
    'Формирование ссылок Росреестр...',
    'Сборка результатов...'
  ];

  let statusIdx = 0;
  const statusInterval = setInterval(() => {
    if (statusIdx < statuses.length) {
      loadingStatus.textContent = statuses[statusIdx++];
    }
  }, 600);

  try {
    const t0 = Date.now();
    const resp = await fetch('/api/search', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });

    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const result = await resp.json();
    const elapsed = ((Date.now() - t0) / 1000).toFixed(2);

    clearInterval(statusInterval);
    loadingOverlay.style.display = 'none';
    renderResults(result, elapsed);

  } catch (err) {
    clearInterval(statusInterval);
    loadingOverlay.style.display = 'none';
    showError(err.message);
  } finally {
    searchBtn.disabled = false;
  }
}

function collectFormData() {
  const fd = new FormData(form);
  return {
    last_name:   fd.get('last_name')?.trim() || '',
    first_name:  fd.get('first_name')?.trim() || '',
    middle_name: fd.get('middle_name')?.trim() || '',
    birth_date:  fd.get('birth_date')?.trim() || '',
    inn:         fd.get('inn')?.trim() || '',
    region:      fd.get('region') || '-1'
  };
}

function shakeInput(el) {
  el.style.animation = 'none';
  el.offsetHeight;
  el.style.animation = 'shake 0.35s ease';
  el.focus();
  el.addEventListener('animationend', () => { el.style.animation = ''; }, { once: true });
}

// Добавляем анимацию shake в CSS динамически
const shakeStyle = document.createElement('style');
shakeStyle.textContent = `
@keyframes shake {
  0%,100%{transform:translateX(0)}
  20%{transform:translateX(-6px)}
  40%{transform:translateX(6px)}
  60%{transform:translateX(-4px)}
  80%{transform:translateX(4px)}
}`;
document.head.appendChild(shakeStyle);

// ─── Render Results ───────────────────────────────────────────────────────────
const SOURCE_ICONS = {
  gavel:           '⚖',
  building:        '🏛',
  'credit-card':   '🪪',
  'alert-triangle':'⚠',
  scale:           '⚖',
  home:            '🏠',
  'external-link': '🔗',
};

const STATUS_LABELS = {
  found:     'Найдено',
  not_found: 'Не найдено',
  manual:    'Ссылки',
  error:     'Ошибка',
  skip:      'Пропущено',
  pending:   'Ожидание',
};

function renderResults(data, elapsed) {
  const results = data.results || [];

  // Summary
  const counts = { found: 0, not_found: 0, manual: 0, error: 0 };
  results.forEach(r => {
    const k = r.status in counts ? r.status : 'error';
    counts[k]++;
  });

  resultsMeta.textContent = `${results.length} источников · ${elapsed}с`;

  resultsSummary.innerHTML = [
    counts.found > 0    ? `<div class="summary-chip"><div class="summary-dot dot-found"></div>Найдено: ${counts.found}</div>` : '',
    counts.not_found > 0 ? `<div class="summary-chip"><div class="summary-dot dot-not"></div>Не найдено: ${counts.not_found}</div>` : '',
    counts.manual > 0   ? `<div class="summary-chip"><div class="summary-dot dot-manual"></div>Ссылки: ${counts.manual}</div>` : '',
    counts.error > 0    ? `<div class="summary-chip"><div class="summary-dot dot-error"></div>Ошибок: ${counts.error}</div>` : '',
  ].join('');

  // Sort: found first, then manual, then not_found, then error
  const ORDER = { found: 0, manual: 1, not_found: 2, error: 3, skip: 4, pending: 5 };
  const sorted = [...results].sort((a, b) => (ORDER[a.status] || 5) - (ORDER[b.status] || 5));

  resultsGrid.innerHTML = '';
  sorted.forEach((res, i) => {
    resultsGrid.appendChild(buildResultCard(res, i));
  });

  resultsArea.style.display = 'block';
  resultsArea.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

function buildResultCard(res, idx) {
  const card = document.createElement('div');
  card.className = `result-card status-${res.status}`;
  if (res.status === 'found' || res.status === 'manual') {
    card.classList.add('expanded');
  }

  const icon = SOURCE_ICONS[res.icon] || '📋';
  const statusLabel = STATUS_LABELS[res.status] || res.status;
  const pillClass = `pill-${res.status}`;

  card.innerHTML = `
    <div class="result-card-header">
      <div class="result-header-left">
        <span class="result-source-icon">${icon}</span>
        <span class="result-source-name">${escHtml(res.source)}</span>
      </div>
      <div class="result-header-right">
        <span class="status-pill ${pillClass}">${statusLabel}</span>
        <svg class="chevron" viewBox="0 0 16 16" fill="none">
          <path d="M4 6l4 4 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </div>
    </div>
    <div class="result-card-body">
      ${buildCardBody(res)}
    </div>
  `;

  card.querySelector('.result-card-header').addEventListener('click', () => {
    card.classList.toggle('expanded');
  });

  return card;
}

function buildCardBody(res) {
  let html = '';

  if (res.source_url) {
    html += `<a class="source-url-link" href="${escAttr(res.source_url)}" target="_blank" rel="noopener">
      ${escHtml(res.source_url)} ↗
    </a>`;
  }

  if (res.status === 'error') {
    html += `<div class="error-msg">${escHtml(res.error || 'Неизвестная ошибка')}</div>`;
    return html;
  }

  if (res.status === 'not_found') {
    html += `<div class="no-records">Записи не найдены в данном реестре</div>`;
    return html;
  }

  if (res.status === 'skip') {
    html += `<div class="no-records">${escHtml(res.error || 'Источник пропущен')}</div>`;
    return html;
  }

  if (res.status === 'manual' && res.error) {
    html += `<div class="manual-note">ℹ ${escHtml(res.error)}</div>`;
  }

  if (!res.records || res.records.length === 0) {
    if (res.searched_url) {
      html += `<div class="no-records">
        <a href="${escAttr(res.searched_url)}" target="_blank" rel="noopener">Открыть в источнике ↗</a>
      </div>`;
    } else {
      html += `<div class="no-records">Нет данных</div>`;
    }
    return html;
  }

  html += '<div class="records-list">';
  res.records.forEach(rec => {
    html += buildRecord(rec);
  });
  html += '</div>';

  return html;
}

function buildRecord(rec) {
  const tagsHtml = (rec.tags || []).map(t => `<span class="record-tag">${escHtml(t)}</span>`).join('');

  let fieldsHtml = '';
  (rec.fields || []).forEach(f => {
    const valClass = getFieldValueClass(f.kind);
    let valContent = '';
    if (f.kind === 'link') {
      valContent = `<a href="${escAttr(f.value)}" target="_blank" rel="noopener" class="field-value link-val">${escHtml(truncate(f.value, 80))}</a>`;
    } else {
      valContent = `<span class="field-value ${valClass}">${escHtml(f.value || '—')}</span>`;
    }
    fieldsHtml += `
      <div class="field-row">
        <span class="field-label">${escHtml(f.label)}</span>
        ${valContent}
      </div>`;
  });

  const sourceLink = rec.source_link
    ? `<div class="record-source-link">
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
          <path d="M7 2h3v3M10 2L5 7M4 3H2v7h7V8" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
        </svg>
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
      ${sourceLink}
    </div>
  `;
}

function getFieldValueClass(kind) {
  switch (kind) {
    case 'badge':  return 'badge-val';
    case 'money':  return 'money-val';
    case 'text':   return '';
    case 'date':   return 'mono';
    default:       return '';
  }
}

// ─── Error state ──────────────────────────────────────────────────────────────
function showError(msg) {
  resultsGrid.innerHTML = `
    <div class="card" style="border-left: 3px solid var(--red)">
      <div class="error-msg">Ошибка запроса: ${escHtml(msg)}<br>
        <small style="opacity:.7">Убедитесь, что сервер запущен и доступен</small>
      </div>
    </div>
  `;
  resultsArea.style.display = 'block';
  resultsMeta.textContent = 'Ошибка';
  resultsSummary.innerHTML = '';
}

// ─── Utils ────────────────────────────────────────────────────────────────────
function escHtml(str) {
  if (!str) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function escAttr(str) {
  return escHtml(str);
}

function truncate(str, n) {
  if (!str) return '';
  return str.length > n ? str.slice(0, n) + '…' : str;
}

// ─── Keyboard shortcut ────────────────────────────────────────────────────────
document.addEventListener('keydown', (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    if (!searchBtn.disabled) runSearch();
  }
});
