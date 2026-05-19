// LASTOP push client (P4) — регистрация SW, подписка/отписка, мост в settings UI.
// API: window.LastopPush.{status, enable, disable}.
// Не запрашивает permission автоматически — только из юзер-инициированного enable().
(function () {
  'use strict';

  var API = '/api';
  var SW_URL = '/sw.js';
  var SW_SCOPE = '/';

  // ── Утилиты ────────────────────────────────────────────────
  function tk() {
    try { return localStorage.getItem('token'); } catch (_) { return null; }
  }

  function isSupported() {
    return (
      typeof window !== 'undefined' &&
      'serviceWorker' in navigator &&
      'PushManager' in window &&
      'Notification' in window
    );
  }

  // Конвертация base64url → Uint8Array для applicationServerKey
  function urlBase64ToUint8Array(base64String) {
    var padding = '='.repeat((4 - base64String.length % 4) % 4);
    var base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
    var raw = atob(base64);
    var out = new Uint8Array(raw.length);
    for (var i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
    return out;
  }

  // Сериализация PushSubscription в формат бэкенда
  function serializeSubscription(sub) {
    var json = sub.toJSON();
    var keys = json.keys || {};
    return {
      endpoint: json.endpoint,
      keys: { p256dh: keys.p256dh || '', auth: keys.auth || '' },
      user_agent: navigator.userAgent || '',
    };
  }

  // ── VAPID public key (кешируем в памяти на сессию) ─────────
  var cachedVapidKey = null;
  function fetchVapidPublicKey() {
    if (cachedVapidKey) return Promise.resolve(cachedVapidKey);
    return lastopFetch(API + '/push/vapid-public-key')
      .then(function (r) { if (!r.ok) throw new Error('vapid ' + r.status); return r.json(); })
      .then(function (d) {
        var key = d && (d.public_key || d.publicKey);
        if (!key) throw new Error('vapid empty');
        cachedVapidKey = key;
        return key;
      });
  }

  // ── Регистрация SW ─────────────────────────────────────────
  var swReadyPromise = null;
  function getRegistration() {
    if (swReadyPromise) return swReadyPromise;
    swReadyPromise = navigator.serviceWorker.register(SW_URL, { scope: SW_SCOPE })
      .then(function () { return navigator.serviceWorker.ready; })
      .catch(function (e) { swReadyPromise = null; throw e; });
    return swReadyPromise;
  }

  // ── Отправка / удаление подписки на бэкенде ────────────────
  function sendSubscribe(sub) {
    var token = tk();
    if (!token) return Promise.resolve({ ok: false, reason: 'no_token' });
    return lastopFetch(API + '/push/subscribe', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token },
      body: JSON.stringify(serializeSubscription(sub)),
    }).then(function (r) { return { ok: r.ok, status: r.status }; })
      .catch(function () { return { ok: false, reason: 'network' }; });
  }

  function sendUnsubscribe(endpoint) {
    var token = tk();
    if (!token) return Promise.resolve({ ok: false, reason: 'no_token' });
    return lastopFetch(API + '/push/unsubscribe', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token },
      body: JSON.stringify({ endpoint: endpoint }),
    }).then(function (r) { return { ok: r.ok, status: r.status }; })
      .catch(function () { return { ok: false, reason: 'network' }; });
  }

  // ── Публичный API ──────────────────────────────────────────
  function status() {
    if (!isSupported()) return Promise.resolve({ supported: false, permission: 'default', subscribed: false });
    var perm = (typeof Notification !== 'undefined') ? Notification.permission : 'default';
    return navigator.serviceWorker.getRegistration(SW_SCOPE).then(function (reg) {
      if (!reg) return { supported: true, permission: perm, subscribed: false };
      return reg.pushManager.getSubscription().then(function (sub) {
        return { supported: true, permission: perm, subscribed: !!sub };
      });
    }).catch(function () { return { supported: true, permission: perm, subscribed: false }; });
  }

  function enable() {
    if (!isSupported()) return Promise.resolve({ ok: false, reason: 'unsupported' });
    if (!tk()) return Promise.resolve({ ok: false, reason: 'no_token' });

    return Notification.requestPermission().then(function (perm) {
      if (perm !== 'granted') return { ok: false, reason: perm }; // denied | default
      return Promise.all([getRegistration(), fetchVapidPublicKey()]).then(function (arr) {
        var reg = arr[0];
        var vapidKey = arr[1];
        return reg.pushManager.getSubscription().then(function (existing) {
          if (existing) return existing;
          return reg.pushManager.subscribe({
            userVisibleOnly: true,
            applicationServerKey: urlBase64ToUint8Array(vapidKey),
          });
        });
      }).then(function (sub) {
        return sendSubscribe(sub).then(function (res) {
          if (res.ok) return { ok: true };
          return { ok: false, reason: 'backend', status: res.status };
        });
      });
    }).catch(function (e) {
      return { ok: false, reason: 'error', error: String(e && e.message || e) };
    });
  }

  function disable() {
    if (!isSupported()) return Promise.resolve({ ok: true });
    return navigator.serviceWorker.getRegistration(SW_SCOPE).then(function (reg) {
      if (!reg) return { ok: true };
      return reg.pushManager.getSubscription().then(function (sub) {
        if (!sub) return { ok: true };
        var endpoint = sub.endpoint;
        return sub.unsubscribe().then(function () {
          return sendUnsubscribe(endpoint).then(function () { return { ok: true }; });
        });
      });
    }).catch(function () { return { ok: true }; });
  }

  // ── Тихая авто-регистрация SW + refresh подписки ───────────
  // Если юзер залогинен и уже давал permission ранее — обновляем
  // last_seen_at через повторный subscribe (бэкенд: ON CONFLICT DO UPDATE).
  function autoBootstrap() {
    if (!isSupported() || !tk()) return;
    getRegistration().then(function (reg) {
      if (typeof Notification === 'undefined' || Notification.permission !== 'granted') return;
      return reg.pushManager.getSubscription().then(function (sub) {
        if (sub) return sendSubscribe(sub);
      });
    }).catch(function () { /* тихо */ });
  }

  // ── Экспорт ────────────────────────────────────────────────
  window.LastopPush = { status: status, enable: enable, disable: disable };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', autoBootstrap, { once: true });
  } else {
    autoBootstrap();
  }
})();
