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
    var stored = localStorage.getItem(THEME_KEY);
    if (stored) return stored;
    // Если в localStorage пусто — следуем настройкам устройства
    return systemPrefersDark() ? 'dark' : 'light';
  }

  function getSource() {
    var stored = localStorage.getItem(SYSTEM_KEY);
    if (stored) return stored;
    // Если источник не задан — по умолчанию следуем системе
    return localStorage.getItem(THEME_KEY) ? 'manual' : 'system';
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

  function createThemeButton() {
    var btn = document.createElement('button');
    btn.id = 'darkFab';
    btn.setAttribute('data-dark-toggle', '');
    btn.setAttribute('aria-label', 'Переключить тёмную тему');
    btn.setAttribute('title', 'Тёмная / светлая тема');
    btn.innerHTML = '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" '
      + 'stroke="currentColor" stroke-width="1.8" stroke-linecap="round" '
      + 'style="color:var(--gmt,#5A8A6A)">'
      + '<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>'
      + '</svg>';
    return btn;
  }

  function injectThemeButton() {
    if (document.body.dataset.darkNoFab !== undefined) return;
    if (document.getElementById('darkFab')) return;

    var settingsLink = document.querySelector(
      '.nav .nav-item[href="/settings.html"], '
      + '.global-sidebar a[href="/settings.html"], '
      + '.nav .nav-item[href*="settings"], '
      + '.nav .nav-item[data-guest-locked][href*="settings"]'
    );
    // У гостя href переписан на javascript:void(0) — 
    // ищем по тексту "Настройки"
    if (!settingsLink) {
      var allItems = document.querySelectorAll('.nav .nav-item');
      for (var i = 0; i < allItems.length; i++) {
        if ((allItems[i].textContent || '').trim().indexOf('Настройки') === 0) {
          settingsLink = allItems[i];
          break;
        }
      }
    }
    var fab = createThemeButton();

    if (settingsLink && settingsLink.parentNode) {
      var row = document.createElement('div');
      row.style.cssText = 'display:flex;align-items:center;gap:8px;width:100%';

      var parent = settingsLink.parentNode;
      parent.insertBefore(row, settingsLink);
      row.appendChild(settingsLink);
      row.appendChild(fab);

      settingsLink.style.flex = '1';
      fab.style.cssText = [
        'width:36px',
        'height:36px',
        'border-radius:10px',
        'border:1.5px solid var(--bdr,#DDE8E2)',
        'background:var(--w,#fff)',
        'cursor:pointer',
        'display:flex',
        'align-items:center',
        'justify-content:center',
        'transition:transform .2s,box-shadow .2s',
        'outline:none'
      ].join(';');
    } else {
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
      document.body.appendChild(fab);
    }

    fab.addEventListener('mouseenter', function () {
      this.style.transform = 'scale(1.06)';
      this.style.boxShadow = '0 6px 20px rgba(0,0,0,.2)';
    });
    fab.addEventListener('mouseleave', function () {
      this.style.transform = 'scale(1)';
      this.style.boxShadow = '';
    });

    bindHandlers();
  }

  applyTheme(resolveTheme());

  document.addEventListener('DOMContentLoaded', function () {
    syncAllToggles(resolveTheme());
    bindHandlers();
    watchSystemPreference();
    if (!document.getElementById('themeDarkToggle')) {
      injectThemeButton();
    }
  });

  window.LastopTheme = {
    toggle: toggle,
    set: function (theme) { setTheme(theme, 'manual'); },
    current: resolveTheme,
    followSystem: function () { setTheme(systemPrefersDark() ? 'dark' : 'light', 'system'); }
  };

}());
