/*
 * ══════════════════════════════════════════════════════════════
 *  LASTOP GROUP — Page Transitions & Smart Home Redirect
 *  Файл: web/assets/page-transitions.js
 *
 *  Три задачи в одном файле:
 *
 *  1. ПЕРЕХВАТ ССЫЛОК НА ГЛАВНУЮ (href="/").
 *     Если пользователь залогинен, клик по "/" ведёт сразу на
 *     /home-auth.html, минуя index.html с его белым экраном
 *     и двойным редиректом. Гости идут на /home-guest.html.
 *     Всё на клиенте — без правок 25 HTML-файлов.
 *
 *  2. УСКОРЕННЫЙ РЕДИРЕКТ на home-guest.html.
 *     Если пользователь открыл /home-guest.html, но у него
 *     есть токен — сразу перекидываем на /home-auth.html,
 *     чтобы не моргать гостевой страницей.
 *
 *  3. ПЛАВНЫЙ ФЕЙД между страницами.
 *     - View Transitions API если поддерживается.
 *     - Иначе — CSS opacity-фейд через классы pt-leaving / pt-ready.
 * ══════════════════════════════════════════════════════════════
 */
(function () {
  'use strict';

  var html = document.documentElement;

  // ─────────────────────────────────────────────────────────────
  // Утилиты
  // ─────────────────────────────────────────────────────────────

  function hasAuthToken() {
    try {
      var t = localStorage.getItem('token');
      return !!(t && t.trim());
    } catch (_) {
      return false;
    }
  }

  function prefersReducedMotion() {
    return window.matchMedia &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }

  function hasNativeViewTransitions() {
    return typeof document.startViewTransition === 'function' ||
      (window.CSS && CSS.supports && CSS.supports('view-transition-name: a'));
  }

  // ─────────────────────────────────────────────────────────────
  // ЗАДАЧА 2: моментальный редирект залогиненного юзера
  // с home-guest.html на home-auth.html.
  // Выполняется сразу же, чтобы не успела отрисоваться гостевая.
  // ─────────────────────────────────────────────────────────────

  (function redirectLoggedInFromGuestHome() {
    // Сравниваем только pathname, без query/hash
    var path = location.pathname;
    var isGuestHome = (
      path === '/home-guest.html' ||
      path === '/index.html' ||
      path === '/'
    );
    if (!isGuestHome) return;
    if (!hasAuthToken()) return;

    // replace, чтобы не засорять историю браузера
    location.replace('/home-auth.html');
  })();

  // Если редирект уже отправлен — дальше инициализация бессмысленна,
  // страница сейчас меняется. Ничего не делаем.
  if (html.__ptRedirecting) return;

  // ─────────────────────────────────────────────────────────────
  // ЗАДАЧА 3a: fade-in на загрузке
  // ─────────────────────────────────────────────────────────────

  if (!prefersReducedMotion()) {
    html.classList.add('pt-ready');
  }

  // ─────────────────────────────────────────────────────────────
  // ЗАДАЧА 1 + 3b: перехват кликов по ссылкам
  // ─────────────────────────────────────────────────────────────

  function isInternalNavLink(a) {
    if (!a || a.tagName !== 'A') return false;
    if (a.target && a.target !== '' && a.target !== '_self') return false;
    if (a.hasAttribute('download')) return false;
    var raw = a.getAttribute('href');
    if (!raw) return false;
    if (raw.charAt(0) === '#') return false;
    if (/^(mailto:|tel:|javascript:)/i.test(raw)) return false;

    try {
      var url = new URL(a.href, location.href);
      if (url.origin !== location.origin) return false;
      // Якорь на той же странице — не навигация
      if (url.pathname === location.pathname &&
          url.search === location.search &&
          url.hash) {
        return false;
      }
      return true;
    } catch (_) {
      return false;
    }
  }

  // Превращает href="/" (а также href на /index.html или /home-guest.html
  // при наличии токена) в правильный целевой URL
  function resolveHomeTarget(url) {
    var path = url.pathname;
    var isHomeLink = (
      path === '/' ||
      path === '/index.html' ||
      path === '/home-guest.html'
    );
    if (!isHomeLink) return null;

    var target = hasAuthToken() ? '/home-auth.html' : '/home-guest.html';
    if (path === target) return null; // уже правильный

    // Сохраняем query и hash если вдруг они были
    return target + url.search + url.hash;
  }

  var useFade = !prefersReducedMotion() && !hasNativeViewTransitions();
  var FADE_MS = 140;

  document.addEventListener('click', function (e) {
    if (e.defaultPrevented) return;
    if (e.button !== 0) return;
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;

    var a = e.target && e.target.closest ? e.target.closest('a') : null;
    if (!isInternalNavLink(a)) return;

    var url;
    try {
      url = new URL(a.href, location.href);
    } catch (_) {
      return;
    }

    // 1. Переадресация главной (href="/") на правильную home-страницу.
    //    Это ядро фикса мигания: кликая на "/", браузер не пойдёт в
    //    index.html с его белым экраном — пойдём сразу на целевую.
    var homeTarget = resolveHomeTarget(url);
    var finalHref = homeTarget != null ? new URL(homeTarget, location.href).href : a.href;

    // Если View Transitions нативно работают — не мешаем, они красивее JS-фейда.
    // Но всё равно нужно перехватить и подменить URL на homeTarget если нужно.
    if (!useFade) {
      if (homeTarget != null) {
        e.preventDefault();
        location.href = finalHref;
      }
      return;
    }

    // 2. JS-фейд: preventDefault, opacity, потом навигация.
    e.preventDefault();
    html.classList.add('pt-leaving');

    // Страховка на случай, если навигация сорвётся
    var fallback = setTimeout(function () {
      html.classList.remove('pt-leaving');
    }, 500);

    setTimeout(function () {
      clearTimeout(fallback);
      location.href = finalHref;
    }, FADE_MS);
  }, true);

  // Возврат через back/forward из bfcache — снимаем класс .pt-leaving,
  // иначе страница может остаться прозрачной
  window.addEventListener('pageshow', function () {
    html.classList.remove('pt-leaving');
  });
})();
