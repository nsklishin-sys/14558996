(function(){
  // Не вставляем на самих юридических страницах и на login/register (там свой layout)
  var path = location.pathname.toLowerCase();
  if (/\/(terms|privacy)\.html?$/.test(path)) return;
  if (/\/(login|register|forgot-password|reset-password)\.html?$/.test(path)) return;

  // CSS — внутри document, чтобы footer выглядел одинаково на любой странице
  var style = document.createElement('style');
  style.textContent = '\
.lastop-legal-footer{margin:48px -16px -16px;padding:24px 32px 28px;border-top:1px solid var(--bdr,#DDE8E2);background:var(--gp,#F0FAF4);font-size:12px;color:var(--gmt,#5A8A6A);font-family:inherit}\
.lastop-legal-footer-inner{max-width:1400px;margin:0 auto;display:flex;flex-wrap:wrap;align-items:center;justify-content:space-between;gap:14px}\
.lastop-legal-footer-brand{display:flex;align-items:center;gap:10px;font-weight:700;color:var(--t,#1A2A22);letter-spacing:.3px}\
.lastop-legal-footer-brand-mark{display:inline-flex;align-items:center;justify-content:center;width:22px;height:22px;border-radius:6px;background:var(--g,#1E8A4C);color:#fff;font-size:11px;font-weight:800}\
.lastop-legal-footer-links{display:flex;flex-wrap:wrap;align-items:center;gap:18px}\
.lastop-legal-footer-links a{color:var(--gmt,#5A8A6A);text-decoration:none;transition:color .15s}\
.lastop-legal-footer-links a:hover{color:var(--g,#1E8A4C)}\
.lastop-legal-footer-copyright{color:var(--gmt,#5A8A6A);font-size:11.5px}\
@media(max-width:720px){.lastop-legal-footer{margin:32px -10px -10px;padding:18px 16px 22px}.lastop-legal-footer-inner{flex-direction:column;align-items:flex-start;gap:10px}}\
';
  document.head.appendChild(style);

  function build(){
    // Уже есть футер?
    if (document.querySelector('.lastop-legal-footer')) return;

    // Куда вставляем: приоритет .main, затем .shell, затем body
    var target = document.querySelector('.main') || document.querySelector('.shell') || document.body;
    if (!target) return;

    var year = new Date().getFullYear();
    var f = document.createElement('footer');
    f.className = 'lastop-legal-footer';
    f.innerHTML = '\
<div class="lastop-legal-footer-inner">\
  <div class="lastop-legal-footer-brand"><span class="lastop-legal-footer-brand-mark">L</span>LASTOP GROUP</div>\
  <div class="lastop-legal-footer-links">\
    <a href="/terms.html" target="_blank" rel="noopener">Пользовательское соглашение</a>\
    <a href="/privacy.html" target="_blank" rel="noopener">Политика конфиденциальности</a>\
    <a href="mailto:partner@lastop.ru">partner@lastop.ru</a>\
  </div>\
  <div class="lastop-legal-footer-copyright">© ' + year + ' ООО «Ластоп Групп»</div>\
</div>';

    // Вставляем как последний дочерний элемент target
    target.appendChild(f);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', build);
  } else {
    build();
  }
})();
