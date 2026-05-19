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

  // Debug-логирование. По умолчанию выключено в проде. Включить
  // в DevTools: localStorage.setItem('lastop_debug', '1') и
  // перезагрузить страницу. Выключить: localStorage.removeItem('lastop_debug')
  // или localStorage.setItem('lastop_debug', '0').
  // Использование: lastopDebug('[upload] start', file.name)
  const LASTOP_DEBUG = (function() {
    try {
      const v = localStorage.getItem('lastop_debug');
      return v === '1' || v === 'true';
    } catch (e) {
      return false;
    }
  })();
  function lastopDebug(...args) {
    if (LASTOP_DEBUG) {
      console.log(...args);
    }
  }
  global.LASTOP_DEBUG = LASTOP_DEBUG;
  global.lastopDebug = lastopDebug;
})(window);
