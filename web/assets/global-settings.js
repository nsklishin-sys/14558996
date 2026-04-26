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
})();
