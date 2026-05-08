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
    <svg viewBox="0 0 24 24"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/></svg>
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
    setAv('topbarAv', ltr, color);
    setAv('pddAv', ltr, color);
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
  function setAv(id, ltr, color) {
    const el = document.getElementById(id);
    if (!el) return;
    el.textContent = ltr;
    el.style.background = color;
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
})();
