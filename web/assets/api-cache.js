(() => {
  if (window.lastopCache) return;

  const NS = 'lt.cache.';
  const DEFAULT_TTL = 60_000; // 60 секунд по умолчанию

  function safeStorage() {
    try {
      const t = '__lt_test__';
      sessionStorage.setItem(t, '1');
      sessionStorage.removeItem(t);
      return sessionStorage;
    } catch {
      return null;
    }
  }

  const store = safeStorage();

  /**
   * Прочитать значение из кеша. Возвращает null если:
   * - sessionStorage недоступен
   * - ключа нет
   * - запись устарела (старше maxAgeMs)
   * - JSON битый
   */
  function get(key, maxAgeMs) {
    if (!store) return null;
    try {
      const raw = store.getItem(NS + key);
      if (!raw) return null;
      const parsed = JSON.parse(raw);
      if (!parsed || typeof parsed.ts !== 'number') return null;
      const age = Date.now() - parsed.ts;
      const ttl = typeof maxAgeMs === 'number' ? maxAgeMs : DEFAULT_TTL;
      if (age > ttl) return null;
      return parsed.value;
    } catch {
      return null;
    }
  }

  /** Сохранить значение в кеш. value должно быть JSON-сериализуемо. */
  function set(key, value) {
    if (!store) return false;
    try {
      store.setItem(NS + key, JSON.stringify({ ts: Date.now(), value }));
      return true;
    } catch {
      // QuotaExceeded или приватный режим — игнорируем
      return false;
    }
  }

  /** Очистить ключ. Без аргументов — все ключи lt.cache.* */
  function clear(key) {
    if (!store) return;
    try {
      if (key) {
        store.removeItem(NS + key);
        return;
      }
      const keys = [];
      for (let i = 0; i < store.length; i++) {
        const k = store.key(i);
        if (k && k.startsWith(NS)) keys.push(k);
      }
      keys.forEach(k => store.removeItem(k));
    } catch {}
  }

  window.lastopCache = { get, set, clear };
})();
