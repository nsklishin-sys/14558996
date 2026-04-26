/**
 * ═══════════════════════════════════════════════════════════════
 *  LASTOP GROUP — Global Settings Loader
 *  Файл: web/assets/global-settings.js
 *
 *  Инжектируется через middleware на каждую страницу.
 *  Загружает настройки текущего юзера через GET /api/settings и
 *  применяет их к DOM:
 *  - data-compact-feed на html (для компактных карточек ленты).
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

  function tk() {
    try {
      return localStorage.getItem('token') || '';
    } catch (_) {
      return '';
    }
  }

  function applySettings(s) {
    if (!s) return;
    var html = document.documentElement;

    // Компактная лента
    if (s.compact_feed) {
      html.setAttribute('data-compact-feed', '1');
    } else {
      html.removeAttribute('data-compact-feed');
    }

    // Широкая раскладка
    if (s.layout_mode === 'wide') {
      html.setAttribute('data-layout-mode', 'wide');
    } else {
      html.removeAttribute('data-layout-mode');
    }

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

    return fetch('/api/settings', { headers: { Authorization: 'Bearer ' + tk() } })
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

})();
