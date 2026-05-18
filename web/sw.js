// LASTOP service worker — минимальный, для PWA install prompt и offline-фолбэка.
// Не кэширует API-запросы и /uploads/.
// Версия меняется при каждом обновлении — старый кеш сбрасывается.
const CACHE_VERSION = 'lastop-v2';
const STATIC_CACHE = ['/favicon.svg', '/manifest.json', '/assets/lastop-group-logo.svg'];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_VERSION).then((cache) => cache.addAll(STATIC_CACHE)).catch(() => {})
  );
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) => Promise.all(
      keys.filter((k) => k !== CACHE_VERSION).map((k) => caches.delete(k))
    ))
  );
  self.clients.claim();
});

// Network-first для критичных ассетов (скрипты и стили).
// Если сеть недоступна — fallback на кэш. После успешного fetch обновляем кэш.
function networkFirst(event) {
  event.respondWith(
    fetch(event.request).then((response) => {
      // Кэшируем только успешные ответы
      if (response && response.status === 200) {
        const clone = response.clone();
        caches.open(CACHE_VERSION).then((cache) => cache.put(event.request, clone)).catch(() => {});
      }
      return response;
    }).catch(() => caches.match(event.request))
  );
}

// Cache-first для редко меняющихся бинарных ассетов (картинки, шрифты).
function cacheFirst(event) {
  event.respondWith(
    caches.match(event.request).then((cached) => cached || fetch(event.request))
  );
}

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);

  // Не кэшируем API, uploads, ws — всегда сеть, без вмешательства
  if (url.pathname.startsWith('/api/') || url.pathname.startsWith('/uploads/')) {
    return;
  }
  // Только GET
  if (event.request.method !== 'GET') return;

  // HTML — network-first
  const isHTML = event.request.headers.get('accept')?.includes('text/html');
  if (isHTML) {
    networkFirst(event);
    return;
  }

  // Скрипты и стили в /assets/ — network-first (важно для обновления shell-injector.js, shell-layout.css)
  if (url.pathname.startsWith('/assets/') &&
      (url.pathname.endsWith('.js') || url.pathname.endsWith('.css'))) {
    networkFirst(event);
    return;
  }

  // Остальные ассеты (картинки, SVG, шрифты) — cache-first
  if (url.pathname.startsWith('/assets/') || url.pathname === '/favicon.svg' || url.pathname === '/manifest.json') {
    cacheFirst(event);
  }
});

// ─────────────────────────────────────────────────────────────
// Push-уведомления (P3)
// Обработчики push и notificationclick. Изолированы от кеша
// — изменения здесь не затрагивают существующую логику fetch.
// ─────────────────────────────────────────────────────────────

// Безопасный парсинг payload. Бэкенд (webpush-go в P6) шлёт JSON;
// при любой ошибке парсинга показываем дефолтную нотификацию,
// чтобы пользователь хотя бы узнал что что-то пришло.
function parsePushPayload(event) {
  if (!event.data) return {};
  try {
    return event.data.json();
  } catch (_) {
    try { return { body: event.data.text() }; } catch (__) { return {}; }
  }
}

self.addEventListener('push', (event) => {
  const data = parsePushPayload(event);
  const title = data.title || 'LASTOP GROUP';
  const options = {
    body: data.body || '',
    icon: data.icon || '/assets/icon-192.png',
    badge: data.badge || '/assets/icon-192.png',
    tag: data.tag || undefined,            // одинаковый tag заменяет предыдущую нотификацию
    renotify: !!data.renotify,
    requireInteraction: !!data.requireInteraction,
    data: data.data || {},                 // { url, type, ... } — читаем в notificationclick
  };
  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const targetUrl = (event.notification.data && event.notification.data.url) || '/';
  // Превращаем относительный URL в абсолютный относительно scope SW
  const absoluteUrl = new URL(targetUrl, self.registration.scope).href;

  event.waitUntil((async () => {
    const allClients = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
    // 1. Точное совпадение URL — фокусируем существующую вкладку
    for (const client of allClients) {
      if (client.url === absoluteUrl && 'focus' in client) {
        return client.focus();
      }
    }
    // 2. Любая вкладка того же origin — фокусируем и навигируем
    for (const client of allClients) {
      try {
        const clientOrigin = new URL(client.url).origin;
        if (clientOrigin === self.location.origin && 'focus' in client) {
          await client.focus();
          if ('navigate' in client) {
            try { await client.navigate(absoluteUrl); } catch (_) {}
          }
          return;
        }
      } catch (_) {}
    }
    // 3. Открываем новое окно
    if (self.clients.openWindow) {
      return self.clients.openWindow(absoluteUrl);
    }
  })());
});
