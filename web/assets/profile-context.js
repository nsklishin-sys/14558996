/* profile-context.js — свитчер активного контекста профиля.
 *
 * Активный контекст хранится в localStorage:
 *   active_company_id   — ID компании (если установлен — корпоративный контекст)
 *   active_community_id — ID сообщества (если установлен — community контекст)
 * Установлен может быть ТОЛЬКО ОДИН из них (взаимоисключающие).
 * Если оба отсутствуют — личный профиль.
 *
 * Скрипт находит элемент #profileContextSlot внутри выпадашки
 * профиля и рисует туда секцию переключения с разделами:
 *   КОНТЕКСТ → Личный
 *   МОИ КОМПАНИИ → ... (где юзер состоит)
 *   МОИ СООБЩЕСТВА → ... (где юзер owner/moderator/admin)
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
    if (id == null) {
      localStorage.removeItem('active_company_id');
    } else {
      localStorage.setItem('active_company_id', String(id));
      localStorage.removeItem('active_community_id'); // взаимоисключение
    }
  }

  function getActiveCommunityID() {
    var v = localStorage.getItem('active_community_id');
    if (!v) return null;
    var n = parseInt(v, 10);
    return isNaN(n) ? null : n;
  }
  function setActiveCommunityID(id) {
    if (id == null) {
      localStorage.removeItem('active_community_id');
    } else {
      localStorage.setItem('active_community_id', String(id));
      localStorage.removeItem('active_company_id'); // взаимоисключение
    }
  }

  function getActiveContextKind() {
    if (getActiveCompanyID() != null) return 'company';
    if (getActiveCommunityID() != null) return 'community';
    return 'personal';
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

  // Экспортируем глобально
  window.getActiveCompanyID = getActiveCompanyID;
  window.setActiveCompanyID = setActiveCompanyID;
  window.getActiveCommunityID = getActiveCommunityID;
  window.setActiveCommunityID = setActiveCommunityID;
  window.getActiveContextKind = getActiveContextKind;

  // Старая функция — для обратной совместимости
  window.withActiveCompanyHeaders = function (h) {
    var headers = Object.assign({}, h || {});
    var id = getActiveCompanyID();
    if (id != null) headers['X-Active-Company-Id'] = String(id);
    return headers;
  };

  // Универсальная функция — добавляет либо company-, либо community-header
  window.withActiveContextHeaders = function (h) {
    var headers = Object.assign({}, h || {});
    var companyID = getActiveCompanyID();
    var communityID = getActiveCommunityID();
    if (companyID != null) {
      headers['X-Active-Company-Id'] = String(companyID);
    } else if (communityID != null) {
      headers['X-Active-Community-Id'] = String(communityID);
    }
    return headers;
  };

  function styles() {
    if (document.getElementById('pcx-styles')) return;
    var css =
      '.pcx-section{padding:6px;border-top:1px solid var(--bdr);border-bottom:1px solid var(--bdr);background:#fafdfb}' +
      '.pcx-label{font-size:9px;font-weight:800;color:var(--gmt);letter-spacing:.08em;text-transform:uppercase;padding:6px 10px 4px}' +
      '.pcx-label + .pcx-label{padding-top:8px;border-top:1px solid var(--bdr);margin-top:4px}' +
      '.pcx-row{display:flex;align-items:center;gap:8px;padding:7px 10px;border-radius:8px;cursor:pointer;color:var(--tm);font-size:12px;transition:background .12s}' +
      '.pcx-row:hover{background:var(--gp);color:var(--g)}' +
      '.pcx-row.active{background:var(--gl);color:var(--g);font-weight:700}' +
      '.pcx-row-av{width:22px;height:22px;border-radius:6px;display:grid;place-items:center;font-size:9px;font-weight:800;color:#fff;flex-shrink:0;background:var(--g);overflow:hidden}' +
      '.pcx-row-av img{width:100%;height:100%;object-fit:cover}' +
      '.pcx-row-name{flex:1;min-width:0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}' +
      '.pcx-row-check{width:14px;height:14px;flex-shrink:0;color:var(--g)}' +
      '.pcx-row-check svg{width:14px;height:14px;fill:none;stroke:currentColor;stroke-width:2.5;stroke-linecap:round;stroke-linejoin:round;display:block}' +
      '.pcx-verified{display:inline-flex;align-items:center;width:12px;height:12px;color:var(--g);flex-shrink:0;margin-left:4px}' +
      '.pcx-verified svg{width:12px;height:12px;fill:currentColor;display:block}' +
      '.pcx-create{color:var(--gmt);font-style:italic}' +
      /* Глобальная защита: ограничиваем высоту выпадашки профиля и включаем скролл */
      '.profile-dd{max-height:calc(100vh - 100px);overflow-y:auto}';
    var el = document.createElement('style');
    el.id = 'pcx-styles';
    el.textContent = css;
    document.head.appendChild(el);
  }

  function rowHTML(opts) {
    // opts: {name, color, avatar, isActive, onClickAttr, verifiedBadge}
    var avHtml = opts.avatar
      ? '<img src="' + escAttr(opts.avatar) + '" alt="">'
      : esc(initials(opts.name));
    var checkHtml = opts.isActive
      ? '<span class="pcx-row-check"><svg viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg></span>'
      : '';
    return (
      '<div class="pcx-row' + (opts.isActive ? ' active' : '') + '" ' + opts.onClickAttr + '>' +
        '<div class="pcx-row-av" style="background:' + escAttr(opts.color || '#5A8A6A') + '">' + avHtml + '</div>' +
        '<div class="pcx-row-name">' + esc(opts.name) + '</div>' +
        (opts.verifiedBadge || '') +
        checkHtml +
      '</div>'
    );
  }

  function render(companies, communities, kind, activeID) {
    var slot = document.getElementById('profileContextSlot');
    if (!slot) return;

    var sections = [];

    // КОНТЕКСТ — личный
    var personalRow = rowHTML({
      name: 'Личный профиль',
      color: '#5A8A6A',
      isActive: kind === 'personal',
      onClickAttr: 'onclick="window.__pcxSwitchPersonal()"'
    });
    sections.push('<div class="pcx-label">Контекст</div>' + personalRow);

    // МОИ КОМПАНИИ
    if (companies && companies.length) {
      var companyRows = companies.map(function (c) {
        var verified = c.is_verified
          ? '<span class="pcx-verified" title="Подтверждена"><svg viewBox="0 0 24 24"><path d="M9 12l2 2 4-4M12 22a10 10 0 1 1 0-20 10 10 0 0 1 0 20z"/></svg></span>'
          : '';
        return rowHTML({
          name: c.name,
          color: c.accent_color || '#1E8A4C',
          avatar: c.logo_url,
          isActive: kind === 'company' && c.id === activeID,
          onClickAttr: 'onclick="window.__pcxSwitchCompany(' + c.id + ')"',
          verifiedBadge: verified
        });
      }).join('');
      sections.push('<div class="pcx-label">Мои компании</div>' + companyRows);
    }

    // Создать компанию
    sections.push(
      '<a class="pcx-row pcx-create" href="/my-company.html?create=1" style="text-decoration:none">' +
        '<div class="pcx-row-av" style="background:#C0DECA;color:#1E8A4C">+</div>' +
        '<div class="pcx-row-name">Создать компанию</div>' +
      '</a>'
    );

    // МОИ СООБЩЕСТВА
    if (communities && communities.length) {
      var communityRows = communities.map(function (c) {
        return rowHTML({
          name: c.name,
          color: c.color || '#5AB080',
          avatar: c.avatar_url,
          isActive: kind === 'community' && c.id === activeID,
          onClickAttr: 'onclick="window.__pcxSwitchCommunity(' + c.id + ')"'
        });
      }).join('');
      sections.push('<div class="pcx-label">Мои сообщества</div>' + communityRows);
    }

    slot.innerHTML = '<div class="pcx-section">' + sections.join('') + '</div>';
  }

  function updateMyCompanyMenuItem(kind, activeID) {
    var anchors = document.querySelectorAll('.pdd-item[href^="/my-company.html"]');
    var cached = window.__pcxCompaniesCache || [];
    var activeCompany = (kind === 'company' && activeID != null)
      ? cached.find(function(c){ return c.id === activeID; })
      : null;
    for (var i = 0; i < anchors.length; i++) {
      var label = anchors[i].querySelector('.pdd-item-label');
      if (label) {
        label.textContent = (kind === 'company') ? 'Управление компанией' : 'Моя компания';
      }
      if (!cached.length && kind !== 'company') {
        anchors[i].style.display = 'none';
      } else {
        anchors[i].style.display = '';
      }
      if (activeCompany && activeCompany.slug) {
        anchors[i].setAttribute('href', '/my-company.html?id=' + encodeURIComponent(activeCompany.slug));
      } else {
        anchors[i].setAttribute('href', '/my-company.html');
      }
    }
  }

  function applySwitch() {
    var kind = getActiveContextKind();
    var activeID = (kind === 'company') ? getActiveCompanyID()
                : (kind === 'community') ? getActiveCommunityID()
                : null;
    var companies = window.__pcxCompaniesCache || [];
    var communities = window.__pcxCommunitiesCache || [];
    render(companies, communities, kind, activeID);
    updateMyCompanyMenuItem(kind, activeID);
    var dd = document.getElementById('profileDD');
    var tp = document.getElementById('topbarProfile');
    if (dd) dd.classList.remove('open');
    if (tp) tp.classList.remove('open');
    document.dispatchEvent(new CustomEvent('profileContextChanged', {
      detail: { kind: kind, companyID: getActiveCompanyID(), communityID: getActiveCommunityID() }
    }));
  }

  window.__pcxSwitchPersonal = function () {
    setActiveCompanyID(null);
    setActiveCommunityID(null);
    applySwitch();
  };
  window.__pcxSwitchCompany = function (companyID) {
    setActiveCompanyID(companyID);
    applySwitch();
  };
  window.__pcxSwitchCommunity = function (communityID) {
    setActiveCommunityID(communityID);
    applySwitch();
  };

  async function load() {
    if (!tk()) return;
    if (!document.getElementById('profileContextSlot')) return;
    styles();
    try {
      var headers = { Authorization: 'Bearer ' + tk() };
      var responses = await Promise.all([
        fetch(API + '/companies?tab=my&limit=50', { headers: headers }).catch(function(){ return null; }),
        fetch(API + '/communities?limit=200', { headers: headers }).catch(function(){ return null; })
      ]);
      var companiesResp = responses[0];
      var communitiesResp = responses[1];

      // Компании
      var companies = [];
      if (companiesResp && companiesResp.ok) {
        var d1 = await companiesResp.json();
        companies = (d1.items || []).map(function (c) {
          return {
            id: c.id,
            name: c.name,
            accent_color: c.accent_color,
            is_verified: c.is_verified,
            slug: c.slug,
            logo_url: c.logo_url || ''
          };
        });
      }
      window.__pcxCompaniesCache = companies;

      // Сообщества — фильтруем только где юзер owner/moderator/admin
      var communities = [];
      if (communitiesResp && communitiesResp.ok) {
        var d2 = await communitiesResp.json();
        var allCommunities = d2.communities || d2.items || [];
        communities = allCommunities
          .filter(function (c) {
            return c.my_role === 'owner' || c.my_role === 'moderator' || c.my_role === 'admin';
          })
          .map(function (c) {
            return {
              id: c.id,
              name: c.name,
              color: c.color || '',
              avatar_url: c.avatar_url || '',
              role: c.my_role
            };
          });
      }
      window.__pcxCommunitiesCache = communities;

      // Сбросить активный контекст если он невалиден
      var companyID = getActiveCompanyID();
      var communityID = getActiveCommunityID();
      if (companyID != null && !companies.some(function (c) { return c.id === companyID; })) {
        setActiveCompanyID(null);
      }
      if (communityID != null && !communities.some(function (c) { return c.id === communityID; })) {
        setActiveCommunityID(null);
      }

      var kind = getActiveContextKind();
      var activeID = (kind === 'company') ? getActiveCompanyID()
                   : (kind === 'community') ? getActiveCommunityID()
                   : null;
      render(companies, communities, kind, activeID);
      updateMyCompanyMenuItem(kind, activeID);
    } catch (err) {
      console.error('profile-context load:', err);
      render([], [], 'personal', null);
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', load);
  } else {
    load();
  }
})();
