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
})(window);
