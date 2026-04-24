/*
 * ══════════════════════════════════════════════════════════════
 *  LASTOP GROUP — Page Transitions (JS fallback)
 *  Файл: web/assets/page-transitions.js
 *
 *  - Помечает <html> классом .pt-ready для fade-in на загрузке
 *  - Перехватывает клики по внутренним ссылкам, навешивает
 *    .pt-leaving для fade-out перед навигацией
 *  - Не активен если браузер поддерживает View Transitions API
 *    (тогда нативный переход красивее)
 *  - Уважает prefers-reduced-motion
 * ══════════════════════════════════════════════════════════════
 */
(function () {
  'use strict';

  var html = document.documentElement;

  // Уважаем системные настройки
  var prm = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)');
  if (prm && prm.matches) return;

  // Нативные View Transitions уже всё делают — JS-fallback не нужен.
  // При этом fade-in на загрузке всё равно полезен для унификации вида.
  var hasNativeTransitions =
    typeof document.startViewTransition === 'function' ||
    (window.CSS && CSS.supports && CSS.supports('view-transition-name: a'));

  // Fade-in при загрузке
  html.classList.add('pt-ready');

  if (hasNativeTransitions) return;

  // Fallback: перехватываем клики по внутренним ссылкам
  function isSameOriginInternalLink(a) {
    if (!a || a.tagName !== 'A') return false;
    if (a.target && a.target !== '' && a.target !== '_self') return false;
    if (a.hasAttribute('download')) return false;
    var href = a.getAttribute('href');
    if (!href) return false;
    if (href.charAt(0) === '#') return false;
    if (/^(mailto:|tel:|javascript:)/i.test(href)) return false;
    // Только same-origin
    try {
      var url = new URL(a.href, location.href);
      if (url.origin !== location.origin) return false;
      // Якорные ссылки в пределах страницы — тоже пропускаем
      if (url.pathname === location.pathname && url.search === location.search && url.hash) return false;
      return true;
    } catch (_) {
      return false;
    }
  }

  document.addEventListener('click', function (e) {
    // Пропускаем клики с модификаторами (открыть в новой вкладке и т.п.)
    if (e.defaultPrevented) return;
    if (e.button !== 0) return;
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;

    var a = e.target && e.target.closest ? e.target.closest('a') : null;
    if (!isSameOriginInternalLink(a)) return;

    e.preventDefault();
    var targetHref = a.href;

    html.classList.add('pt-leaving');

    // Страховка от зависания, если по какой-то причине навигация сорвётся
    var fallbackTimer = setTimeout(function () {
      html.classList.remove('pt-leaving');
    }, 400);

    // Ждём короткий фейд, потом навигируем
    setTimeout(function () {
      clearTimeout(fallbackTimer);
      location.href = targetHref;
    }, 140);
  }, true);

  // Когда пользователь возвращается по bfcache (Back/Forward Cache) —
  // снимаем класс .pt-leaving, иначе страница останется прозрачной
  window.addEventListener('pageshow', function (e) {
    html.classList.remove('pt-leaving');
  });
})();
