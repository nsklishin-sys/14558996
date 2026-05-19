/* utils.js — общие утилиты для всех страниц LASTOP.
   Один source of truth. Раньше каждый HTML файл объявлял
   свою function esc() — было 8 разных версий, 20 из них
   экранировали только &<> (без кавычек), что приводило к
   XSS в HTML-атрибутах. */

(function(global) {
  'use strict';

  // Полное HTML-экранирование. Безопасно для текстовых нод
  // И для атрибутов (одинарные/двойные кавычки тоже экранируются).
  const ESC_MAP = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;'
  };
  function lastopEsc(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ESC_MAP[c]);
  }

  global.lastopEsc = lastopEsc;

  const LASTOP_DEBUG = false;
  function lastopDebug(...args) {
    if (LASTOP_DEBUG) console.debug('[LASTOP]', ...args);
  }
  global.LASTOP_DEBUG = LASTOP_DEBUG;
  global.lastopDebug = lastopDebug;

  // ─── letter(name) — инициал(ы) из имени для плашки аватара ─────────
  // "Никита Клишин" → "НК", "Артур" → "А", "" → "?"
  function lastopLetter(s) {
    const parts = String(s || '').trim().split(/\s+/).filter(Boolean);
    if (!parts.length) return '?';
    if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
    return parts[0][0].toUpperCase();
  }
  global.lastopLetter = lastopLetter;

  // ─── avc(seed) — цвет аватара по строке-ключу (детерминированный) ──
  // Единая палитра на всю платформу. Раньше у каждой страницы был
  // свой набор → один и тот же юзер виделся разными цветами.
  const LASTOP_AVATAR_PALETTE = [
    '#5AB080', '#3A90C0', '#9060C0', '#C07030', '#1A8A6A',
    '#B05090', '#208090', '#3B6D11', '#633806', '#185FA5'
  ];
  function lastopAvc(s) {
    const str = String(s || '');
    const ch = str.length ? str.charCodeAt(0) : 0;
    return LASTOP_AVATAR_PALETTE[ch % LASTOP_AVATAR_PALETTE.length];
  }
  global.lastopAvc = lastopAvc;
  global.LASTOP_AVATAR_PALETTE = LASTOP_AVATAR_PALETTE;

  // ─── ago(timestamp) — «5 мин назад», «2 ч назад» и т.д. ────────────
  // timestamp может быть ISO-строкой, Date или числом ms.
  function lastopAgo(ts) {
    if (!ts) return '';
    const d = new Date(ts);
    if (isNaN(d.getTime())) return '';
    const m = Math.max(1, Math.floor((Date.now() - d.getTime()) / 60000));
    if (m < 60) return m + ' мин назад';
    if (m < 1440) return Math.floor(m / 60) + ' ч назад';
    if (m < 10080) return Math.floor(m / 1440) + ' д назад';
    return d.toLocaleDateString('ru-RU', { day: '2-digit', month: 'short' });
  }
  global.lastopAgo = lastopAgo;

  // ─── fmtNum(n) — «1 234 567» (русский локаль с пробелами) ──────────
  // НЕ используем в analytics.html — там своя версия с M/к-префиксами.
  function lastopFmtNum(n) {
    return (Number(n) || 0).toLocaleString('ru-RU');
  }
  global.lastopFmtNum = lastopFmtNum;
})(window);
