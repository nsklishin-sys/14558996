// LASTOP captcha loader: смотрит /api/captcha/config, и если CAPTCHA_TYPE=yandex —
// заменяет math-капчу на виджет Yandex SmartCaptcha. При success вызывает
// window.LASTOP_CAPTCHA.onToken(token).
// При CAPTCHA_TYPE=noop ничего не делает, math-капча работает как раньше.
(function () {
  window.LASTOP_CAPTCHA = window.LASTOP_CAPTCHA || { type: 'noop', token: '' };

  async function init() {
    try {
      const r = await lastopFetch('/api/captcha/config');
      if (!r.ok) return;
      const d = await r.json();
      if (d.type !== 'yandex' || !d.site_key) return;
      window.LASTOP_CAPTCHA.type = 'yandex';
      window.LASTOP_CAPTCHA.siteKey = d.site_key;
      renderYandexWidget();
    } catch {}
  }

  function renderYandexWidget() {
    const wrap = document.querySelector('.captcha-wrap');
    if (!wrap) return;
    wrap.innerHTML = '<div class="captcha-label">Проверка безопасности</div>'
      + '<div id="ya-captcha-container" style="min-height:100px"></div>';

    if (!document.getElementById('ya-captcha-script')) {
      const s = document.createElement('script');
      s.id = 'ya-captcha-script';
      s.src = 'https://smartcaptcha.yandexcloud.net/captcha.js?render=onload&onload=__lastopYaCaptchaOnLoad';
      s.defer = true;
      document.head.appendChild(s);
    }

    window.__lastopYaCaptchaOnLoad = function () {
      if (!window.smartCaptcha) return;
      window.smartCaptcha.render('ya-captcha-container', {
        sitekey: window.LASTOP_CAPTCHA.siteKey,
        hl: 'ru',
        callback: function (token) {
          window.LASTOP_CAPTCHA.token = token;
        }
      });
    };
  }

  // Унифицированная проверка для login/register
  window.LASTOP_CAPTCHA.check = function () {
    if (window.LASTOP_CAPTCHA.type === 'yandex') {
      return !!window.LASTOP_CAPTCHA.token;
    }
    // math fallback — используем существующую функцию checkCaptcha() если она есть
    if (typeof window.checkCaptcha === 'function') return window.checkCaptcha();
    return true;
  };
  window.LASTOP_CAPTCHA.getToken = function () {
    return window.LASTOP_CAPTCHA.token || '';
  };
  window.LASTOP_CAPTCHA.reset = function () {
    window.LASTOP_CAPTCHA.token = '';
    if (window.LASTOP_CAPTCHA.type === 'yandex' && window.smartCaptcha) {
      try {
        const widget = document.getElementById('ya-captcha-container');
        if (widget) widget.innerHTML = '';
        if (typeof window.__lastopYaCaptchaOnLoad === 'function') window.__lastopYaCaptchaOnLoad();
      } catch {}
    } else if (typeof window.newCaptcha === 'function') {
      window.newCaptcha();
    }
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
