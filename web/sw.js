// LASTOP service worker — disabled.
// Старые клиенты получат этот файл, само-разрегистрируются и очистят кеши.
self.addEventListener('install', (event) => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil((async () => {
    // Чистим все кеши
    const keys = await caches.keys();
    await Promise.all(keys.map(k => caches.delete(k)));
    // Снимаем регистрацию у всех клиентов
    await self.registration.unregister();
    // Перезагружаем все открытые вкладки
    const clients = await self.clients.matchAll({ type: 'window' });
    clients.forEach(c => c.navigate(c.url));
  })());
});

// Fetch не перехватываем — все запросы идут напрямую в сеть
self.addEventListener('fetch', () => {});
