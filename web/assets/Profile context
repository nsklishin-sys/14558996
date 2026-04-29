/* profile-context.js — свитчер активного контекста профиля.
 *
 * Активный контекст хранится в localStorage.active_company_id:
 *   null/отсутствует — личный профиль
 *   <id числом>      — корпоративный профиль этой компании
 *
 * Скрипт находит элемент #profileContextSlot внутри выпадашки
 * профиля и рисует туда секцию переключения. Также обновляет
 * пункт меню "Моя компания" на "Управление компанией" если
 * активен корпоративный контекст.
 */
(function () {
  'use strict';

  var API = '/api';
  function tk() { return localStorage.getItem('token'); }
  function getActiveCompanyID() {
    var v = localStorage.getItem('active_company_id');
    if (!v) return null;
    var n = parseInt(v, 10);
    return isNaN(n) ? null : n;
  }
  function setActiveCompanyID(id) {
    if (id == null) localStorage.removeItem('active_company_id');
    else localStorage.setItem('active_company_id', String(id));
  }
  function esc(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, function (m) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[m];
    });
  }
  function escAttr(s) { return esc(s); }
  function initials(name) {
    if (!name) return '?';
    return name.split(/\s+/).filter(Boolean).map(function (s) { return s[0]; }).slice(0, 2).join('').toUpperCase();
  }

  // Экспортируем глобально — формы создания будут читать
  window.getActiveCompanyID = getActiveCompanyID;
  window.setActiveCompanyID = setActiveCompanyID;

  // Подмешать заголовок к запросу (для бэка B-7)
  window.withActiveCompanyHeaders = function (h) {
    var headers = Object.assign({}, h || {});
    var id = getActiveCompanyID();
    if (id != null) headers['X-Active-Company-Id'] = String(id);
    return headers;
  };

  function styles() {
    if (document.getElementById('pcx-styles')) return;
    var css =
      '.pcx-section{padding:6px;border-top:1px solid var(--bdr);border-bottom:1px solid var(--bdr);background:#fafdfb}' +
      '.pcx-label{font-size:9px;font-weight:800;color:var(--gmt);letter-spacing:.08em;text-transform:uppercase;padding:6px 10px 4px}' +
      '.pcx-row{display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:8px;cursor:pointer;color:var(--tm);font-size:12px;transition:background .12s}' +
      '.pcx-row:hover{background:var(--gp);color:var(--g)}' +
      '.pcx-row.active{background:var(--gl);color:var(--g);font-weight:700}' +
      '.pcx-row-av{width:22px;height:22px;border-radius:6px;display:grid;place-items:center;font-size:9px;font-weight:800;color:#fff;flex-shrink:0;background:var(--g)}' +
      '.pcx-row-name{flex:1;min-width:0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}' +
      '.pcx-row-check{width:14px;height:14px;flex-shrink:0;color:var(--g)}' +
      '.pcx-row-check svg{width:14px;height:14px;fill:none;stroke:currentColor;stroke-width:2.5;stroke-linecap:round;stroke-linejoin:round;display:block}' +
      '.pcx-verified{display:inline-flex;align-items:center;width:12px;height:12px;color:var(--g);flex-shrink:0;margin-left:4px}' +
      '.pcx-verified svg{width:12px;height:12px;fill:currentColor;display:block}' +
      '.pcx-create{color:var(--gmt);font-style:italic}';
    var el = document.createElement('style');
    el.id = 'pcx-styles';
    el.textContent = css;
    document.head.appendChild(el);
  }

  function render(companies, activeID) {
    var slot = document.getElementById('profileContextSlot');
    if (!slot) return;

    var rows = [];
    rows.push(
      '<div class="pcx-row' + (activeID == null ? ' active' : '') + '" onclick="window.__pcxSwitch(null)">' +
        '<div class="pcx-row-av" style="background:#5A8A6A">Я</div>' +
        '<div class="pcx-row-name">Личный профиль</div>' +
        (activeID == null ? '<span class="pcx-row-check"><svg viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg></span>' : '') +
      '</div>'
    );

    for (var i = 0; i < companies.length; i++) {
      var c = companies[i];
      var isActive = c.id === activeID;
      var ini = initials(c.name);
      var color = c.accent_color || '#1E8A4C';
      var verifiedBadge = c.is_verified
        ? '<span class="pcx-verified" title="Подтверждена"><svg viewBox="0 0 24 24"><path d="M9 12l2 2 4-4M12 22a10 10 0 1 1 0-20 10 10 0 0 1 0 20z"/></svg></span>'
        : '';
      rows.push(
        '<div class="pcx-row' + (isActive ? ' active' : '') + '" onclick="window.__pcxSwitch(' + c.id + ')">' +
          '<div class="pcx-row-av" style="background:' + escAttr(color) + '">' + esc(ini) + '</div>' +
          '<div class="pcx-row-name">' + esc(c.name) + '</div>' +
          verifiedBadge +
          (isActive ? '<span class="pcx-row-check"><svg viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg></span>' : '') +
        '</div>'
      );
    }

    rows.push(
      '<a class="pcx-row pcx-create" href="/my-company.html" style="text-decoration:none">' +
        '<div class="pcx-row-av" style="background:#C0DECA;color:#1E8A4C">+</div>' +
        '<div class="pcx-row-name">Создать компанию</div>' +
      '</a>'
    );

    slot.innerHTML =
      '<div class="pcx-section">' +
        '<div class="pcx-label">Профиль</div>' +
        rows.join('') +
      '</div>';
  }

  function updateMyCompanyMenuItem(activeID) {
    // Меняем текст пункта "Моя компания" → "Управление компанией"
    // если активен корпоративный контекст.
    var anchors = document.querySelectorAll('.pdd-item[href="/my-company.html"]');
    for (var i = 0; i < anchors.length; i++) {
      var label = anchors[i].querySelector('.pdd-item-label');
      if (label) {
        label.textContent = activeID == null ? 'Моя компания' : 'Управление компанией';
      }
    }
  }

  window.__pcxSwitch = function (companyID) {
    setActiveCompanyID(companyID);
    updateMyCompanyMenuItem(companyID);
    // Перерисуем выпадашку без перезагрузки страницы
    var cached = window.__pcxCompaniesCache || [];
    render(cached, companyID);
    // Закрыть выпадашку
    var dd = document.getElementById('profileDD');
    var tp = document.getElementById('topbarProfile');
    if (dd) dd.classList.remove('open');
    if (tp) tp.classList.remove('open');
    // Уведомить страницу о смене контекста (формы могут перерисовать
    // блок «Опубликовать от лица»).
    document.dispatchEvent(new CustomEvent('profileContextChanged', { detail: { companyID: companyID } }));
  };

  async function load() {
    if (!tk()) return;
    if (!document.getElementById('profileContextSlot')) return;
    styles();
    try {
      var r = await fetch(API + '/companies?tab=my&limit=50', { headers: { Authorization: 'Bearer ' + tk() } });
      if (!r.ok) {
        render([], getActiveCompanyID());
        return;
      }
      var d = await r.json();
      var items = (d.items || []).map(function (c) {
        return { id: c.id, name: c.name, accent_color: c.accent_color, is_verified: c.is_verified, slug: c.slug };
      });
      window.__pcxCompaniesCache = items;
      var activeID = getActiveCompanyID();
      // Если активная компания не в списке (удалена / юзер исключён) — сбросить на личный
      if (activeID != null && !items.some(function (c) { return c.id === activeID; })) {
        setActiveCompanyID(null);
        activeID = null;
      }
      render(items, activeID);
      updateMyCompanyMenuItem(activeID);
    } catch (err) {
      console.error('profile-context load:', err);
      render([], getActiveCompanyID());
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', load);
  } else {
    load();
  }
})();
