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
