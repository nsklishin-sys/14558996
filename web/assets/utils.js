/* utils.js — общие утилиты для всех страниц LASTOP.
   Один source of truth. Раньше каждый HTML файл объявлял
   свою function esc() — было 8 разных версий, 20 из них
   экранировали только &<> (без кавычек), что приводило к
   XSS в HTML-атрибутах. */

(function(global) {
  'use strict';

  // Полное HTML-экранирование. Безопасно для текстовых нод
  // И для атрибутов (одинарные/двойные кавычки тоже экранируются).
  const ESC_MAP = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;'
  };
  function lastopEsc(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ESC_MAP[c]);
  }

  global.lastopEsc = lastopEsc;

  // Debug-логирование. По умолчанию выключено в проде. Включить
  // в DevTools: localStorage.setItem('lastop_debug', '1') и
  // перезагрузить страницу. Выключить: localStorage.removeItem('lastop_debug')
  // или localStorage.setItem('lastop_debug', '0').
  // Использование: lastopDebug('[upload] start', file.name)
  const LASTOP_DEBUG = (function() {
    try {
      const v = localStorage.getItem('lastop_debug');
      return v === '1' || v === 'true';
    } catch (e) {
      return false;
    }
  })();
  function lastopDebug(...args) {
    if (LASTOP_DEBUG) {
      console.log(...args);
    }
  }
  global.LASTOP_DEBUG = LASTOP_DEBUG;
  global.lastopDebug = lastopDebug;

  // ─── letter(name) — инициал(ы) из имени для плашки аватара ─────────
  // "Никита Клишин" → "НК", "Артур" → "А", "" → "?"
  function lastopLetter(s) {
    const parts = String(s || '').trim().split(/\s+/).filter(Boolean);
    if (!parts.length) return '?';
    if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
    return parts[0][0].toUpperCase();
  }
  global.lastopLetter = lastopLetter;

  // ─── avc(seed) — цвет аватара по строке-ключу (детерминированный) ──
  // Единая палитра на всю платформу. Раньше у каждой страницы был
  // свой набор → один и тот же юзер виделся разными цветами.
  const LASTOP_AVATAR_PALETTE = [
    '#5AB080', '#3A90C0', '#9060C0', '#C07030', '#1A8A6A',
    '#B05090', '#208090', '#3B6D11', '#633806', '#185FA5'
  ];
  function lastopAvc(s) {
    const str = String(s || '');
    const ch = str.length ? str.charCodeAt(0) : 0;
    return LASTOP_AVATAR_PALETTE[ch % LASTOP_AVATAR_PALETTE.length];
  }
  global.lastopAvc = lastopAvc;
  global.LASTOP_AVATAR_PALETTE = LASTOP_AVATAR_PALETTE;

  // ─── ago(timestamp) — «5 мин назад», «2 ч назад» и т.д. ────────────
  // timestamp может быть ISO-строкой, Date или числом ms.
  function lastopAgo(ts) {
    if (!ts) return '';
    const d = new Date(ts);
    if (isNaN(d.getTime())) return '';
    const m = Math.max(1, Math.floor((Date.now() - d.getTime()) / 60000));
    if (m < 60) return m + ' мин назад';
    if (m < 1440) return Math.floor(m / 60) + ' ч назад';
    if (m < 10080) return Math.floor(m / 1440) + ' д назад';
    return d.toLocaleDateString('ru-RU', { day: '2-digit', month: 'short' });
  }
  global.lastopAgo = lastopAgo;

  // ─── fmtNum(n) — «1 234 567» (русский локаль с пробелами) ──────────
  // НЕ используем в analytics.html — там своя версия с M/к-префиксами.
  function lastopFmtNum(n) {
    return (Number(n) || 0).toLocaleString('ru-RU');
  }
  global.lastopFmtNum = lastopFmtNum;

  // ─── lastopFetch — drop-in замена fetch с cookies и CSRF ──────────
  // Автоматически:
  //  1. credentials: 'include' — браузер шлёт cookies нашего домена
  //  2. На write-методах (POST/PUT/DELETE/PATCH) добавляет
  //     X-CSRF-Token header из cookie lastop_csrf
  //  3. Сохраняет любые existing headers (включая Authorization: Bearer
  //     которые фронт пока продолжает слать — bearer на бэке fallback).
  //
  // Использование: const r = await lastopFetch('/api/...', { method, body });
  function getCsrfFromCookie() {
    const all = document.cookie.split(';');
    for (const c of all) {
      const [name, ...rest] = c.trim().split('=');
      if (name === 'lastop_csrf') {
        return rest.join('=');
      }
    }
    return '';
  }
  const WRITE_METHODS = { POST: 1, PUT: 1, DELETE: 1, PATCH: 1 };
  function lastopFetch(url, opts) {
    opts = opts || {};
    // Сливаем credentials: 'include' — для cookie-auth обязательно.
    if (opts.credentials === undefined) opts.credentials = 'include';

    // Определяем method (по умолчанию GET)
    const method = String(opts.method || 'GET').toUpperCase();

    // На write-методах подкладываем X-CSRF-Token (если cookie есть)
    if (WRITE_METHODS[method]) {
      const csrf = getCsrfFromCookie();
      if (csrf) {
        // Объект Headers vs plain object — обрабатываем оба варианта
        if (opts.headers instanceof Headers) {
          opts.headers.set('X-CSRF-Token', csrf);
        } else {
          opts.headers = Object.assign({}, opts.headers || {}, {
            'X-CSRF-Token': csrf
          });
        }
      }
    }
    return fetch(url, opts).then(function(response){
      // Глобальный перехват 401: если у юзера отозвана сессия 
      // (бан, terminate-sessions, истечение, logout с другого 
      // устройства) — чистим localStorage и редиректим на /login.
      // Исключения: /, /login.html, /register.html, /reset-password.html — 
      // чтобы не зациклиться и не сломать auth-страницы.
      if (response.status === 401) {
        var p = window.location.pathname.replace(/\.html$/, '');
        var isPublic = p === '/' || p === '/login' || p === '/register' || p === '/reset-password';
        // Был ли пользователь залогинен. Для ГОСТЯ 401 — норма (авторизованные
        // эндпоинты недоступны), редиректить не нужно. Редирект только когда сессия
        // реально слетела у залогиненного.
        var wasLoggedIn = false;
        try { wasLoggedIn = !!localStorage.getItem('user'); } catch(e){}
        if (!isPublic && wasLoggedIn) {
          try { localStorage.removeItem('user'); } catch(e){}
          try { localStorage.removeItem('token'); } catch(e){}
          if (!window.__lastopAuthRedirect) {
            window.__lastopAuthRedirect = true;
            window.location.replace('/login?expired=1');
          }
        }
      }
      // Глобальный перехват "Аккаунт заблокирован" — забаненный юзер 
      // получает 403 с этим текстом, его нужно выкинуть так же.
      if (response.status === 403) {
        response.clone().json().then(function(body){
          if (body && body.error && /заблокирован/i.test(body.error)) {
            try { localStorage.removeItem('user'); } catch(e){}
            try { localStorage.removeItem('token'); } catch(e){}
            // На самой странице логина/регистрации бан показывает модалка
            // с полными данными (причина+срок) — глобальный редирект тут
            // только перезагрузил бы страницу и стёр данные модалки.
            var curPath = (location.pathname || '').replace(/\.html$/, '');
            var onAuthPage = curPath === '/login' || curPath === '/register';
            if (!onAuthPage && !window.__lastopAuthRedirect) {
              window.__lastopAuthRedirect = true;
              window.location.replace('/login?banned=1');
            }
          }
        }).catch(function(){});
      }
      return response;
    });
  }
  global.lastopFetch = lastopFetch;



  // Phase 3 (19.05): tk() возвращает truthy/falsy флаг авторизации.
  // После миграции на HttpOnly cookies токен в JS НЕ ДОСТУПЕН — поэтому
  // tk() не возвращает реальное значение токена. Но многие места кода
  // используют if (tk()) / if (!tk()) для проверки залогиненности.
  // Чтобы они продолжали работать, tk() возвращает:
  //   - '' (falsy) если пользователь НЕ залогинен (нет user в localStorage)
  //   - 'cookie' (truthy) если залогинен (есть user в localStorage)
  // В местах вида но он их и не использует
  // (auth идёт через cookies). Серверу приходят только cookies + CSRF.
  function tk() {
    try {
      return localStorage.getItem('user') ? 'cookie' : '';
    } catch (e) {
      return '';
    }
  }
  global.tk = tk;

})(window);
