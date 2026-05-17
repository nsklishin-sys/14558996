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

  // === Этапы 2+ постепенно добавляют функции сюда ===

    // ─── Э8: глобальная контекстная кнопка действия в топбаре ───
  // Инжектируется в .profile-dd-wrap (рядом с круглым аватаром) на
  // списочных страницах. При тапе программно кликает родную кнопку
  // страницы — её action (модалка/переход) исполнится сам по себе.
  (function initFabAction() {
    // Словарь: path → массив селекторов-кандидатов кнопки действия.
    // Первый найденный (и видимый) — целевой. Порядок важен:
    // на exhibitions сначала ищем «Заявка на стенд» (.btn-add),
    // потом «Создать выставку» (#btnCreateExhibition).
    var PAGE_ACTIONS = {
      '/companies.html':   ['.toolbar a.btn-create', '.toolbar .btn-create'],
      '/projects.html':    ['.proj-toolbar .btn-create'],
      '/events.html':      ['.ev-toolbar .btn-create', '.toolbar .btn-create'],
      '/exhibitions.html': ['.ex-toolbar .btn-add', '.ex-toolbar #btnCreateExhibition'],
      '/jobs.html':        ['.jobs-toolbar .btn-create-job'],
      '/forum.html':       ['.forum-toolbar .btn-create'],
      '/communities.html': ['.comm-toolbar .btn-create'],
      '/catalog.html':     ['.cat-toolbar .btn-create'],
      '/dashboard.html':   ['.news-toolbar .btn-write']
    };

    var path = location.pathname || '';
    var selectors = PAGE_ACTIONS[path];
    if (!selectors) return; // эта страница без FAB

    // Найти первую видимую кнопку из списка кандидатов.
    function findTargetBtn() {
      for (var i = 0; i < selectors.length; i++) {
        var nodes = document.querySelectorAll(selectors[i]);
        for (var j = 0; j < nodes.length; j++) {
          var el = nodes[j];
          // display:none → пропускаем (например, у юзера нет прав).
          // visibility:hidden и offsetParent тоже учитываем.
          if (el.offsetParent !== null || el.getClientRects().length > 0) {
            return el;
          }
        }
      }
      return null;
    }

    // Получить читаемый лейбл из целевой кнопки для aria-label/title.
    function readLabel(btn) {
      if (!btn) return 'Создать';
      // Клонируем без svg, читаем текст.
      var clone = btn.cloneNode(true);
      var svg = clone.querySelector('svg');
      if (svg) svg.remove();
      var txt = (clone.textContent || '').trim();
      return txt || 'Создать';
    }

    // Инжектировать FAB в .profile-dd-wrap (левее .topbar-profile).
    function injectFab(targetBtn) {
      var wrap = document.querySelector('.topbar .profile-dd-wrap');
      if (!wrap) return null;
      if (wrap.querySelector('.m-fab-action')) return wrap.querySelector('.m-fab-action');

      var fab = document.createElement('button');
      fab.type = 'button';
      fab.className = 'm-fab-action';
      fab.setAttribute('aria-label', readLabel(targetBtn));
      fab.setAttribute('title', readLabel(targetBtn));
      fab.innerHTML =
        '<svg viewBox="0 0 24 24"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>';

      // Программный клик по родной кнопке. На <a> .click() также
      // отрабатывает href, на <button> срабатывает onclick.
      fab.addEventListener('click', function(e) {
        e.preventDefault();
        e.stopPropagation();
        // Перечитываем target — мог поменяться (например, toggle на jobs).
        var current = findTargetBtn();
        if (current) {
          // Обновляем лейбл по факту тапа — на jobs/catalog текст мог
          // поменяться после переключения toggle.
          fab.setAttribute('aria-label', readLabel(current));
          fab.setAttribute('title', readLabel(current));
          current.click();
        }
      });

      // FAB вставляется первым ребёнком .profile-dd-wrap (левее аватара).
      wrap.insertBefore(fab, wrap.firstChild);
      // Помечаем <html> что FAB смонтирован — CSS скроет родную кнопку.
      document.documentElement.classList.add('m-fab-mounted');
      return fab;
    }

    // Попытка маунта с ретраями: некоторые страницы рендерят
    // .btn-create через JS после загрузки данных.
    var attempts = 0;
    var MAX_ATTEMPTS = 20; // 20 × 200мс = 4 секунды максимум

    function tryMount() {
      var btn = findTargetBtn();
      if (btn) {
        injectFab(btn);
        return;
      }
      attempts++;
      if (attempts < MAX_ATTEMPTS) {
        setTimeout(tryMount, 200);
      }
    }

    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', tryMount);
    } else {
      tryMount();
    }
  })();

