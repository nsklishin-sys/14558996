// ── WebSocket клиент LASTOP ──────────────────────────────────────
// Глобальный синглтон. Подключается автоматически после auth, держит
// одно соединение, авто-переподключается с backoff, шлёт ping каждые 25с.
//
// API:
//   wsClient.on('notif:new', cb)      — подписка на серверное событие
//   wsClient.off('notif:new', cb)     — отписка
//   wsClient.send({type, ...})        — послать сообщение серверу
//   wsClient.isConnected()            — состояние

(function () {
  if (window.wsClient) return;

  const RECONNECT_BASE_MS = 1000;
  const RECONNECT_MAX_MS = 30000;
  const PING_INTERVAL_MS = 25000;

  let ws = null;
  let reconnectAttempts = 0;
  let reconnectTimer = null;
  let pingTimer = null;
  let connected = false;
  const handlers = {}; // type -> Set<callback>

  function url() {
    const token = (function () {
      try { return null; } catch { return null; }
    })();
    if (!token) return null;
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    return proto + '//' + location.host + '/api/ws?token=' + encodeURIComponent(token);
  }

  function emit(type, data) {
    const set = handlers[type];
    if (!set) return;
    set.forEach(function (cb) {
      try { cb(data); } catch (e) { console.warn('[ws] handler error', type, e); }
    });
  }

  function connect() {
    const u = url();
    if (!u) return;
    try {
      ws = new WebSocket(u);
    } catch (e) {
      scheduleReconnect();
      return;
    }
    ws.onopen = function () {
      // Re-subscribe ко всем комнатам после reconnect
      try {
        if (window._wsRooms && window._wsRooms.size) {
          window._wsRooms.forEach(function(room){
            try { ws.send(JSON.stringify({ type: 'subscribe', room: room })); } catch {}
          });
        }
      } catch {}
      connected = true;
      reconnectAttempts = 0;
      emit('_open', null);
      // ping каждые 25с
      if (pingTimer) clearInterval(pingTimer);
      pingTimer = setInterval(function () {
        if (ws && ws.readyState === WebSocket.OPEN) {
          try { ws.send(JSON.stringify({ type: 'ping' })); } catch {}
        }
      }, PING_INTERVAL_MS);
    };
    ws.onmessage = function (e) {
      let msg;
      try { msg = JSON.parse(e.data); } catch { return; }
      if (!msg || !msg.type) return;
      if (msg.type === 'pong') return;
      emit(msg.type, msg.data);
    };
    ws.onerror = function () { /* close сработает следом */ };
    ws.onclose = function () {
      connected = false;
      if (pingTimer) { clearInterval(pingTimer); pingTimer = null; }
      emit('_close', null);
      scheduleReconnect();
    };
  }

  function scheduleReconnect() {
    if (reconnectTimer) return;
    const delay = Math.min(RECONNECT_BASE_MS * Math.pow(2, reconnectAttempts), RECONNECT_MAX_MS);
    reconnectAttempts++;
    reconnectTimer = setTimeout(function () {
      reconnectTimer = null;
      connect();
    }, delay);
  }

  window.wsClient = {
    subscribe: function(room){
      try { ws && ws.readyState === 1 && ws.send(JSON.stringify({ type: 'subscribe', room: room })); } catch {}
      // Запоминаем для повторной подписки при reconnect
      if (!window._wsRooms) window._wsRooms = new Set();
      window._wsRooms.add(room);
    },
    unsubscribe: function(room){
      try { ws && ws.readyState === 1 && ws.send(JSON.stringify({ type: 'unsubscribe', room: room })); } catch {}
      if (window._wsRooms) window._wsRooms.delete(room);
    },
    on: function (type, cb) {
      if (!handlers[type]) handlers[type] = new Set();
      handlers[type].add(cb);
    },
    off: function (type, cb) {
      if (handlers[type]) handlers[type].delete(cb);
    },
    send: function (obj) {
      if (ws && ws.readyState === WebSocket.OPEN) {
        try { ws.send(JSON.stringify(obj)); } catch {}
      }
    },
    isConnected: function () { return connected; },
    _connect: connect, // для ручного reconnect
  };

  // Автостарт при наличии токена
  function tryStart() {
    try {
      if (null) connect();
    } catch {}
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', tryStart);
  } else {
    tryStart();
  }

  // При смене вкладки — попробовать переподключиться если упал
  document.addEventListener('visibilitychange', function () {
    if (document.visibilityState === 'visible' && !connected) {
      reconnectAttempts = 0;
      if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null; }
      connect();
    }
  });
})();
