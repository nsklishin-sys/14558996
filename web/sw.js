// LASTOP service worker — минимальный, для PWA install prompt и offline-фолбэка.
// Не кэширует API-запросы и /uploads/.
// Версия меняется при каждом обновлении — старый кеш сбрасывается.
const CACHE_VERSION = 'lastop-v1';
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

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);
  // Не кэшируем API, uploads, ws — всегда сеть
  if (url.pathname.startsWith('/api/') || url.pathname.startsWith('/uploads/')) {
    return;
  }
  // Только GET
  if (event.request.method !== 'GET') return;
  // Network-first для HTML, cache-first для статики
  const isHTML = event.request.headers.get('accept')?.includes('text/html');
  if (isHTML) {
    event.respondWith(
      fetch(event.request).catch(() => caches.match(event.request))
    );
    return;
  }
  // cache-first для favicon, manifest, картинок ассетов
  if (url.pathname.startsWith('/assets/') || url.pathname === '/favicon.svg' || url.pathname === '/manifest.json') {
    event.respondWith(
      caches.match(event.request).then((cached) => cached || fetch(event.request))
    );
  }
});