// ─── Э10: чат на мобиле — двухэкранный iOS-паттерн ───
  // Список диалогов и открытый чат — два full-width view, переключаются
  // классами .m-show-dialogs / .m-show-chat на .chat-panel.
  (function initMobileChat() {
    if (location.pathname !== '/chat.html') return;

    function init() {
      var panel = document.querySelector('.chat-panel');
      if (!panel) return;
      // Защита от двойной инициализации.
      if (panel.classList.contains('m-show-dialogs') ||
          panel.classList.contains('m-show-chat')) return;

      // Дефолтное состояние: показываем список диалогов.
      // Если URL содержит ?user=X — открыли чат с конкретным юзером,
      // сразу показываем экран чата.
      var params = new URLSearchParams(location.search);
      var hasOpenChat = params.has('user') || params.has('chat') || params.has('id');
      panel.classList.add(hasOpenChat ? 'm-show-chat' : 'm-show-dialogs');

      // Делегирование клика по диалогам в списке.
      // Селектор `.dialog` или `.d-item` — берём широкий, ищем
      // ближайший li/div который и есть «диалог».
      var dialogsList = panel.querySelector('.dialogs');
      if (dialogsList) {
        dialogsList.addEventListener('click', function(e) {
          // Не реагируем на клики по поиску/настройкам внутри списка.
          if (e.target.closest('input, button.dialogs-head-btn')) return;
          // Любой клик внутри ".dialogs" по дочернему элементу-диалогу.
          // Существующий JS chat.html сам обработает onclick загрузки сообщений.
          // Мы только переключаем view, и только если действительно был
          // выбран диалог (есть кликнутый элемент с data-id или onclick="loadDialog").
          var target = e.target.closest('.d-item, [data-conv-id], [data-user-id], [onclick*="selectConversation"], [onclick*="loadDialog"], [onclick*="openChat"]');
          if (target) {
            // Даём существующему обработчику отработать первым.
            setTimeout(function() {
              panel.classList.remove('m-show-dialogs');
              panel.classList.add('m-show-chat');
            }, 0);
          }
        });
      }

      // Инжектируем кнопку «Назад» в .chat-head (левее аватара).
      var chatHead = panel.querySelector('.chat-head');
      if (chatHead && !chatHead.querySelector('.m-chat-back')) {
        var backBtn = document.createElement('button');
        backBtn.type = 'button';
        backBtn.className = 'm-chat-back';
        backBtn.setAttribute('aria-label', 'Назад к списку диалогов');
        backBtn.innerHTML =
          '<svg viewBox="0 0 24 24"><polyline points="15 18 9 12 15 6"/></svg>';
        backBtn.addEventListener('click', function() {
          panel.classList.remove('m-show-chat');
          panel.classList.add('m-show-dialogs');
        });
        chatHead.insertBefore(backBtn, chatHead.firstChild);
      }
    }

    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', init);
    } else {
      init();
    }
  })();

