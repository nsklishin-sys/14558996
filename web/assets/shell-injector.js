// ── Shell Injector ──────────────────────────────────────────────
// Унифицированный topbar и nav для всех залогиненных страниц.
// Эталон — home-auth.html. Подключается через middleware-инжекцию.
//
// Логика:
//   1. Если на странице есть <header class="topbar"></header> (пустой) —
//      наполняет его эталонной разметкой (search + profile dropdown).
//   2. Если есть <nav class="nav"></nav> (пустой) — наполняет nav-эталоном.
//   3. Подсвечивает active по location.pathname.
//   4. Заполняет topbarAv/Name/Role + pddAv/Name/Role из localStorage.user
//      и обновляет с /api/me если есть токен.
//   5. Подключает toggleProfileDD, закрытие по клику снаружи, Escape.
//
// На страницах где topbar/nav уже наполнены вручную (legacy) — НЕ трогает.

(function () {
  if (window.__lastopShellInjected) return;
  window.__lastopShellInjected = true;

  // ── HTML-эталон topbar ──
  const TOPBAR_HTML = `
  <div class="search-wrap">
    <span class="search-icon"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7"></circle><path d="M20 20l-3.5-3.5"></path></svg></span>
    <input type="search" id="searchInput" placeholder="Поиск пользователей, компаний, форума…" autocomplete="off">
  </div>
  <div class="profile-dd-wrap">
    <div class="topbar-profile" id="topbarProfile" onclick="toggleProfileDD(event)">
      <div class="tp-av" id="topbarAv">?</div>
      <div class="tp-info">
        <span class="tp-name" id="topbarName">Загрузка…</span>
        <span class="tp-role" id="topbarRole">Участник LASTOP</span>
      </div>
      <svg class="tp-arrow" viewBox="0 0 24 24"><polyline points="6 9 12 15 18 9"/></svg>
    </div>
    <div class="profile-dd" id="profileDD">
      <div class="pdd-head">
        <div class="pdd-av" id="pddAv">?</div>
        <div class="pdd-info">
          <div class="pdd-name" id="pddName">Загрузка…</div>
          <div class="pdd-role" id="pddRole">Участник LASTOP</div>
        </div>
      </div>
      <a href="/profile.html" class="pdd-view">Открыть профиль</a>
      <div class="pdd-body">
        <a href="/friends.html" class="pdd-item">
          <svg viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          <span class="pdd-item-label">Друзья</span>
          <span class="pdd-count" id="pddFriends">0</span>
        </a>
        <a class="pdd-item" href="/saved.html">
          <svg viewBox="0 0 24 24"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>
          Сохранённое
        </a>
        <a href="/my-company.html" class="pdd-item">
          <svg viewBox="0 0 24 24"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/></svg>
          <span class="pdd-item-label">Моя компания</span>
        </a>
        <a href="/communities.html?filter=my" class="pdd-item">
          <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
          <span class="pdd-item-label">Мои сообщества</span>
          <span class="pdd-count" id="pddComm">0</span>
        </a>
        <a href="/analytics.html" class="pdd-item">
          <svg viewBox="0 0 24 24"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
          <span class="pdd-item-label">Аналитика</span>
        </a>
        <div id="profileContextSlot"></div>
        <a href="javascript:void(0)" class="pdd-item logout" onclick="logout()">
          <svg viewBox="0 0 24 24"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
          <span class="pdd-item-label">Выход</span>
        </a>
      </div>
    </div>
  </div>`;

  // ── HTML-эталон nav ──
  const NAV_HTML = `
  <div class="nav-group">Навигация</div>
  <a class="nav-item" href="/" data-match="^/$|^/home-auth\\.html$">
    <svg viewBox="0 0 24 24"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9,22 9,12 15,12 15,22"/></svg>
    <span>Главная</span>
  </a>
  <a class="nav-item" href="/dashboard.html" data-match="^/dashboard|^/news-detail">
    <svg viewBox="0 0 24 24"><path d="M4 6h16M4 12h16M4 18h10"/></svg>
    <span>Новости</span>
  </a>
  <a class="nav-item" href="/projects.html" data-match="^/projects|^/project-detail">
    <svg viewBox="0 0 24 24"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
    <span>Проекты</span>
  </a>
  <a class="nav-item" href="/events.html" data-match="^/events|^/event-detail">
    <svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
    <span>Мероприятия</span>
  </a>
  <a class="nav-item" href="/exhibitions.html" data-match="^/exhibition">
    <svg viewBox="0 0 24 24"><path d="M3 9l9-7 9 7"/><rect x="5" y="9" width="14" height="12" rx="1"/><path d="M9 21V13h6v8"/></svg>
    <span>Выставки</span>
  </a>
  <a class="nav-item" href="/jobs.html" data-match="^/jobs|^/job-detail|^/resume-detail">
    <svg viewBox="0 0 24 24"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><circle cx="12" cy="13" r="2"/><path d="M8 19c0-2.2 1.8-4 4-4s4 1.8 4 4"/></svg>
    <span>Резюме</span>
  </a>
  <a class="nav-item" href="/catalog.html" data-match="^/catalog|^/product-detail|^/service-detail">
    <svg viewBox="0 0 24 24"><path d="M6 2L3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4z"/><line x1="3" y1="6" x2="21" y2="6"/><path d="M16 10a4 4 0 0 1-8 0"/></svg>
    <span>Товары и услуги</span>
  </a>
  <div class="nav-group">Общение</div>
  <a class="nav-item" href="/forum.html" data-match="^/forum">
    <svg viewBox="0 0 24 24"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
    <span>Форум</span>
  </a>
  <a class="nav-item" href="/companies.html" data-match="^/companies|^/company-detail|^/my-company">
    <svg viewBox="0 0 24 24"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/></svg>
    <span>Компании</span>
  </a>
  <a class="nav-item" href="/communities.html" data-match="^/communities|^/community-detail|^/my-community">
    <svg viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
    <span>Сообщества</span>
  </a>
  <div class="nav-group">Профиль</div>
  <a class="nav-item" href="/chat.html" data-match="^/chat">
    <svg viewBox="0 0 24 24"><rect x="2" y="4" width="20" height="16" rx="2"/><polyline points="2,4 12,13 22,4"/></svg>
    <span>Чат</span>
  </a>
  <a class="nav-item" href="/profile.html" data-match="^/profile">
    <svg viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
    <span>Профиль</span>
  </a>
  <div class="nav-bottom">
    <a class="nav-item" href="/settings.html" data-match="^/settings" style="color:var(--gmt)">
      <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
      <span>Настройки</span>
    </a>
  </div>`;

  // ── HTML для bottom-nav (мобильная нижняя навигация ≤640px) ──
  const BOTTOM_NAV_HTML = `
    <a href="/home-auth.html" class="bn-item" data-match="^/(home-auth\.html|index_.*\.html)?$">
      <svg viewBox="0 0 24 24"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9,22 9,12 15,12 15,22"/></svg>
      <span>Главная</span>
    </a>
    <button type="button" class="bn-item bn-menu-btn" onclick="lastopOpenBottomMenu()">
      <svg viewBox="0 0 24 24"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
      <span>Меню</span>
    </button>
    <button type="button" class="bn-item bn-create-btn" onclick="lastopOpenCreateAction()" aria-label="Создать">
      <span class="bn-create-circle">
        <svg viewBox="0 0 24 24"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
      </span>
    </button>
    <a href="/chat.html" class="bn-item" data-match="^/chat\.html">
      <svg viewBox="0 0 24 24"><rect x="2" y="4" width="20" height="16" rx="2"/><polyline points="2,4 12,13 22,4"/></svg>
      <span>Чат</span>
      <span class="bn-badge" id="bnChatBadge" style="display:none">0</span>
    </a>
    <a href="/profile.html" class="bn-item" data-match="^/profile\.html">
      <svg viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
      <span>Профиль</span>
    </a>`;

  // ── HTML для bottom-menu-sheet (раскрывающееся меню всех разделов) ──
  const BOTTOM_MENU_HTML = `
    <div class="bms-title">Все разделы</div>
    <div class="bms-grid">
      <a href="/dashboard.html" class="bms-item">
        <svg viewBox="0 0 24 24"><path d="M4 6h16M4 12h16M4 18h10"/></svg>
        <span>Новости</span>
      </a>
      <a href="/forum.html" class="bms-item">
        <svg viewBox="0 0 24 24"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
        <span>Форум</span>
      </a>
      <a href="/companies.html" class="bms-item">
        <svg viewBox="0 0 24 24"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/></svg>
        <span>Компании</span>
      </a>
      <a href="/communities.html" class="bms-item">
        <svg viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/></svg>
        <span>Сообщества</span>
      </a>
      <a href="/projects.html" class="bms-item">
        <svg viewBox="0 0 24 24"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
        <span>Проекты</span>
      </a>
      <a href="/events.html" class="bms-item">
        <svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
        <span>Мероприятия</span>
      </a>
      <a href="/jobs.html" class="bms-item">
        <svg viewBox="0 0 24 24"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/><line x1="12" y1="12" x2="12" y2="17"/><line x1="9" y1="14.5" x2="15" y2="14.5"/></svg>
        <span>Резюме</span>
      </a>
      <a href="/catalog.html" class="bms-item">
        <svg viewBox="0 0 24 24"><path d="M6 2L3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4z"/><line x1="3" y1="6" x2="21" y2="6"/><path d="M16 10a4 4 0 0 1-8 0"/></svg>
        <span>Товары и услуги</span>
      </a>
    </div>
    <div class="bms-footer">
      <a href="/settings.html" class="bms-settings">
        <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.6 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.6a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
        <span>Настройки</span>
      </a>
    </div>`;

  function letter(s) {
    const p = String(s || '').trim().split(/\s+/).filter(Boolean);
    if (!p.length) return '?';
    if (p.length >= 2) return (p[0][0] + p[1][0]).toUpperCase();
    return p[0][0].toUpperCase();
  }

  function colorFor(s) {
    const C = ['#5AB080','#3A90C0','#9060C0','#C07030','#1A8A6A','#B05090','#208090','#3B6D11','#633806','#185FA5'];
    return C[(String(s||'').charCodeAt(0)||0) % C.length];
  }

  function inject() {
    // 0. Bottom-nav (только если есть topbar — значит это shell-страница)
    injectBottomNav();
    // 1. Topbar
    const topbar = document.querySelector('header.topbar');
    if (topbar && !topbar.children.length) {
      topbar.innerHTML = TOPBAR_HTML;
    }
    // 2. Nav
    const nav = document.querySelector('nav.nav');
    if (nav && !nav.children.length) {
      nav.innerHTML = NAV_HTML;
      // Подсветить active по pathname
      const path = location.pathname;
      const items = nav.querySelectorAll('.nav-item[data-match]');
      let firstMatch = null;
      items.forEach(a => {
        try {
          const re = new RegExp(a.getAttribute('data-match'));
          if (re.test(path) && !firstMatch) firstMatch = a;
        } catch {}
      });
      if (firstMatch) firstMatch.classList.add('active');
    }
    // 3. Заполнить профиль из localStorage (мгновенно, без сети)
    populateUserFromCache();
    // 4. Обновить с сервера
    refreshUserFromServer();
    // 5. Подключить выпадашку
    bindDropdownHandlers();
  }

  function populateUserFromCache() {
    // Гость — не заполняем (guest-nav.js сам подставит «Войти/Регистрация»)
    let token = '';
    try { token = localStorage.getItem('token') || ''; } catch {}
    if (!token) return;
    let u = {};
    try { u = JSON.parse(localStorage.getItem('user') || '{}'); } catch {}
    const name = u.full_name || u.name || ((u.first_name || '') + ' ' + (u.last_name || '')).trim() || 'Пользователь';
    const role = u.position || u.role || 'Участник LASTOP';
    const ltr = letter(name);
    const color = colorFor(name);
    setText('topbarName', name);
    // Глобальный обработчик Enter в поиске → /search.html
    const searchInput = document.getElementById('searchInput');
    if (searchInput && !searchInput.dataset.searchBound) {
      searchInput.dataset.searchBound = '1';
      searchInput.addEventListener('keydown', function(e){
        if (e.key === 'Enter') {
          const q = e.target.value.trim();
          if (q) location.href = '/search.html?q=' + encodeURIComponent(q);
        }
      });
    }
    setText('topbarRole', role);
    setText('pddName', name);
    setText('pddRole', role);
    const avatarUrl = u.avatar_url || '';
    setAv('topbarAv', ltr, color, avatarUrl);
    setAv('pddAv', ltr, color, avatarUrl);
    loadDropdownCounters();
    // Если активен контекст компании/сообщества — применить его поверх личного
    // (profile-context.js может ещё не догрузиться — повторяем через RAF несколько раз)
    let _ctxTries = 0;
    function tryApplyCtx(){
      if (typeof window.__pcxApplyTopbar === 'function'){
        window.__pcxApplyTopbar();
      } else if (_ctxTries++ < 20){
        setTimeout(tryApplyCtx, 60);
      }
    }
    tryApplyCtx();
    // Подписка на изменения контекста — обновлять топбар без перезагрузки
    if (!window.__shellCtxBound){
      window.__shellCtxBound = true;
      window.addEventListener('lastop:context-changed', function(){
        if (typeof window.__pcxApplyTopbar === 'function') window.__pcxApplyTopbar();
      });
    }
  }

  async function loadDropdownCounters() {
    let token = '';
    try { token = localStorage.getItem('token') || ''; } catch {}
    if (!token) return;
    const headers = { Authorization: 'Bearer ' + token };
    try {
      const r = await fetch('/api/friends', { headers });
      if (r.ok) {
        const d = await r.json();
        const n = (d.friends || []).length;
        const el = document.getElementById('pddFriends');
        if (el) el.textContent = String(n);
      }
    } catch {}
    try {
      const r = await fetch('/api/communities?filter=my', { headers });
      if (r.ok) {
        const d = await r.json();
        const n = (d.communities || []).length;
        const el = document.getElementById('pddComm');
        if (el) el.textContent = String(n);
      }
    } catch {}
  }

  async function refreshUserFromServer() {
    let token = '';
    try { token = localStorage.getItem('token') || ''; } catch {}
    if (!token) return;
    try {
      const r = await fetch('/api/me', { headers: { Authorization: 'Bearer ' + token } });
      if (!r.ok) return;
      const { user } = await r.json();
      if (!user) return;
      try { localStorage.setItem('user', JSON.stringify(user)); } catch {}
      populateUserFromCache();
    } catch {}
  }

  function setText(id, v) {
    const el = document.getElementById(id);
    if (el) el.textContent = v;
  }
  // Глобальный helper — может вызываться и из shell-injector, и из страничных applyMe.
  // Хранит avatar URL для каждого id чтобы переживать перезаписи textContent.
  window.LASTOP_AVATARS = window.LASTOP_AVATARS || {};

  window.LASTOP_SET_AV = function (id, ltr, color, avatarUrl) {
    const el = document.getElementById(id);
    if (!el) return;
    // Запоминаем актуальный avatar_url для этого элемента.
    // Если avatarUrl не передан — пробуем взять из localStorage.user.avatar_url
    // (только для аватарок текущего пользователя: topbarAv, pddAv, composeAv, profileAv).
    if (avatarUrl !== undefined) {
      window.LASTOP_AVATARS[id] = avatarUrl || '';
    } else if (['topbarAv','pddAv','composeAv','profileAv'].includes(id)) {
      try {
        const cached = JSON.parse(localStorage.getItem('user') || 'null');
        if (cached && cached.avatar_url) {
          window.LASTOP_AVATARS[id] = cached.avatar_url;
        }
      } catch {}
    }
    const url = window.LASTOP_AVATARS[id] || '';
    // Базовая буква и цвет
    if (ltr !== undefined && ltr !== null) {
      // Записываем букву через textContent; если img снесут страничные скрипты,
      // восстановим его через MutationObserver ниже.
      el.textContent = ltr;
    }
    if (color) el.style.background = color;
    // Удаляем старый img
    const old = el.querySelector('img.lastop-av-img');
    if (old) old.remove();
    if (url) {
      el.style.position = el.style.position || 'relative';
      el.style.overflow = 'hidden';
      const img = document.createElement('img');
      img.className = 'lastop-av-img';
      img.src = url;
      img.alt = '';
      img.style.cssText = 'position:absolute;inset:0;width:100%;height:100%;object-fit:cover;border-radius:inherit;z-index:1;pointer-events:none';
      el.appendChild(img);
    }
  };

  // Локальная обёртка для совместимости
  function setAv(id, ltr, color, avatarUrl) {
    window.LASTOP_SET_AV(id, ltr, color, avatarUrl);
  }

  // Наблюдатель: если страничный applyMe перезаписал textContent (убив img),
  // восстанавливаем img из LASTOP_AVATARS.
  function watchAvatar(id) {
    const el = document.getElementById(id);
    if (!el || el._lastopWatched) return;
    el._lastopWatched = true;
    const observer = new MutationObserver(() => {
      const url = window.LASTOP_AVATARS[id] || '';
      if (!url) return;
      if (!el.querySelector('img.lastop-av-img')) {
        const img = document.createElement('img');
        img.className = 'lastop-av-img';
        img.src = url;
        img.alt = '';
        img.style.cssText = 'position:absolute;inset:0;width:100%;height:100%;object-fit:cover;border-radius:inherit;z-index:1;pointer-events:none';
        el.style.position = el.style.position || 'relative';
        el.style.overflow = 'hidden';
        el.appendChild(img);
      }
    });
    observer.observe(el, { childList: true, characterData: true, subtree: true });
  }

  // Запускаем наблюдатель сразу при загрузке (без ожидания /api/me)
  function bootstrapAvatarWatchers() {
    const ids = ['topbarAv', 'pddAv', 'composeAv', 'profileAv', 'heroAv'];
    ids.forEach(watchAvatar);
    // Если в localStorage уже есть user — мгновенно ставим аватарку без ожидания /me
    try {
      const cached = JSON.parse(localStorage.getItem('user') || 'null');
      if (cached && cached.avatar_url) {
        ids.forEach(id => {
          window.LASTOP_AVATARS[id] = cached.avatar_url;
          // Принудительно вставляем img сразу — даже если страничный applyMe
          // уже отработал и стёр содержимое (textContent='НК').
          const el = document.getElementById(id);
          if (el && !el.querySelector('img.lastop-av-img')) {
            const img = document.createElement('img');
            img.className = 'lastop-av-img';
            img.src = cached.avatar_url;
            img.alt = '';
            img.style.cssText = 'position:absolute;inset:0;width:100%;height:100%;object-fit:cover;border-radius:inherit;z-index:1;pointer-events:none';
            el.style.position = el.style.position || 'relative';
            el.style.overflow = 'hidden';
            el.appendChild(img);
          }
        });
      }
    } catch {}
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', bootstrapAvatarWatchers);
  } else {
    bootstrapAvatarWatchers();
  }

  function bindDropdownHandlers() {
    if (!window.toggleProfileDD) {
      window.toggleProfileDD = function (e) {
        if (e && e.stopPropagation) e.stopPropagation();
        const dd = document.getElementById('profileDD');
        const tp = document.getElementById('topbarProfile');
        if (dd) dd.classList.toggle('open');
        if (tp) tp.classList.toggle('open');
      };
    }
    if (!window.__lastopDDBound) {
      window.__lastopDDBound = true;
      document.addEventListener('click', function (e) {
        const dd = document.getElementById('profileDD');
        const tp = document.getElementById('topbarProfile');
        if (!dd || !tp) return;
        if (dd.contains(e.target) || tp.contains(e.target)) return;
        dd.classList.remove('open');
        tp.classList.remove('open');
      });
      document.addEventListener('keydown', function (e) {
        if (e.key !== 'Escape') return;
        const dd = document.getElementById('profileDD');
        const tp = document.getElementById('topbarProfile');
        if (dd) dd.classList.remove('open');
        if (tp) tp.classList.remove('open');
      });
    }
    // logout fallback (если страница не определила свою)
    if (!window.logout) {
      window.logout = function () {
        try { localStorage.removeItem('token'); localStorage.removeItem('user'); } catch {}
        location.href = '/home-guest.html';
      };
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', inject);
  } else {
    inject();
  }

  // ════════════════════════════════════════════════════════════
  //  ГОРИЗОНТАЛЬНЫЙ СКРОЛЛ КОЛЕСОМ МЫШИ — глобально для платформы
  // ════════════════════════════════════════════════════════════
  // Преобразует вертикальное колесо мыши (deltaY) в горизонтальный
  // скролл для контейнеров, где overflow-x = auto|scroll.
  // Тачпад с реальным горизонтальным жестом (deltaX != 0) не трогаем —
  // он работает нативно.
  function lastopBindHorizWheel(el){
    if (!el || el.dataset.hwBound) return;
    const cs = window.getComputedStyle(el);
    const ox = cs.overflowX;
    if (ox !== 'auto' && ox !== 'scroll') return;
    // Подключаем только если действительно есть что скроллить (или может появиться позже).
    el.dataset.hwBound = '1';
    el.dataset.hw = '1';
    el.addEventListener('wheel', function(e){
      // Реальный горизонтальный жест (тачпад) — не вмешиваемся
      if (Math.abs(e.deltaX) > Math.abs(e.deltaY)) return;
      // Нечего скроллить — пусть событие идёт родителю
      if (el.scrollWidth <= el.clientWidth) return;
      e.preventDefault();
      el.scrollLeft += e.deltaY;
    }, { passive: false });
  }

  // Сканер: ищет все потенциально-горизонтальные контейнеры на странице
  function lastopScanHorizContainers(root){
    if (!root || !root.querySelectorAll) return;
    // Ограничиваем поиск разумным набором — все элементы с inline/computed overflow-x
    // были бы дороги, поэтому идём по селекторам с обозначенным горизонтальным
    // скроллом + общая маркировка data-horiz-scroll для опт-ин.
    const SELECTORS = [
      '[data-horiz-scroll]',
      '.d-tabs', '.ip-tabs',           // chat
      '.proj-tabs',                     // projects
      '.comm-tabs',                     // communities
      '.s-tabs-row',                    // search
      '.filter-tabs',                   // jobs/events/forum
      '.tabs-row',                      // jobs
      '.nav-pills',                     // generic
      '.scroll-x'                       // generic
    ];
    let nodes;
    try { nodes = root.querySelectorAll(SELECTORS.join(',')); }
    catch (e) { return; }
    for (let i = 0; i < nodes.length; i++) lastopBindHorizWheel(nodes[i]);
  }

  // Начальное сканирование + наблюдение за новыми элементами
  function lastopStartHorizWatcher(){
    lastopScanHorizContainers(document.body);
    if (window.MutationObserver && !window.__lastopHWObs){
      try {
        const obs = new MutationObserver(function(muts){
          for (let i = 0; i < muts.length; i++){
            const added = muts[i].addedNodes;
            for (let j = 0; j < added.length; j++){
              const n = added[j];
              if (n.nodeType !== 1) continue;
              lastopScanHorizContainers(n);
              // Если сам добавленный узел подходит — обработаем напрямую
              try { lastopBindHorizWheel(n); } catch(e){}
            }
          }
        });
        obs.observe(document.body, { childList: true, subtree: true });
        window.__lastopHWObs = obs;
      } catch (e) {}
    }
  }

  // Экспорт хелпера для опт-ин из страничного кода
  window.LASTOP_attachHorizWheel = lastopBindHorizWheel;

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', lastopStartHorizWatcher);
  } else {
    lastopStartHorizWatcher();
  }

  // ─────────────────────────────────────────────────────────────
  // BOTTOM-NAV: инжектируется один раз в конец <body>.
  // Виден только на ≤640px (стили в mobile-v3.css).
  // ─────────────────────────────────────────────────────────────
  function injectBottomNav() {
    // Только для shell-страниц (где есть topbar или nav)
    if (!document.querySelector('header.topbar') && !document.querySelector('nav.nav')) return;
    // Только один раз
    if (document.querySelector('nav.bottom-nav')) return;

    // Bottom-nav
    const bn = document.createElement('nav');
    bn.className = 'bottom-nav';
    bn.setAttribute('aria-label', 'Нижняя навигация');
    bn.innerHTML = BOTTOM_NAV_HTML;
    document.body.appendChild(bn);

    // Bottom-menu overlay + sheet
    const ov = document.createElement('div');
    ov.className = 'bottom-menu-overlay';
    ov.id = 'lastopBottomMenuOverlay';
    ov.onclick = lastopCloseBottomMenu;
    document.body.appendChild(ov);

    const sheet = document.createElement('div');
    sheet.className = 'bottom-menu-sheet';
    sheet.id = 'lastopBottomMenuSheet';
    sheet.setAttribute('role', 'dialog');
    sheet.setAttribute('aria-label', 'Все разделы');
    sheet.innerHTML = BOTTOM_MENU_HTML;
    document.body.appendChild(sheet);

    // Подсветка активного пункта по pathname
    const path = location.pathname;
    bn.querySelectorAll('.bn-item[data-match]').forEach(a => {
      try {
        const re = new RegExp(a.getAttribute('data-match'));
        if (re.test(path)) a.classList.add('active');
      } catch(e) {}
    });

    // Badge для чата — если есть свежие непрочитанные сообщения
    try {
      const tk = localStorage.getItem('token');
      if (tk) {
        fetch('/api/chat/unread-count', { headers: { Authorization: 'Bearer ' + tk } })
          .then(r => r.ok ? r.json() : null)
          .then(d => {
            const cnt = (d && (d.count || d.unread_count)) || 0;
            const badge = document.getElementById('bnChatBadge');
            if (badge) {
              if (cnt > 0) {
                badge.textContent = cnt > 99 ? '99+' : String(cnt);
                badge.style.display = 'grid';
              }
            }
          })
          .catch(()=>{});
      }
    } catch(e) {}

    // Закрытие меню по Escape
    document.addEventListener('keydown', function(e){
      if (e.key === 'Escape') lastopCloseBottomMenu();
    });

    // Закрытие меню при свайпе вниз (touch)
    let touchStartY = 0;
    sheet.addEventListener('touchstart', function(e){
      touchStartY = e.touches[0].clientY;
    }, { passive: true });
    sheet.addEventListener('touchend', function(e){
      const dy = e.changedTouches[0].clientY - touchStartY;
      if (dy > 80) lastopCloseBottomMenu();
    }, { passive: true });
  }

  // Открыть модалку создания контекстно для текущей страницы
  function lastopOpenCreateAction() {
    var path = location.pathname || '';
    // Маппинг: страница → имя функции открытия модалки на этой странице
    var map = [
      [/^\/companies\.html/,  ['openCreateModal','openAddCompanyModal','openCompanyModal']],
      [/^\/projects\.html/,   ['openCreateModal','openCreateProjectModal','openProjectModal']],
      [/^\/events\.html/,     ['openCreateModal','openCreateEventModal','openEventModal']],
      [/^\/jobs\.html/,       ['openCreateModal','openCreateJobModal','openJobModal']],
      [/^\/forum\.html/,      ['openCreateTopic','openCreateModal','openNewTopicModal']],
      [/^\/communities\.html/,['openCreateModal','openCreateCommunityModal']],
      [/^\/catalog\.html/,    ['openCreateModal','openCreateItemModal']],
      [/^\/saved\.html/,      []],
      [/^\/dashboard\.html/,  ['openCreatePost','openCreateModal']],
      [/^\/(home-auth\.html|index_.*\.html|)?$/, ['openCreatePost']],
    ];
    for (var i = 0; i < map.length; i++) {
      if (map[i][0].test(path)) {
        var fns = map[i][1];
        for (var j = 0; j < fns.length; j++) {
          if (typeof window[fns[j]] === 'function') {
            try { window[fns[j]](); return; } catch(e) {}
          }
        }
        break;
      }
    }
    // Фоллбек: пытаемся кликнуть оригинальную .btn-create на странице
    var fallbackBtn = document.querySelector('.btn-create, .btn-create-job, [data-action="create"]');
    if (fallbackBtn) { fallbackBtn.click(); return; }
    // Иначе — открываем Меню sheet (юзер хотя бы выберет раздел)
    if (typeof lastopOpenBottomMenu === 'function') lastopOpenBottomMenu();
  }
  window.lastopOpenCreateAction = lastopOpenCreateAction;

  // Открыть/закрыть Меню sheet
  function lastopOpenBottomMenu() {
    const ov = document.getElementById('lastopBottomMenuOverlay');
    const sh = document.getElementById('lastopBottomMenuSheet');
    if (ov) ov.classList.add('open');
    if (sh) sh.classList.add('open');
  }
  function lastopCloseBottomMenu() {
    const ov = document.getElementById('lastopBottomMenuOverlay');
    const sh = document.getElementById('lastopBottomMenuSheet');
    if (ov) ov.classList.remove('open');
    if (sh) sh.classList.remove('open');
  }

  // Экспорт в global scope для onclick
  window.lastopOpenBottomMenu = lastopOpenBottomMenu;
  window.lastopCloseBottomMenu = lastopCloseBottomMenu;
})();
