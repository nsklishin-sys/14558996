/**
 * ══════════════════════════════════════════════════════════════
 *  LASTOP GROUP — Dark Theme Controller
 *  Файл: web/assets/dark-theme-init.js
 *
 *  Подключение: первым скриптом в <head> (ДО рендера страницы)
 *  <script src="/assets/dark-theme-init.js"></script>
 *
 *  Совместим с существующим theme.js — не конфликтует.
 * ══════════════════════════════════════════════════════════════
 */
(function () {
  'use strict';

  var THEME_KEY  = 'lastop-theme';
  var SYSTEM_KEY = 'lastop-theme-source';

  function getPreference() {
    return localStorage.getItem(THEME_KEY) || 'light';
  }

  function getSource() {
    return localStorage.getItem(SYSTEM_KEY) || 'manual';
  }

  function systemPrefersDark() {
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  }

  function resolveTheme() {
    if (getSource() === 'system') {
      return systemPrefersDark() ? 'dark' : 'light';
    }
    return getPreference();
  }

  function applyTheme(theme) {
    var isDark = theme === 'dark';
    document.documentElement.setAttribute('data-theme', theme);

    function applyToBody() {
      if (isDark) {
        document.body.classList.add('dark-theme');
      } else {
        document.body.classList.remove('dark-theme');
      }
      if (isDark) {
        document.documentElement.classList.add('dark');
      } else {
        document.documentElement.classList.remove('dark');
      }
    }

    if (document.body) {
      applyToBody();
    } else {
      document.addEventListener('DOMContentLoaded', applyToBody);
    }
  }

  function setTheme(theme, source) {
    localStorage.setItem(THEME_KEY, theme);
    localStorage.setItem(SYSTEM_KEY, source || 'manual');
    applyTheme(theme);
    syncAllToggles(theme);
  }

  function toggle() {
    var current = resolveTheme();
    setTheme(current === 'dark' ? 'light' : 'dark', 'manual');
  }

  function syncAllToggles(theme) {
    var isDark = theme === 'dark';
    var checkbox = document.getElementById('themeDarkToggle');
    if (checkbox) checkbox.checked = isDark;

    document.querySelectorAll('[data-dark-toggle]').forEach(function (el) {
      el.setAttribute('aria-pressed', String(isDark));
      el.classList.toggle('active', isDark);
      var iconEl = el.querySelector('[data-theme-icon]');
      if (iconEl) {
        iconEl.setAttribute('data-theme-icon', isDark ? 'dark' : 'light');
      }
    });
  }

  function watchSystemPreference() {
    if (!window.matchMedia) return;
    var mq = window.matchMedia('(prefers-color-scheme: dark)');
    mq.addEventListener('change', function (e) {
      if (getSource() === 'system') {
        applyTheme(e.matches ? 'dark' : 'light');
        syncAllToggles(e.matches ? 'dark' : 'light');
      }
    });
  }

  function bindHandlers() {
    var checkbox = document.getElementById('themeDarkToggle');
    if (checkbox && !checkbox.dataset.darkBound) {
      checkbox.dataset.darkBound = '1';
      checkbox.addEventListener('change', function () {
        setTheme(this.checked ? 'dark' : 'light', 'manual');
      });
    }

    document.querySelectorAll('[data-dark-toggle]').forEach(function (el) {
      if (el.dataset.darkBound) return;
      el.dataset.darkBound = '1';
      el.addEventListener('click', function () {
        toggle();
      });
    });
  }

  function injectFloatingButton() {
    if (document.body.dataset.darkNoFab !== undefined) return;
    if (document.getElementById('darkFab')) return;

    var fab = document.createElement('button');
    fab.id = 'darkFab';
    fab.setAttribute('data-dark-toggle', '');
    fab.setAttribute('aria-label', 'Переключить тёмную тему');
    fab.setAttribute('title', 'Тёмная / светлая тема');
    fab.style.cssText = [
      'position:fixed',
      'bottom:24px',
      'right:24px',
      'z-index:9999',
      'width:44px',
      'height:44px',
      'border-radius:50%',
      'border:1.5px solid var(--bdr,#DDE8E2)',
      'background:var(--w,#fff)',
      'box-shadow:0 4px 16px rgba(0,0,0,.14)',
      'cursor:pointer',
      'display:flex',
      'align-items:center',
      'justify-content:center',
      'transition:transform .2s,box-shadow .2s',
      'outline:none'
    ].join(';');

    fab.innerHTML = '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" '
      + 'stroke="currentColor" stroke-width="1.8" stroke-linecap="round" '
      + 'style="color:var(--gmt,#5A8A6A)">'
      + '<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>'
      + '</svg>';

    fab.addEventListener('mouseenter', function () {
      this.style.transform = 'scale(1.1)';
      this.style.boxShadow = '0 6px 20px rgba(0,0,0,.2)';
    });
    fab.addEventListener('mouseleave', function () {
      this.style.transform = 'scale(1)';
      this.style.boxShadow = '0 4px 16px rgba(0,0,0,.14)';
    });

    document.body.appendChild(fab);
    bindHandlers();
  }

  applyTheme(resolveTheme());

  document.addEventListener('DOMContentLoaded', function () {
    syncAllToggles(resolveTheme());
    bindHandlers();
    watchSystemPreference();
    if (!document.getElementById('themeDarkToggle')) {
      injectFloatingButton();
    }
  });

  window.LastopTheme = {
    toggle: toggle,
    set: function (theme) { setTheme(theme, 'manual'); },
    current: resolveTheme,
    followSystem: function () { setTheme(systemPrefersDark() ? 'dark' : 'light', 'system'); }
  };

}());
