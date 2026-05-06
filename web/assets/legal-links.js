/* legal-links.js — вставляет ссылки на юр. документы в выпадашку профиля.
 * Находит .profile-dd .pdd-item.logout на любой странице
 * и вставляет ПЕРЕД ней блок с разделителями и тремя ссылками. */
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
  <a href="/terms.html" target="_blank" rel="noopener" class="pdd-item">\
    <svg viewBox="0 0 24 24"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="9" y1="13" x2="15" y2="13"/><line x1="9" y1="17" x2="15" y2="17"/></svg>\
    <span class="pdd-item-label">Пользовательское соглашение</span>\
  </a>\
  <a href="/privacy.html" target="_blank" rel="noopener" class="pdd-item">\
    <svg viewBox="0 0 24 24"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>\
    <span class="pdd-item-label">Политика конфиденциальности</span>\
  </a>\
  <a href="mailto:partner@lastop.ru" class="pdd-item">\
    <svg viewBox="0 0 24 24"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/><polyline points="22,6 12,13 2,6"/></svg>\
    <span class="pdd-item-label">Связаться с нами</span>\
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
