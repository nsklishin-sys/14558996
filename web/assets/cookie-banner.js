// LASTOP cookie banner — однократное согласие на куки и обработку данных по 152-ФЗ.
// Согласие хранится в localStorage. После принятия баннер не показывается.
// Без согласия аналитика (Я.Метрика) не запускается.
(function(){
  var KEY = 'lastop-cookies-consent';
  var existing = null;
  try { existing = localStorage.getItem(KEY); } catch(e) {}

  function setConsent(value){
    try { localStorage.setItem(KEY, value); } catch(e) {}
    // Триггер для других скриптов (например Я.Метрики) что согласие получено
    try {
      window.dispatchEvent(new CustomEvent('lastop-cookie-consent', { detail: { value: value } }));
    } catch(e) {}
  }

  // API: проверить согласие
  window.lastopCookieConsent = function(){
    try { return localStorage.getItem(KEY); } catch(e) { return null; }
  };

  // Если уже принято или явно отклонено — не показываем
  if (existing === 'accepted' || existing === 'rejected') return;

  function render(){
    if (document.getElementById('lastop-cookie-banner')) return;
    var bar = document.createElement('div');
    bar.id = 'lastop-cookie-banner';
    bar.setAttribute('role', 'dialog');
    bar.setAttribute('aria-label', 'Уведомление о cookies');
    bar.innerHTML =
      '<style>' +
      '#lastop-cookie-banner{position:fixed;bottom:16px;left:16px;right:16px;max-width:560px;margin:0 auto;background:#fff;border:1px solid #DDE8E2;border-radius:14px;padding:14px 18px;box-shadow:0 12px 40px rgba(30,138,76,.18);z-index:99999;font-family:Manrope,system-ui,sans-serif;color:#1A2A22;display:flex;align-items:center;gap:14px;flex-wrap:wrap}' +
      '#lastop-cookie-banner .lcb-text{flex:1;min-width:240px;font-size:12.5px;line-height:1.55;color:#3A5245}' +
      '#lastop-cookie-banner .lcb-text a{color:#1E8A4C;font-weight:700;text-decoration:none}' +
      '#lastop-cookie-banner .lcb-text a:hover{text-decoration:underline}' +
      '#lastop-cookie-banner .lcb-actions{display:flex;gap:8px;flex-shrink:0}' +
      '#lastop-cookie-banner button{font-family:inherit;font-size:12px;font-weight:700;padding:8px 14px;border-radius:8px;cursor:pointer;border:1.5px solid transparent;transition:opacity .15s,background .15s}' +
      '#lastop-cookie-banner .lcb-accept{background:#1E8A4C;color:#fff;border-color:#1E8A4C}' +
      '#lastop-cookie-banner .lcb-accept:hover{background:#22A05A}' +
      '#lastop-cookie-banner .lcb-reject{background:#fff;color:#3A5245;border-color:#DDE8E2}' +
      '#lastop-cookie-banner .lcb-reject:hover{background:#F0FAF4;color:#1E8A4C;border-color:#C0DECA}' +
      '@media(max-width:520px){#lastop-cookie-banner{padding:12px 14px}#lastop-cookie-banner .lcb-actions{width:100%}#lastop-cookie-banner button{flex:1}}' +
      '</style>' +
      '<div class="lcb-text">' +
        'Мы используем файлы cookie для работы платформы и аналитики. ' +
        'Продолжая, вы соглашаетесь с <a href="/privacy" target="_blank">Политикой обработки персональных данных</a> ' +
        'и <a href="/legal/cookies.html" target="_blank">Политикой использования cookie</a> ' +
        'в соответствии с 152-ФЗ.' +
      '</div>' +
      '<div class="lcb-actions">' +
        '<button class="lcb-reject" type="button">Только необходимые</button>' +
        '<button class="lcb-accept" type="button">Принять все</button>' +
      '</div>';
    bar.querySelector('.lcb-accept').addEventListener('click', function(){
      setConsent('accepted');
      bar.remove();
    });
    bar.querySelector('.lcb-reject').addEventListener('click', function(){
      setConsent('rejected');
      bar.remove();
    });
    document.body.appendChild(bar);
  }

  if (document.body){
    render();
  } else {
    document.addEventListener('DOMContentLoaded', render);
  }
})();
