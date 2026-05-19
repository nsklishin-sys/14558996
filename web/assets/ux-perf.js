(() => {
  if (window.lastopUX) return;

  /**
   * Универсальный debounce. Возвращает функцию которая вызовет fn
   * через `delay` мс после последнего вызова.
   */
  function debounce(fn, delay) {
    let t = null;
    const wrapped = function (...args) {
      clearTimeout(t);
      t = setTimeout(() => fn.apply(this, args), delay);
    };
    wrapped.cancel = () => clearTimeout(t);
    return wrapped;
  }

  /**
   * Подписать input на debounced-обработчик ввода.
   * - input: <input> элемент или его ID
   * - handler: функция, вызовется через delay мс паузы
   * - delay: мс, дефолт 300
   */
  function attachSearchDebounce(input, handler, delay) {
    const el = typeof input === 'string' ? document.getElementById(input) : input;
    if (!el || el.__ltDebounced) return;
    el.__ltDebounced = true;
    const debounced = debounce(handler, delay || 300);
    el.addEventListener('input', e => debounced(e));
  }

  /**
   * Prefetch URL — делает обычный fetch (для HTTP-кеша браузера).
   * Идемпотентно: один URL не префетчится дважды за сессию.
   */
  const _prefetched = new Set();
  function prefetch(url) {
    if (!url || _prefetched.has(url)) return;
    _prefetched.add(url);
    try {
      const token = localStorage.getItem('token');
      const headers = token ? { } : {};
      lastopFetch(url, { headers, credentials: 'same-origin' }).catch(() => {});
    } catch {}
  }

  /**
   * Маппинг URL nav-страницы → endpoints для префетча.
   * Используется привязкой к стандартному <a class="nav-item" href="…">
   * из shell-injector.
   */
  const NAV_PREFETCH = {
    '/home-auth.html':    ['/api/feed?limit=30', '/api/chat/conversations?limit=4'],
    '/dashboard.html':    ['/api/feed?limit=30&type=news', '/api/posts/top?period=week&limit=5', '/api/posts/trends'],
    '/projects.html':     ['/api/projects?limit=50'],
    '/companies.html':    ['/api/companies?limit=50'],
    '/communities.html':  ['/api/communities?limit=50'],
    '/jobs.html':         ['/api/jobs?limit=30'],
    '/exhibitions.html':  ['/api/exhibitions?limit=30'],
    '/events.html':       ['/api/events?limit=30'],
    '/chat.html':         ['/api/chat/conversations?limit=30'],
    '/profile.html':      ['/api/me'],
    '/forum.html':        ['/api/forum/topics?limit=30'],
    '/notifications.html':['/api/notifications?limit=30'],
  };

  /**
   * Бутстрап префетча для всей nav-навигации. При наведении на
   * .nav-item — запускается prefetch соответствующих endpoints.
   */
  function bootstrapNavPrefetch() {
    document.querySelectorAll('a.nav-item, .nav a').forEach(a => {
      if (a.__ltPrefetched) return;
      a.__ltPrefetched = true;
      const enter = () => {
        try {
          const href = new URL(a.href, location.origin).pathname;
          const list = NAV_PREFETCH[href];
          if (!list) return;
          list.forEach(prefetch);
        } catch {}
      };
      a.addEventListener('mouseenter', enter, { passive: true });
      a.addEventListener('focus', enter, { passive: true });
      // Сенсорный — touchstart: гарантированный сигнал что юзер хочет перейти
      a.addEventListener('touchstart', enter, { passive: true });
    });
  }

  // Автоматический бутстрап после загрузки DOM
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', bootstrapNavPrefetch);
  } else {
    bootstrapNavPrefetch();
  }

  // Также после shell-injector подмены навигации — повторно проходим
  // (shell-injector делает MutationObserver, поэтому достаточно одного
  // отложенного вызова)
  setTimeout(bootstrapNavPrefetch, 800);

  window.lastopUX = { debounce, attachSearchDebounce, prefetch, bootstrapNavPrefetch };
})();
