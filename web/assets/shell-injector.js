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
    <span>Карьера</span>
  </a>
  <div class="nav-group">Торговая площадка</div>
  <a class="nav-item" href="/catalog.html" data-match="^/catalog|^/product-detail|^/service-detail">
    <svg viewBox="0 0 24 24"><path d="M6 2L3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4z"/><line x1="3" y1="6" x2="21" y2="6"/><path d="M16 10a4 4 0 0 1-8 0"/></svg>
    <span>Товары и услуги</span>
  </a>
  <a class="nav-item" href="/emarket.html" data-match="^/emarket">
    <svg viewBox="0 0 24 24"><path d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.3 2.3c-.6.6-.2 1.7.7 1.7H17"/><circle cx="9" cy="20" r="1"/><circle cx="17" cy="20" r="1"/></svg>
    <span>E-market</span>
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
    <span class="nav-badge" id="navChatUnread" style="display:none"></span>
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
    // Гость — не заполняем (guest-nav.js сам подставит «Войти/Регистрация»).
    // Phase 3 cookie-only: единственный признак логина — наличие user
    // в localStorage. Прежняя проверка localStorage.token больше не
    // работает (token cookie теперь httpOnly, в localStorage его нет).
    let u = {};
    try { u = JSON.parse(localStorage.getItem('user') || '{}'); } catch {}
    if (!u || !(u.id || u.public_id)) return;
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
    loadChatUnread();
    if (!window.__shellChatUnreadBound) {
      window.__shellChatUnreadBound = true;
      setInterval(function(){ if (!document.hidden) loadChatUnread(); }, 30000);
      document.addEventListener('visibilitychange', function(){ if (!document.hidden) loadChatUnread(); });
    }
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
        loadChatUnread();
      });
    }
  }

  async function loadChatUnread() {
    var hasUser = false;
    try { hasUser = !!JSON.parse(localStorage.getItem('user') || '{}').id; } catch {}
    if (!hasUser) return;
    try {
      const r = await lastopFetch('/api/chat/unread_total', { headers:{} });
      if (!r.ok) return;
      const d = await r.json();
      const n = (d && typeof d.count === 'number') ? d.count : 0;
      setChatUnreadBadge(n);
    } catch {}
  }
  function setChatUnreadBadge(n) {
    const el = document.getElementById('navChatUnread');
    if (!el) return;
    if (n > 0) {
      el.textContent = n > 99 ? '99+' : String(n);
      el.style.display = '';
    } else {
      el.textContent = '';
      el.style.display = 'none';
    }
  }
  window.lastopSetChatUnread = setChatUnreadBadge;
  window.lastopRefreshChatUnread = loadChatUnread;

  async function loadDropdownCounters() {
    var hasUser = false;
    try { hasUser = !!JSON.parse(localStorage.getItem('user') || '{}').id; } catch {}
    if (!hasUser) return;
    const headers={};
    try {
      const r = await lastopFetch('/api/friends', { headers });
      if (r.ok) {
        const d = await r.json();
        const n = (d.friends || []).length;
        const el = document.getElementById('pddFriends');
        if (el) el.textContent = String(n);
      }
    } catch {}
    try {
      const r = await lastopFetch('/api/communities?tab=my', { headers });
      if (r.ok) {
        const d = await r.json();
        const n = (d.communities || []).length;
        const el = document.getElementById('pddComm');
        if (el) el.textContent = String(n);
      }
    } catch {}
  }

  async function refreshUserFromServer() {
    var hasUser = false;
    try { hasUser = !!JSON.parse(localStorage.getItem('user') || '{}').id; } catch {}
    if (!hasUser) return;
    try {
      const r = await lastopFetch('/api/me', { headers:{} });
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
      window.logout = async function () {
        try { await lastopFetch('/api/auth/logout', { method: 'POST' }); } catch {}
        try { localStorage.removeItem('user'); } catch {}
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

  // ── Отключение разделов (техработы) ───────────────────────────
  function lastopCheckSection() {
    var SECTIONS = [
      { k: 'dashboard', l: 'Новости', re: /^\/dashboard|^\/news-detail/ },
      { k: 'projects', l: 'Проекты', re: /^\/projects|^\/project-detail/ },
      { k: 'events', l: 'Мероприятия', re: /^\/events|^\/event-detail/ },
      { k: 'exhibitions', l: 'Выставки', re: /^\/exhibition/ },
      { k: 'jobs', l: 'Карьера', re: /^\/jobs|^\/job-detail|^\/resume-detail/ },
      { k: 'catalog', l: 'Товары и услуги', re: /^\/catalog|^\/product-detail|^\/service-detail/ },
      { k: 'forum', l: 'Форум', re: /^\/forum/ },
      { k: 'companies', l: 'Компании', re: /^\/companies|^\/company-detail|^\/my-company/ },
      { k: 'communities', l: 'Сообщества', re: /^\/communities|^\/community-detail|^\/my-community/ },
      { k: 'chat', l: 'Чат', re: /^\/chat/ }
    ];
    var path = location.pathname;
    var cur = null;
    for (var i = 0; i < SECTIONS.length; i++) {
      if (SECTIONS[i].re.test(path)) { cur = SECTIONS[i]; break; }
    }
    if (!cur) return; // раздел не отключаемый (главная/профиль/настройки и т.д.)
    // Админы и владельцы проходят
    var isPriv = false;
    try {
      var u = JSON.parse(localStorage.getItem('user') || '{}');
      isPriv = !!(u.is_admin || u.is_owner);
    } catch (e) {}
    fetch('/api/sections/status', { credentials: 'include' })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (d) {
        var off = (d && d.off) || {};
        if (!off[cur.k]) return;
        var main = document.querySelector('.main') || document.querySelector('main');
        if (!main) return;
        // Админы/владельцы: раздел работает, но сверху баннер-напоминание
        if (isPriv) {
          if (document.getElementById('lastopSectionWarn')) return;
          var warn = document.createElement('div');
          warn.id = 'lastopSectionWarn';
          warn.style.cssText = 'background:#FFF6E6;border:1.5px solid #F0DCA0;border-radius:12px;padding:12px 16px;margin-bottom:10px;display:flex;align-items:center;gap:10px;flex-shrink:0';
          warn.innerHTML =
            '<span style="font-size:20px;flex-shrink:0">⚠️</span>' +
            '<div style="flex:1;min-width:0"><div style="font-size:13px;font-weight:700;color:#8A5A00">Раздел «' + cur.l + '» выключен для пользователей</div>' +
            '<div style="font-size:12px;color:#8A5A00;opacity:.8;margin-top:1px">Вы видите его как администратор. Пользователям показывается плашка о техработах — не забудьте включить раздел обратно.</div></div>' +
            '<a href="/admin.html" style="flex-shrink:0;padding:7px 14px;border-radius:8px;background:#8A5A00;color:#fff;text-decoration:none;font-size:12px;font-weight:600;white-space:nowrap">Консоль разделов</a>';
          main.insertBefore(warn, main.firstChild);
          return;
        }
        main.innerHTML =
          '<div style="display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center;padding:60px 24px;min-height:60vh">' +
            '<div style="font-size:54px;margin-bottom:18px">🛠️</div>' +
            '<div style="font-size:22px;font-weight:800;color:var(--t,#1A2A22);margin-bottom:10px">Раздел «' + cur.l + '» временно недоступен</div>' +
            '<div style="font-size:15px;color:var(--gmt,#5A8A6A);max-width:440px;line-height:1.6">Ведём технические работы — скоро всё заработает. Спасибо за терпение!</div>' +
            '<div style="font-size:13px;color:var(--gmt,#5A8A6A);margin-top:8px;opacity:.7">Приносим извинения за доставленные неудобства.</div>' +
            '<a href="/" style="margin-top:24px;padding:11px 24px;border-radius:12px;background:var(--g,#1E8A4C);color:#fff;text-decoration:none;font-weight:600;font-size:14px">На главную</a>' +
          '</div>';
      })
      .catch(function () {});
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', lastopCheckSection);
  } else {
    lastopCheckSection();
  }

})();
