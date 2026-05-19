/* company-badge.js — общий хелпер для отрисовки плашки верификации компании.
 * Используется на companies.html, company-detail.html, my-company.html и в profile-context.js.
 * window.lastopCompanyBadge(company, size?) → HTML строка плашки или ''.
 */
(function () {
  'use strict';

  // size: 'inline' (рядом с именем, мелкая иконка) | 'card' (полноценный pill в карточке) | 'full' (большая плашка статуса)
  function badge(c, size) {
    if (!c) return '';
    size = size || 'inline';
    var st = c.verification_status || (c.is_verified ? 'verified' : 'none');
    // Конфиг: { color, bg, bdr, icon (svg path), label, hint }
    var cfg = {
      verified: {
        color: '#1E8A4C', bg: '#E8F5EE', bdr: '#C0DECA',
        icon: '<path d="M9 12l2 2 4-4M12 22a10 10 0 1 1 0-20 10 10 0 0 1 0 20z"/>',
        label: 'Верифицирована',
        hint: 'Компания проверена модератором платформы.'
      },
      inn_verified: {
        color: '#185FA5', bg: '#E6F1FB', bdr: '#BAD3EF',
        icon: '<path d="M9 12l2 2 4-4"/><circle cx="12" cy="12" r="10"/>',
        label: 'ИНН подтверждён',
        hint: 'ИНН компании найден в реестре ЕГРЮЛ. Финальная верификация ожидает модератора.'
      },
      pending: {
        color: '#7A4A00', bg: '#FFF3DF', bdr: '#F0CB80',
        icon: '<circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>',
        label: 'На модерации',
        hint: 'Компания создана, проверка данных модератором в процессе.'
      },
      rejected: {
        color: '#A03030', bg: '#FBEAEA', bdr: '#EFC2C2',
        icon: '<circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/>',
        label: 'Отклонена',
        hint: 'Модератор отклонил запрос на верификацию. Свяжитесь с поддержкой.'
      },
      none: null
    };
    var conf = cfg[st];
    if (!conf) return '';

    if (size === 'inline') {
      // Маленькая круглая иконка возле имени
      return '<span class="co-vbadge co-vbadge-inline" data-vstatus="' + st + '" ' +
             'style="display:inline-flex;align-items:center;width:16px;height:16px;color:' + conf.color + '" ' +
             'title="' + conf.label + ' — ' + conf.hint + '">' +
             '<svg viewBox="0 0 24 24" style="width:16px;height:16px;fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round">' + conf.icon + '</svg>' +
             '</span>';
    }
    if (size === 'card') {
      // Капсульная плашка с подписью
      return '<span class="co-vbadge co-vbadge-card" data-vstatus="' + st + '" ' +
             'style="display:inline-flex;align-items:center;gap:5px;padding:3px 9px 3px 7px;border-radius:99px;' +
             'background:' + conf.bg + ';border:1px solid ' + conf.bdr + ';color:' + conf.color + ';' +
             'font-size:10.5px;font-weight:700;letter-spacing:.01em;line-height:1.2" ' +
             'title="' + conf.hint + '">' +
             '<svg viewBox="0 0 24 24" style="width:11px;height:11px;fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round">' + conf.icon + '</svg>' +
             conf.label +
             '</span>';
    }
    if (size === 'full') {
      // Большая плашка с текстом и подписью (для my-company.html)
      return '<div class="co-vbadge co-vbadge-full" data-vstatus="' + st + '" ' +
             'style="display:flex;align-items:center;gap:10px;padding:10px 14px;border-radius:12px;' +
             'background:' + conf.bg + ';border:1px solid ' + conf.bdr + ';color:' + conf.color + '">' +
             '<svg viewBox="0 0 24 24" style="width:18px;height:18px;flex-shrink:0;fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round">' + conf.icon + '</svg>' +
             '<div style="flex:1;min-width:0">' +
             '<div style="font-size:13px;font-weight:700;line-height:1.3">' + conf.label + '</div>' +
             '<div style="font-size:11.5px;line-height:1.45;opacity:.85;margin-top:1px">' + conf.hint + '</div>' +
             '</div>' +
             '</div>';
    }
    return '';
  }

  window.lastopCompanyBadge = badge;
})();
