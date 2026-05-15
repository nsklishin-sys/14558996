/* ═══════════════════════════════════════════════════════════════
   mobile-injector.js — Мобильная JS-логика LASTOP GROUP
   ═══════════════════════════════════════════════════════════════
   Версия: 3.0 (вторая попытка, начата 15.05.26)
   Killswitch: env MOBILE_V3_DISABLE=1 + restart lastop сервиса

   ПРАВИЛА:
   - Файл загружается defer, после DOMContentLoaded инициализация
   - Все мобильные манипуляции DOM ТОЛЬКО отсюда
   - НЕ редактировать shell-injector.js под мобильные нужды
   - При width > 640 — ранний return (десктоп нетронут)

   См. также:
   - web/assets/mobile-v3.css — мобильные стили
   - cmd/server/main.go renderHTMLInject() — точка подключения
   - Obsidian: 07 — Правила работы с Codex (раздел «📱»)
   - Obsidian: 23 — Дорожная карта мобильной адаптации
   ═══════════════════════════════════════════════════════════════ */

(function() {
  'use strict';

  // Killswitch на стороне клиента — на случай если CSS подгружен,
  // а JS почему-то нужно дополнительно проверить.
  // По умолчанию пусто — на каждом этапе добавляются модули.

  // Ранний выход для десктопа — экономим работу
  if (window.innerWidth > 640) {
    return;
  }

  // Проверка наличия токена — без него мы на гостевой странице.
  // Гостям мобильную навигацию не показываем (bottom-nav и sheet-меню
  // ведут на разделы для авторизованных).
  var hasToken = false;
  try {
    var raw = localStorage.getItem('token') || '';
    hasToken = raw.trim().length > 0;
  } catch (_) { /* localStorage недоступен — считаем гостем */ }

  if (!hasToken) {
    // На /home-guest.html у гостя на мобиле — редирект на /login.html.
    // Гостевая главная сейчас не адаптирована под мобилу, временно
    // показываем сразу login (компактная страница которая работает).
    // /login.html, /register.html и другие auth-страницы оставляем
    // как есть — на них bottom-nav просто не появится.
    var path = location.pathname || '';
    if (path === '/home-guest.html') {
      location.replace('/login.html');
      return;
    }
    // Прочие гостевые: ничего не инжектируем, выходим тихо.
    return;
  }

  // === Э2: bottom-nav + sheet-меню ============================

  // 5 пунктов bottom-nav: Главная · Меню · Уведомления · Чат · Профиль.
  // data-match — regex по pathname для подсветки активного пункта.
  const BOTTOM_NAV_HTML =
    '<nav class="m-bn" id="mobileBottomNav" aria-label="Мобильная навигация">' +
      '<a class="m-bn-item" href="/" data-match="^/$|^/home-auth\.html$">' +
        '<svg class="m-bn-ico" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9,22 9,12 15,12 15,22"/></svg>' +
        '<span class="m-bn-lbl">Главная</span>' +
      '</a>' +
      '<button type="button" class="m-bn-item m-bn-menu" onclick="lastopMobileOpenMenu()" aria-label="Открыть меню">' +
        '<svg class="m-bn-ico" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>' +
        '<span class="m-bn-lbl">Меню</span>' +
      '</button>' +
      '<a class="m-bn-item" href="/notifications.html" data-match="^/notifications">' +
        '<svg class="m-bn-ico" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>' +
        '<span class="m-bn-lbl">Уведомления</span>' +
      '</a>' +
      '<a class="m-bn-item" href="/chat.html" data-match="^/chat">' +
        '<svg class="m-bn-ico" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><polyline points="2,4 12,13 22,4"/></svg>' +
        '<span class="m-bn-lbl">Чат</span>' +
      '</a>' +
      '<a class="m-bn-item" href="/profile.html" data-match="^/profile">' +
        '<svg class="m-bn-ico" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>' +
        '<span class="m-bn-lbl">Профиль</span>' +
      '</a>' +
    '</nav>';

  // Sheet-меню: остальные разделы (открывается тапом на «Меню» в bottom-nav).
  // Структура зеркалит NAV_HTML из shell-injector.js минус 4 пункта которые уже в bottom-nav.
  const MENU_SHEET_HTML =
    '<div class="m-menu-bg" id="mobileMenuBg" onclick="lastopMobileCloseMenu()"></div>' +
    '<aside class="m-menu" id="mobileMenu" role="menu" aria-label="Меню разделов">' +
      '<div class="m-menu-handle"></div>' +
      '<div class="m-menu-title">Все разделы</div>' +
      '<a class="m-menu-item" href="/dashboard.html"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M4 6h16M4 12h16M4 18h10"/></svg>Новости</a>' +
      '<a class="m-menu-item" href="/projects.html"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>Проекты</a>' +
      '<a class="m-menu-item" href="/events.html"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>Мероприятия</a>' +
      '<a class="m-menu-item" href="/exhibitions.html"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7"/><rect x="5" y="9" width="14" height="12" rx="1"/><path d="M9 21V13h6v8"/></svg>Выставки</a>' +
      '<a class="m-menu-item" href="/jobs.html"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><circle cx="12" cy="13" r="2"/><path d="M8 19c0-2.2 1.8-4 4-4s4 1.8 4 4"/></svg>Резюме</a>' +
      '<a class="m-menu-item" href="/catalog.html"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M6 2L3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4z"/><line x1="3" y1="6" x2="21" y2="6"/><path d="M16 10a4 4 0 0 1-8 0"/></svg>Товары и услуги</a>' +
      '<a class="m-menu-item" href="/forum.html"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>Форум</a>' +
      '<a class="m-menu-item" href="/companies.html"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/></svg>Компании</a>' +
      '<a class="m-menu-item" href="/communities.html"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>Сообщества</a>' +
      '<a class="m-menu-item m-menu-settings" href="/settings.html"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>Настройки</a>' +
    '</aside>';

  // Глобальные функции для onclick-обработчиков (вызываются из HTML атрибутов).
  window.lastopMobileOpenMenu = function() {
    const bg = document.getElementById('mobileMenuBg');
    const menu = document.getElementById('mobileMenu');
    if (bg) bg.classList.add('m-open');
    if (menu) menu.classList.add('m-open');
    document.documentElement.classList.add('m-menu-locked');
  };
  window.lastopMobileCloseMenu = function() {
    const bg = document.getElementById('mobileMenuBg');
    const menu = document.getElementById('mobileMenu');
    if (bg) bg.classList.remove('m-open');
    if (menu) menu.classList.remove('m-open');
    document.documentElement.classList.remove('m-menu-locked');
  };

  // Подсветка активного пункта bottom-nav по pathname.
  function highlightActive() {
    const path = location.pathname || '/';
    const items = document.querySelectorAll('.m-bn-item[data-match]');
    items.forEach(function(it) {
      const m = it.getAttribute('data-match');
      try {
        if (new RegExp(m).test(path)) it.classList.add('m-bn-active');
      } catch (_) { /* битый regex — игнор */ }
    });
  }

  // Инжекция HTML и инициализация.
  function init() {
    // Защита от двойной инициализации (если скрипт вдруг подключён дважды).
    if (document.getElementById('mobileBottomNav')) return;
    document.body.insertAdjacentHTML('beforeend', BOTTOM_NAV_HTML + MENU_SHEET_HTML);
    highlightActive();

    // Закрытие меню по Escape.
    document.addEventListener('keydown', function(e) {
      if (e.key === 'Escape') window.lastopMobileCloseMenu();
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

})();
