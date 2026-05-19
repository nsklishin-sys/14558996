// LASTOP Yandex Metrica integration.
// Загружается только при наличии cookie-согласия (cookie-banner.js должен быть загружен).
// Counter ID берётся из window.LASTOP_YA_METRIKA_ID, который инжектится сервером.
(function(){
  var counterID = window.LASTOP_YA_METRIKA_ID || '';
  if (!counterID) return;

  function loadMetrica(){
    if (window.__lastopMetricaLoaded) return;
    window.__lastopMetricaLoaded = true;
    (function(m,e,t,r,i,k,a){
      m[i]=m[i]||function(){(m[i].a=m[i].a||[]).push(arguments)};
      m[i].l=1*new Date();
      for(var j=0;j<document.scripts.length;j++){
        if(document.scripts[j].src===r){return;}
      }
      k=e.createElement(t);a=e.getElementsByTagName(t)[0];
      k.async=1;k.src=r;a.parentNode.insertBefore(k,a);
    })(window,document,"script","https://mc.yandex.ru/metrika/tag.js","ym");
    window.ym(counterID, "init", {
      clickmap: true,
      trackLinks: true,
      accurateTrackBounce: true,
      webvisor: false,
      defer: true
    });
  }

  function check(){
    var consent = (window.lastopCookieConsent && window.lastopCookieConsent()) || null;
    if (consent === 'accepted'){
      loadMetrica();
    }
  }

  if (document.readyState === 'loading'){
    document.addEventListener('DOMContentLoaded', check);
  } else {
    check();
  }
  window.addEventListener('lastop-cookie-consent', function(e){
    if (e && e.detail && e.detail.value === 'accepted'){
      loadMetrica();
    }
  });
})();
