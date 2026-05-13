/* legal-links.js — вставляет ссылки в выпадашку профиля.
 * "О нас" — главный пункт. Плюс быстрый доступ к Пользовательскому соглашению
 * и Политике конфиденциальности. Остальные документы — на странице /about.html. */
(function () {
  'use strict';

  function build() {
    var dropdowns = document.querySelectorAll('.profile-dd');
    if (!dropdowns.length) return;

    dropdowns.forEach(function (dd) {
      if (dd.querySelector('.pdd-legal-block')) return;
      var logout = dd.querySelector('.pdd-item.logout');
      if (!logout) return;

      var html = '\
<div class="pdd-legal-block">\
  <div class="pdd-div"></div>\
  <a href="/about.html" class="pdd-item">\
    <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>\
    <span class="pdd-item-label">О нас</span>\
  </a>\
  <a href="/terms.html" target="_blank" rel="noopener" class="pdd-item">\
    <svg viewBox="0 0 24 24"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="9" y1="13" x2="15" y2="13"/><line x1="9" y1="17" x2="15" y2="17"/></svg>\
    <span class="pdd-item-label">Пользовательское соглашение</span>\
  </a>\
  <a href="/privacy.html" target="_blank" rel="noopener" class="pdd-item">\
    <svg viewBox="0 0 24 24"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>\
    <span class="pdd-item-label">Политика конфиденциальности</span>\
  </a>\
  <div class="pdd-div"></div>\
</div>';

      var wrap = document.createElement('div');
      wrap.innerHTML = html;
      logout.parentNode.insertBefore(wrap.firstElementChild, logout);
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', build);
  } else {
    build();
  }
  setTimeout(build, 300);
  setTimeout(build, 1000);
})();
