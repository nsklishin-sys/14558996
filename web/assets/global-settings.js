/**
 * ═══════════════════════════════════════════════════════════════
 *  LASTOP GROUP — Global Settings Loader
 *  Файл: web/assets/global-settings.js
 *
 *  Инжектируется через middleware на каждую страницу.
 *  Загружает настройки текущего юзера через GET /api/settings и
 *  применяет их к DOM:
 *  - data-layout-mode на html (для широкой раскладки).
 *  - data-msg-perm на html (можно ли пользователю писать).
 *  - Тёмная тема уже применяется через dark-theme-init.js.
 *
 *  CSS-стили этих режимов лежат в shell-layout.css.
 * ═══════════════════════════════════════════════════════════════
 */
(function () {
  'use strict';

  var KEY_CACHE = 'lastop_settings_cache_v1';

  // Phase 4 (L-7): после cookie-only auth tk() — это просто
  // isLoggedIn() boolean.
  function tk() {
    try { return localStorage.getItem('user') ? 'cookie' : ''; }
    catch (_) { return ''; }
  }

  function applySettings(s) {
    if (!s) return;
    var html = document.documentElement;

    // Кому можно писать (для скрытия кнопок «Написать» на чужих профилях)
    if (s.privacy_who_can_message) {
      html.setAttribute('data-msg-perm', s.privacy_who_can_message);
    } else {
      html.removeAttribute('data-msg-perm');
    }

    // Сохраняем в localStorage чтобы при следующей загрузке применить мгновенно
    try {
      localStorage.setItem(KEY_CACHE, JSON.stringify(s));
    } catch (_) {}
  }

  function loadAndApplyGlobalSettings() {
    if (!tk()) return Promise.resolve(null);

    return lastopFetch('/api/settings', { headers:{} })
      .then(function (r) {
        return r.ok ? r.json() : null;
      })
      .then(function (data) {
        if (data && data.settings) {
          applySettings(data.settings);
          return data.settings;
        }
        return null;
      })
      .catch(function () {
        return null;
      });
  }

  // Сначала применяем закешированные настройки (мгновенно, без ожидания API)
  try {
    var cached = localStorage.getItem(KEY_CACHE);
    if (cached) applySettings(JSON.parse(cached));
  } catch (_) {}

  // Затем грузим свежие с бэка
  loadAndApplyGlobalSettings();

  // Экспортируем функцию чтобы settings.html и другие места могли применить
  // настройки немедленно после сохранения, без перезагрузки страницы.
  window.LastopSettings = {
    apply: applySettings,
    getCached: function () {
      try {
        return JSON.parse(localStorage.getItem(KEY_CACHE) || 'null');
      } catch (_) {
        return null;
      }
    },
    loadAndApplyGlobalSettings: loadAndApplyGlobalSettings,
  };

  window.loadAndApplyGlobalSettings = loadAndApplyGlobalSettings;

  // ═══════════════════════════════════════════════════════════════
  // LastopFormat — единый хелпер форматирования дат/времени.
  // Читает locale, timezone, date_format из закешированных настроек.
  // Падает в дефолты ru-RU / Europe/Moscow / DD.MM.YYYY если кеш пуст.
  // ═══════════════════════════════════════════════════════════════

  function getFormatSettings() {
    var s = null;
    try {
      s = JSON.parse(localStorage.getItem(KEY_CACHE) || 'null');
    } catch (_) {}
    return {
      locale: (s && s.locale) || 'ru-RU',
      timezone: (s && s.timezone) || 'Europe/Moscow',
      dateFormat: (s && s.date_format) || 'DD.MM.YYYY',
    };
  }

  // Преобразуем сокращённую локаль в полную для toLocale*
  function fullLocale(short) {
    if (!short) return 'ru-RU';
    if (short === 'ru') return 'ru-RU';
    if (short === 'en') return 'en-US';
    // Если уже полная — возвращаем как есть
    if (short.indexOf('-') > 0) return short;
    return short + '-' + short.toUpperCase();
  }

  function safeDate(input) {
    if (!input) return null;
    var d = input instanceof Date ? input : new Date(input);
    if (isNaN(d.getTime())) return null;
    return d;
  }

  // Полная дата и время: "26 апр. 2026, 17:30"
  function formatDateTime(iso) {
    var d = safeDate(iso);
    if (!d) return '';
    var s = getFormatSettings();
    var loc = fullLocale(s.locale);
    try {
      return d.toLocaleString(loc, {
        timeZone: s.timezone,
        day: 'numeric',
        month: 'short',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch (_) {
      return d.toLocaleString();
    }
  }

  // Только дата: "26 апр. 2026" или "26.04.2026" в зависимости от формата
  function formatDate(iso) {
    var d = safeDate(iso);
    if (!d) return '';
    var s = getFormatSettings();
    var loc = fullLocale(s.locale);
    try {
      return d.toLocaleDateString(loc, {
        timeZone: s.timezone,
        day: 'numeric',
        month: 'short',
        year: 'numeric',
      });
    } catch (_) {
      return d.toLocaleDateString();
    }
  }

  // Короткая дата без года: "26 апр" — для постов, комментариев свежих
  function formatDateShort(iso) {
    var d = safeDate(iso);
    if (!d) return '';
    var s = getFormatSettings();
    var loc = fullLocale(s.locale);
    try {
      return d.toLocaleDateString(loc, {
        timeZone: s.timezone,
        day: 'numeric',
        month: 'short',
      }).replace(/\.$/, '');
    } catch (_) {
      return d.toLocaleDateString();
    }
  }

  // Только время: "17:30"
  function formatTime(iso) {
    var d = safeDate(iso);
    if (!d) return '';
    var s = getFormatSettings();
    var loc = fullLocale(s.locale);
    try {
      return d.toLocaleTimeString(loc, {
        timeZone: s.timezone,
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch (_) {
      return d.toLocaleTimeString();
    }
  }

  window.LastopFormat = {
    date: formatDate,
    dateShort: formatDateShort,
    time: formatTime,
    dateTime: formatDateTime,
  };

  // ═══════════════════════════════════════════════════════════════
  // Рекламная плашка — добавляется в правый сайдбар на всех страницах
  // у которых он есть. Если уже была — не дублируем.
  // В будущем заменится системой ротации рекламных кампаний.
  // ═══════════════════════════════════════════════════════════════

  function injectAdBlock() {
    const sidebar = document.querySelector('aside.right');
    if (!sidebar) return; // нет правого сайдбара
    let ad = sidebar.querySelector('.ad-block');
    if (!ad) {
      ad = document.createElement('div');
      ad.className = 'ad-block';
      sidebar.appendChild(ad);
    }
    function esc(s){return String(s==null?'':s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');}
    fetch('/api/ads', { credentials: 'include' })
      .then(function(r){ return r.ok ? r.json() : null; })
      .then(function(d){
        var ads = (d && d.ads) || [];
        if (!ads.length) { ad.style.display = 'none'; return; }
        ad.style.display = '';
        ad.style.transition = 'opacity .25s';
        var idx = 0;
        function render(a){
          var img = a.image_url ? '<img src="'+esc(a.image_url)+'" style="max-width:100%;border-radius:8px;margin-bottom:8px" alt="">' : '';
          ad.innerHTML =
            '<div class="ad-label">реклама</div>' +
            '<div class="ad-content">' + img +
              '<div class="ad-title">'+esc(a.title)+'</div>' +
              (a.body ? '<div class="ad-sub">'+esc(a.body)+'</div>' : '') +
              '<button class="ad-btn" type="button">Узнать подробнее</button>' +
            '</div>';
          var btn = ad.querySelector('.ad-btn');
          if (btn) btn.addEventListener('click', function(){
            try { fetch('/api/ads/click?id='+a.id, {method:'POST',credentials:'include'}); } catch(e){}
            if (a.link_url) window.location.href = a.link_url;
          });
          try { fetch('/api/ads/view?id='+a.id, {method:'POST',credentials:'include'}); } catch(e){}
        }
        render(ads[0]);
        if (ads.length > 1) {
          setInterval(function(){
            idx = (idx + 1) % ads.length;
            ad.style.opacity = '0';
            setTimeout(function(){ render(ads[idx]); ad.style.opacity = '1'; }, 250);
          }, 18000);
        }
      })
      .catch(function(){ ad.style.display = 'none'; });
  }

  // Запускаем после загрузки DOM
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', injectAdBlock);
  } else {
    injectAdBlock();
  }


  // ═══════════════════════════════════════════════════════════════
  // Колокольчик уведомлений — инжектится в .topbar на всех страницах
  // ═══════════════════════════════════════════════════════════════

  let _notifPollTimer = null;
  let _notifBellEl = null;
  let _notifDDEl = null;
  let _notifBadgeEl = null;
  let _notifIsOpen = false;
  let _notifLastUnread = 0;

  function injectNotifBell() {
    // Phase 3 cookie-only: признак логина — user в localStorage.
    const isAuthed = (function(){ try { return !!JSON.parse(localStorage.getItem('user') || '{}').id; } catch { return false; } })();
    if (!isAuthed) return;

    const topbar = document.querySelector('.topbar');
    if (!topbar) return; // нет шапки на странице (например login.html)
    if (topbar.querySelector('.notif-bell')) return; // уже есть, не дублируем

    // Вставляем колокольчик внутрь правой ячейки топбара рядом с профилем
    const profileWrap = topbar.querySelector('.profile-dd-wrap');

    const wrap = document.createElement('div');
    wrap.className = 'notif-bell-wrap';
    wrap.innerHTML =
      '<button class="notif-bell" id="notifBell" type="button" aria-label="Уведомления">' +
        '<svg viewBox="0 0 24 24"><path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>' +
        '<span class="notif-bell-badge" id="notifBellBadge">0</span>' +
      '</button>' +
      '<div class="notif-dd" id="notifDD" role="menu">' +
        '<div class="notif-dd-head">' +
          '<span class="notif-dd-title">Уведомления</span>' +
          '<button class="notif-mark-all" id="notifMarkAll" type="button">Прочитать всё</button>' +
        '</div>' +
        '<div class="notif-list" id="notifList">' +
          '<div class="notif-empty"><span class="nei">📭</span>Загрузка…</div>' +
        '</div>' +
        '<div class="notif-dd-foot"><a href="/notifications.html">Все уведомления →</a></div>' +
      '</div>';

    const topbarProfile = profileWrap ? profileWrap.querySelector('.topbar-profile') : null;

    if (profileWrap && topbarProfile) {
      // Кладём колокольчик ВНУТРЬ .profile-dd-wrap, перед .topbar-profile
      profileWrap.insertBefore(wrap, topbarProfile);
    } else if (profileWrap) {
      // Если карточки профиля нет, но обёртка есть — в начало обёртки
      profileWrap.insertBefore(wrap, profileWrap.firstChild);
    } else {
      // Совсем другая разметка — fallback в конец topbar
      topbar.appendChild(wrap);
    }

    _notifBellEl = wrap.querySelector('.notif-bell');
    _notifDDEl = wrap.querySelector('.notif-dd');
    _notifBadgeEl = wrap.querySelector('.notif-bell-badge');

    _notifBellEl.addEventListener('click', function(e) {
      e.stopPropagation();
      toggleNotifDropdown();
    });

    wrap.querySelector('#notifMarkAll').addEventListener('click', function(e) {
      e.stopPropagation();
      markAllNotifications();
    });

    // Закрываем dropdown при клике вне
    document.addEventListener('click', function(e) {
      if (_notifIsOpen && !wrap.contains(e.target)) {
        closeNotifDropdown();
      }
    });

    // Закрываем по Escape
    document.addEventListener('keydown', function(e) {
      if (e.key === 'Escape' && _notifIsOpen) closeNotifDropdown();
    });

    // Стартуем polling и первый запрос
    refreshUnreadCount();
    if (_notifPollTimer) clearInterval(_notifPollTimer);
    // WS пушит notif:new мгновенно — polling оставляем редким fallback'ом.
    _notifPollTimer = setInterval(refreshUnreadCount, 60000);
    // Обновлять сразу когда вкладка снова стала активной
    document.addEventListener('visibilitychange', function() {
      if (document.visibilityState === 'visible') refreshUnreadCount();
    });
    // И на focus окна (на случай если visibility не сработал)
    window.addEventListener('focus', refreshUnreadCount);
    // Real-time push через WebSocket
    if (window.wsClient) {
      window.wsClient.on('notif:new', function() {
        refreshUnreadCount();
        // Если выпадашка уведомлений открыта — обновим список тоже
        if (_notifIsOpen) loadNotifList();
      });
    }
  }

  function toggleNotifDropdown() {
    if (_notifIsOpen) {
      closeNotifDropdown();
    } else {
      openNotifDropdown();
    }
  }

  function openNotifDropdown() {
    if (!_notifDDEl || !_notifBellEl) return;
    _notifIsOpen = true;
    _notifDDEl.classList.add('open');
    _notifBellEl.classList.add('open');
    loadNotifList();
  }

  function closeNotifDropdown() {
    if (!_notifDDEl || !_notifBellEl) return;
    _notifIsOpen = false;
    _notifDDEl.classList.remove('open');
    _notifBellEl.classList.remove('open');
  }

  async function refreshUnreadCount() {
    const isAuthed = (function(){ try { return !!JSON.parse(localStorage.getItem('user') || '{}').id; } catch { return false; } })();
    if (!isAuthed || !_notifBadgeEl) return;
    try {
      const r = await lastopFetch('/api/notifications/unread_count', {
        headers:{}
      });
      if (!r.ok) return;
      const d = await r.json();
      const n = Number(d.count || 0);
      _notifLastUnread = n;
      if (n > 0) {
        _notifBadgeEl.textContent = n > 99 ? '99+' : String(n);
        _notifBadgeEl.classList.add('has-unread');
      } else {
        _notifBadgeEl.classList.remove('has-unread');
      }
    } catch { /* без шума */ }
  }

  async function loadNotifList() {
    const list = document.getElementById('notifList');
    if (!list) return;
    const isAuthed = (function(){ try { return !!JSON.parse(localStorage.getItem('user') || '{}').id; } catch { return false; } })();
    if (!isAuthed) {
      list.innerHTML = '<div class="notif-empty">Войдите чтобы видеть уведомления</div>';
      return;
    }
    try {
      const r = await lastopFetch('/api/notifications?limit=10', {
        headers:{}
      });
      if (!r.ok) throw 0;
      const d = await r.json();
      renderNotifList(d.notifications || []);
    } catch {
      list.innerHTML = '<div class="notif-empty">Не удалось загрузить</div>';
    }
  }

  function renderNotifList(items) {
    const list = document.getElementById('notifList');
    if (!list) return;
    if (!items.length) {
      list.innerHTML = '<div class="notif-empty"><span class="nei">📭</span>Уведомлений пока нет</div>';
      return;
    }
    list.innerHTML = items.map(notifItemHTML).join('');

    // Делегированный клик: пометить как прочитанное и перейти по ссылке
    list.querySelectorAll('.notif-item').forEach(function(el) {
      el.addEventListener('click', async function(e) {
        const id = el.getAttribute('data-id');
        const href = el.getAttribute('data-href');
        const wasUnread = el.classList.contains('unread');
        if (wasUnread && id) {
          // Помечаем прочитанным «оптимистично»
          el.classList.remove('unread');
          if (_notifLastUnread > 0) {
            _notifLastUnread--;
            if (_notifLastUnread > 0) {
              _notifBadgeEl.textContent = _notifLastUnread > 99 ? '99+' : String(_notifLastUnread);
            } else {
              _notifBadgeEl.classList.remove('has-unread');
            }
          }
          // На фоне отправляем POST
          try {
            await lastopFetch('/api/notifications/' + id + '/read', {
              method: 'POST'
            });
          } catch {}
        }
        if (href && href !== '#') {
          // Не блокируем — браузер сам перейдёт по data-href
          window.location.href = href;
        }
      });
    });
  }

  function notifItemHTML(n) {
    const isUnread = !n.is_read;
    const initial = (n.actor_name || 'С').charAt(0).toUpperCase();
    const isSystem = !n.actor_name || !n.actor_public_id;
    const avClass = isSystem ? 'notif-av notif-av-system' : 'notif-av';
    const avStyle = isSystem ? '' : 'background:' + escAttr(n.actor_color || '#1E8A4C') + ';overflow:hidden';
    const avContent = isSystem
      ? svgSystemBell()
      : (n.actor_avatar
          ? '<img src="' + escAttr(n.actor_avatar) + '" alt="" style="width:100%;height:100%;object-fit:cover">'
          : escHtml(initial));
    const href = buildNotifHref(n);
    const time = formatNotifTime(n.created_at);

    return '<a class="notif-item ' + (isUnread ? 'unread' : '') + '" data-id="' + escAttr(String(n.id)) + '" data-href="' + escAttr(href) + '">' +
      '<div class="' + avClass + '" style="' + avStyle + '">' + avContent + '</div>' +
      '<div class="notif-body">' +
        '<div class="notif-title">' + escHtml(n.title || '') + '</div>' +
        (n.preview ? '<div class="notif-preview">' + escHtml(n.preview) + '</div>' : '') +
        '<div class="notif-time">' + escHtml(time) + '</div>' +
      '</div>' +
    '</a>';
  }

  function svgSystemBell() {
    return '<svg viewBox="0 0 24 24" style="width:16px;height:16px;fill:none;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round"><path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>';
  }

  function buildNotifHref(n) {
    // Маппинг source_type → URL
    switch (n.source_type) {
      case 'post':
        return n.source_public_id
          ? '/news-detail.html?id=' + encodeURIComponent(n.source_public_id)
          : '/dashboard.html';
      case 'project':
      case 'need':
        return n.source_public_id
          ? '/project-detail.html?id=' + encodeURIComponent(n.source_public_id)
          : '/projects.html';
      case 'comment':
        return n.source_public_id
          ? '/news-detail.html?id=' + encodeURIComponent(n.source_public_id)
          : '/dashboard.html';
      case 'forum_topic':
        return n.source_public_id
          ? '/forum.html?topic=' + encodeURIComponent(n.source_public_id)
          : '/forum.html';
      case 'chat':
        return '/chat.html';
      default:
        return '#';
    }
  }

  function formatNotifTime(iso) {
    if (!iso) return '';
    try {
      const d = new Date(iso);
      const now = new Date();
      const diff = (now - d) / 1000; // секунды
      if (diff < 60) return 'сейчас';
      if (diff < 3600) return Math.floor(diff / 60) + ' мин';
      if (diff < 86400) return Math.floor(diff / 3600) + ' ч';
      if (diff < 86400 * 7) return Math.floor(diff / 86400) + ' д';
      return d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' });
    } catch { return ''; }
  }

  async function markAllNotifications() {
    const isAuthed = (function(){ try { return !!JSON.parse(localStorage.getItem('user') || '{}').id; } catch { return false; } })();
    if (!isAuthed) return;
    const btn = document.getElementById('notifMarkAll');
    if (btn) { btn.disabled = true; btn.textContent = '...'; }
    try {
      await lastopFetch('/api/notifications/read_all', {
        method: 'POST'
      });
      _notifLastUnread = 0;
      if (_notifBadgeEl) _notifBadgeEl.classList.remove('has-unread');
      // Перерисовываем список (все станут «прочитанными»)
      const list = document.getElementById('notifList');
      if (list) {
        list.querySelectorAll('.notif-item.unread').forEach(function(el) {
          el.classList.remove('unread');
        });
      }
    } catch {}
    finally {
      if (btn) { btn.disabled = false; btn.textContent = 'Прочитать всё'; }
    }
  }

  // Helpers — экранирование (если global-settings уже имеет esc — переиспользуем)
  function escHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, function(c){
      return ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'})[c];
    });
  }
  function escAttr(s) { return escHtml(s); }

  // Запускаем после DOM
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', injectNotifBell);
  } else {
    injectNotifBell();
  }

})();