// ─── Э7.1: settings — dropdown-селектор разделов на мобиле ───
  // Триггер инжектируется в начало .settings-body. Тап на триггер →
  // .settings-menu получает класс .m-open и выезжает сверху.
  // Тап на пункт меню → выпадашка закрывается, switchPanel срабатывает
  // через свой собственный обработчик (мы не вмешиваемся в его логику).
  (function initSettingsDropdown() {
    if (location.pathname !== '/settings.html') return;

    function init() {
      var body = document.querySelector('.settings-body');
      var menu = document.querySelector('.settings-menu');
      if (!body || !menu) return;
      // Защита от двойной инициализации.
      if (body.querySelector('.m-set-trigger')) return;

      // Создаём триггер. Метка обновляется из активного пункта меню.
      var trigger = document.createElement('button');
      trigger.type = 'button';
      trigger.className = 'm-set-trigger';
      trigger.innerHTML =
        '<svg class="m-set-trigger-ico" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>' +
        '<span class="m-set-trigger-lbl">Профиль</span>' +
        '<svg class="m-set-trigger-chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>';

      // Backdrop для затемнения фона и закрытия по тапу вне выпадашки.
      var backdrop = document.createElement('div');
      backdrop.className = 'm-set-backdrop';

      body.insertBefore(trigger, body.firstChild);
      body.appendChild(backdrop);

      // Обновить лейбл триггера из текущего активного пункта.
      function syncTriggerLabel() {
        var active = menu.querySelector('.sm-item.active');
        if (!active) return;
        var lblEl = trigger.querySelector('.m-set-trigger-lbl');
        if (!lblEl) return;
        // Берём текст без иконки — клонируем и удаляем svg.
        var clone = active.cloneNode(true);
        var svg = clone.querySelector('svg');
        if (svg) svg.remove();
        lblEl.textContent = (clone.textContent || '').trim();
      }
      syncTriggerLabel();

      function open() {
        menu.classList.add('m-open');
        trigger.classList.add('m-open');
        backdrop.classList.add('m-open');
      }
      function close() {
        menu.classList.remove('m-open');
        trigger.classList.remove('m-open');
        backdrop.classList.remove('m-open');
      }

      // Тап на триггер → открыть/закрыть.
      trigger.addEventListener('click', function(e) {
        e.stopPropagation();
        if (menu.classList.contains('m-open')) close();
        else open();
      });

      // Тап на backdrop → закрыть.
      backdrop.addEventListener('click', close);

      // Тап на пункт меню → закрыть выпадашку и обновить лейбл.
      // Используем capture-фазу через document, чтобы наш обработчик
      // сработал параллельно с существующим switchPanel.
      menu.addEventListener('click', function(e) {
        var item = e.target.closest('.sm-item[data-panel]');
        if (!item) return;
        // Даём switchPanel выполниться (он навешен в settings.html),
        // потом синхронизируем лейбл триггера.
        setTimeout(function() {
          syncTriggerLabel();
          close();
        }, 0);
      });

      // Escape — закрыть.
      document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape' && menu.classList.contains('m-open')) close();
      });
    }

    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', init);
    } else {
      init();
    }
  })();

  // ─── Э6.1: календарь на главной — sheet с событиями выбранного дня ───
  // Скрытие .cal-hero-right делается через CSS. Здесь добавляем интерактив:
  // тап на день с событиями → выезжает блок с событиями этого дня.
  // Привязываемся только к /home-auth.html (главная для авторизованных).
  (function initHomeCalDaySheet() {
    var path = location.pathname || '';
    if (path !== '/home-auth.html' && path !== '/') return;

    // Контейнер dom-grid пересоздаётся при каждом renderHomeCalendar(),
    // поэтому ставим делегирование на .cal-hero (родитель) — он стабилен.
    function ensureDaySheet() {
      var hero = document.querySelector('.cal-hero');
      if (!hero) return null;
      var sheet = hero.querySelector('.m-cal-daysheet');
      if (!sheet) {
        sheet = document.createElement('div');
        sheet.className = 'm-cal-daysheet';
        sheet.innerHTML =
          '<div class="m-cal-daysheet-head">' +
            '<span class="m-cal-daysheet-title">События дня</span>' +
            '<button type="button" class="m-cal-daysheet-close" aria-label="Закрыть">×</button>' +
          '</div>' +
          '<div class="m-cal-daysheet-list"></div>';
        hero.appendChild(sheet);
        sheet.querySelector('.m-cal-daysheet-close').addEventListener('click', function(e) {
          e.stopPropagation();
          sheet.classList.remove('m-open');
        });
      }
      return sheet;
    }

    function fmtDate(d) {
      try {
        return d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' });
      } catch (_) {
        return (d.getDate() + '.' + (d.getMonth() + 1));
      }
    }
    function fmtTime(ev) {
      if (ev.is_all_day) return 'весь день';
      try {
        return new Date(ev.starts_at).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
      } catch (_) {
        return '';
      }
    }
    function evHref(ev) {
      if (ev.source === 'personal' || !ev.link) {
        var sa = new Date(ev.starts_at);
        var iso = sa.getFullYear() + '-' +
          String(sa.getMonth() + 1).padStart(2, '0') + '-' +
          String(sa.getDate()).padStart(2, '0');
        return '/calendar.html?date=' + iso;
      }
      return ev.link;
    }
    function escapeHtml(s) {
      return String(s == null ? '' : s)
        .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    // Показать события выбранного дня в day-sheet.
    function showEventsForDay(day) {
      var state = window.CAL_STATE;
      if (!state || !Array.isArray(state.events)) return;
      var sheet = ensureDaySheet();
      if (!sheet) return;

      var year = state.year, month = state.month;
      var dayEvents = state.events.filter(function(ev) {
        var sa = new Date(ev.starts_at);
        return sa.getFullYear() === year && sa.getMonth() === month && sa.getDate() === day;
      }).sort(function(a, b) { return new Date(a.starts_at) - new Date(b.starts_at); });

      var titleEl = sheet.querySelector('.m-cal-daysheet-title');
      if (titleEl) titleEl.textContent = 'События — ' + fmtDate(new Date(year, month, day));

      var list = sheet.querySelector('.m-cal-daysheet-list');
      if (!list) return;

      if (!dayEvents.length) {
        list.innerHTML = '<div class="m-cal-daysheet-empty">Нет событий в этот день</div>';
      } else {
        list.innerHTML = dayEvents.map(function(ev) {
          var color = ev.color || ev._color || '#1E8A4C';
          var time = fmtTime(ev);
          var loc = ev.location ? ' · ' + escapeHtml(ev.location) : '';
          return '<a class="m-cal-evrow" href="' + escapeHtml(evHref(ev)) + '">' +
            '<span class="m-cal-evrow-dot" style="background:' + escapeHtml(color) + '"></span>' +
            '<div class="m-cal-evrow-body">' +
              '<div class="m-cal-evrow-title">' + escapeHtml(ev.title || 'Событие') + '</div>' +
              '<div class="m-cal-evrow-meta">' + escapeHtml(time) + escapeHtml(loc) + '</div>' +
            '</div>' +
          '</a>';
        }).join('');
      }
      sheet.classList.add('m-open');
    }

    // Делегирование клика по дню. Слушаем на родителе .cal-hero, поскольку
    // .cal-h-grid пересоздаётся при каждом renderHomeCalendar.
    document.addEventListener('click', function(e) {
      var dayBtn = e.target.closest('.cal-h-day:not(.other-month)');
      if (!dayBtn) return;
      // Проверяем что мы на главной (страница могла поменяться после initial load)
      if (location.pathname !== '/home-auth.html' && location.pathname !== '/') return;

      // Парсим число из onclick="pickDay(d, 'YYYY-MM-DD')"
      var onclick = dayBtn.getAttribute('onclick') || '';
      var m = onclick.match(/pickDay\((\d+)/);
      if (!m) return;
      var day = parseInt(m[1], 10);
      if (!day) return;

      // pickDay (из home-auth.html) сам отрабатывает: toggle selected, ререндер.
      // Мы добавляем своё поведение в bubble — показываем sheet.
      // Используем setTimeout 0, чтобы pickDay успел обновить CAL_STATE.selectedDay.
      setTimeout(function() {
        var state = window.CAL_STATE;
        if (!state) return;
        // Если день стал selected — показываем. Если deselected (повторный тап) — скрываем.
        if (state.selectedDay === day) {
          showEventsForDay(day);
        } else {
          var sheet = document.querySelector('.m-cal-daysheet');
          if (sheet) sheet.classList.remove('m-open');
        }
      }, 0);
    });

    // Escape — закрыть sheet.
    document.addEventListener('keydown', function(e) {
      if (e.key === 'Escape') {
        var sheet = document.querySelector('.m-cal-daysheet.m-open');
        if (sheet) sheet.classList.remove('m-open');
      }
    });
  })();

})();
