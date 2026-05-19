/* repost-picker.js — универсальная выпадашка "Куда репостнуть?"
 *
 * Использование:
 *   window.lastopRepostPicker.open({
 *     anchor: domElementToAnchorTo,
 *     postPublicID: 'pst_xxxx',
 *     onSuccess: function(repost) { ... }   // опционально, вызывается после успеха
 *   });
 *
 * Сам компонент:
 *   - сам подгружает список целей через GET /api/repost-targets (с sessionStorage кешем)
 *   - рендерит floating-меню рядом с anchor
 *   - при клике на цель показывает inline-поле комментария и кнопку "Опубликовать"
 *   - шлёт POST /api/posts/{id}/repost
 *   - закрывает себя, показывает toast, дёргает onSuccess
 */
(function () {
  'use strict';

  const API = '/api';
  const CACHE_KEY = 'lastop_repost_targets_v1';
  const CACHE_TTL = 5 * 60 * 1000; // 5 минут

  // Phase 4 (L-7): после cookie-only auth tk() — это просто
  // isLoggedIn() boolean. Возвращает 'cookie' если есть user в
  // localStorage, '' если нет.
  function tk() {
    try { return localStorage.getItem('user') ? 'cookie' : ''; }
    catch (_) { return ''; }
  }
  function authHeaders(extra) {
    var h = {};
    if (extra) {
      for (var k in extra) h[k] = extra[k];
    }
    return h;
  }
  function esc(s) {
    return String(s || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }
  function letter(s) {
    return String(s || '?').charAt(0).toUpperCase();
  }
  function toast(msg) {
    if (window.lastopToast) return window.lastopToast(msg);
    // Fallback: alert как крайний вариант
    var t = document.createElement('div');
    t.style.cssText = 'position:fixed;bottom:24px;left:50%;transform:translateX(-50%);background:#1A2A22;color:#fff;padding:11px 22px;border-radius:12px;font-size:13px;font-weight:600;z-index:9999;box-shadow:0 8px 24px rgba(0,0,0,.18)';
    t.textContent = msg;
    document.body.appendChild(t);
    setTimeout(function () { t.remove(); }, 2800);
  }

  async function loadTargets(force) {
    if (!force) {
      try {
        var cached = sessionStorage.getItem(CACHE_KEY);
        if (cached) {
          var parsed = JSON.parse(cached);
          if (parsed && parsed.at && (Date.now() - parsed.at) < CACHE_TTL && Array.isArray(parsed.targets)) {
            return parsed.targets;
          }
        }
      } catch (e) {}
    }
    try {
      var r = await lastopFetch(API + '/repost-targets', { headers: authHeaders() });
      if (!r.ok) throw new Error('http ' + r.status);
      var d = await r.json();
      var targets = Array.isArray(d.targets) ? d.targets : [];
      try {
        sessionStorage.setItem(CACHE_KEY, JSON.stringify({ at: Date.now(), targets: targets }));
      } catch (e) {}
      return targets;
    } catch (e) {
      console.error('[repost-picker] loadTargets failed:', e);
      return null;
    }
  }

  function colorFromName(s) {
    var colors = ['#5AB080','#3A90C0','#9060C0','#C07030','#1A8A6A','#B05090','#208090','#3B6D11','#633806','#185FA5'];
    var ch = String(s || '').charCodeAt(0) || 0;
    return colors[ch % colors.length];
  }

  // ── стили один раз
  function injectStyles() {
    if (document.getElementById('lastop-repost-picker-styles')) return;
    var css = '' +
      '.lrp-overlay{position:fixed;inset:0;z-index:9000;background:transparent}' +
      '.lrp-pop{position:fixed;z-index:9001;width:340px;max-height:480px;background:#fff;border:1px solid #DDE8E2;border-radius:14px;box-shadow:0 16px 40px rgba(30,138,76,.18);display:flex;flex-direction:column;overflow:hidden;font-family:Manrope,sans-serif}' +
      '.lrp-header{padding:11px 14px;border-bottom:1px solid #DDE8E2;display:flex;align-items:center;justify-content:space-between}' +
      '.lrp-title{font-size:12px;font-weight:700;color:#5A8A6A;letter-spacing:.05em;text-transform:uppercase}' +
      '.lrp-close{background:none;border:none;cursor:pointer;width:24px;height:24px;border-radius:6px;display:grid;place-items:center;color:#5A8A6A}' +
      '.lrp-close:hover{background:#F0FAF4;color:#1E8A4C}' +
      '.lrp-close svg{width:14px;height:14px;fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round}' +
      '.lrp-body{flex:1;overflow-y:auto;padding:5px 0}' +
      '.lrp-body::-webkit-scrollbar{width:4px}.lrp-body::-webkit-scrollbar-thumb{background:#C0DECA;border-radius:99px}' +
      '.lrp-item{display:flex;align-items:center;gap:10px;padding:9px 14px;cursor:pointer;transition:background .12s;border:none;background:none;width:100%;text-align:left;font-family:inherit}' +
      '.lrp-item:hover{background:#F0FAF4}' +
      '.lrp-av{width:32px;height:32px;border-radius:8px;display:grid;place-items:center;color:#fff;font-size:12px;font-weight:800;flex-shrink:0;overflow:hidden}' +
      '.lrp-av img{width:100%;height:100%;object-fit:cover}' +
      '.lrp-info{flex:1;min-width:0}' +
      '.lrp-name{font-size:13px;font-weight:700;color:#1A2A22;line-height:1.3;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}' +
      '.lrp-sub{font-size:11px;color:#5A8A6A;margin-top:1px}' +
      '.lrp-divider{height:1px;background:#DDE8E2;margin:5px 14px}' +
      '.lrp-section{padding:8px 14px 4px;font-size:10px;font-weight:700;color:#5A8A6A;letter-spacing:.08em;text-transform:uppercase}' +
      '.lrp-comment-pane{padding:12px 14px;border-top:1px solid #DDE8E2;background:#F0FAF4}' +
      '.lrp-comment-pane.hidden{display:none}' +
      '.lrp-selected{display:flex;align-items:center;gap:8px;margin-bottom:9px;font-size:12px;color:#3A5245}' +
      '.lrp-selected b{color:#1A2A22}' +
      '.lrp-selected .lrp-av{width:24px;height:24px;font-size:10px}' +
      '.lrp-textarea{width:100%;border:1.5px solid #DDE8E2;border-radius:10px;padding:9px 11px;font-family:inherit;font-size:12.5px;color:#1A2A22;outline:none;resize:none;height:62px;background:#fff;line-height:1.45}' +
      '.lrp-textarea:focus{border-color:#22A05A}' +
      '.lrp-actions{display:flex;justify-content:flex-end;gap:7px;margin-top:9px}' +
      '.lrp-btn{padding:7px 14px;border-radius:99px;border:none;font-family:inherit;font-size:12px;font-weight:700;cursor:pointer;transition:all .15s}' +
      '.lrp-btn-sec{background:#fff;border:1.5px solid #DDE8E2;color:#5A8A6A}' +
      '.lrp-btn-sec:hover{background:#F2F5F3}' +
      '.lrp-btn-pri{background:#1E8A4C;color:#fff}' +
      '.lrp-btn-pri:hover{background:#22A05A}' +
      '.lrp-btn-pri:disabled{background:#C0DECA;cursor:not-allowed}' +
      '.lrp-loading{padding:18px;text-align:center;color:#5A8A6A;font-size:12px}' +
      '.lrp-empty{padding:18px;text-align:center;color:#5A8A6A;font-size:12px}';
    var style = document.createElement('style');
    style.id = 'lastop-repost-picker-styles';
    style.textContent = css;
    document.head.appendChild(style);
  }

  function buildAvatar(target) {
    var color = colorFromName(target.name);
    if (target.avatar) {
      return '<div class="lrp-av" style="background:' + color + '"><img src="' + esc(target.avatar) + '" alt=""></div>';
    }
    return '<div class="lrp-av" style="background:' + color + '">' + esc(letter(target.name)) + '</div>';
  }

  // Глобальное состояние одного открытого picker'а
  var current = null;

  function close() {
    if (!current) return;
    if (current.overlay) current.overlay.remove();
    if (current.pop) current.pop.remove();
    document.removeEventListener('keydown', current.keyHandler, true);
    current = null;
  }

  function position(pop, anchor) {
    var rect = anchor.getBoundingClientRect();
    var vw = window.innerWidth;
    var vh = window.innerHeight;
    var w = 340;
    var left = rect.left;
    if (left + w > vw - 12) left = Math.max(12, vw - w - 12);
    var top = rect.bottom + 6;
    if (top + 400 > vh - 12) {
      top = Math.max(12, rect.top - 400 - 6);
    }
    pop.style.left = left + 'px';
    pop.style.top = top + 'px';
  }

  async function open(opts) {
    opts = opts || {};
    if (!opts.postPublicID) {
      console.error('[repost-picker] postPublicID required');
      return;
    }
    injectStyles();
    close();

    var overlay = document.createElement('div');
    overlay.className = 'lrp-overlay';
    overlay.addEventListener('click', close);

    var pop = document.createElement('div');
    pop.className = 'lrp-pop';
    pop.addEventListener('click', function (e) { e.stopPropagation(); });

    pop.innerHTML = '' +
      '<div class="lrp-header">' +
      '  <span class="lrp-title">Поделиться в…</span>' +
      '  <button class="lrp-close" type="button"><svg viewBox="0 0 24 24"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>' +
      '</div>' +
      '<div class="lrp-body"><div class="lrp-loading">Загрузка…</div></div>' +
      '<div class="lrp-comment-pane hidden"></div>';

    pop.querySelector('.lrp-close').addEventListener('click', close);

    document.body.appendChild(overlay);
    document.body.appendChild(pop);

    if (opts.anchor) {
      position(pop, opts.anchor);
      window.addEventListener('scroll', function onScroll() {
        if (!current) {
          window.removeEventListener('scroll', onScroll, true);
          return;
        }
        position(pop, opts.anchor);
      }, true);
    } else {
      pop.style.left = '50%';
      pop.style.top = '50%';
      pop.style.transform = 'translate(-50%, -50%)';
    }

    var keyHandler = function (e) {
      if (e.key === 'Escape') {
        e.preventDefault();
        close();
      }
    };
    document.addEventListener('keydown', keyHandler, true);

    current = { overlay: overlay, pop: pop, keyHandler: keyHandler, opts: opts };

    var targets = await loadTargets(false);
    if (!current) return; // picker могли закрыть пока грузили
    var body = pop.querySelector('.lrp-body');

    if (!targets) {
      body.innerHTML = '<div class="lrp-empty">Не удалось загрузить список. Попробуйте позже.</div>';
      return;
    }
    if (!targets.length) {
      body.innerHTML = '<div class="lrp-empty">Нет доступных направлений</div>';
      return;
    }

    // Группировка
    var personal = targets.filter(function (t) { return t.type === 'user'; });
    var companies = targets.filter(function (t) { return t.type === 'company'; });
    var communities = targets.filter(function (t) { return t.type === 'community'; });
    var friends = targets.filter(function (t) { return t.type === 'friend'; });

    var html = '';
    personal.forEach(function (t) {
      html += renderItem(t);
    });
    if (companies.length) {
      html += '<div class="lrp-divider"></div><div class="lrp-section">Компании</div>';
      companies.forEach(function (t) { html += renderItem(t); });
    }
    if (communities.length) {
      html += '<div class="lrp-divider"></div><div class="lrp-section">Сообщества</div>';
      communities.forEach(function (t) { html += renderItem(t); });
    }
    if (friends.length) {
      html += '<div class="lrp-divider"></div><div class="lrp-section">Отправить в чат</div>';
      friends.forEach(function (t) { html += renderItem(t); });
    }
    body.innerHTML = html;

    body.querySelectorAll('.lrp-item').forEach(function (el) {
      el.addEventListener('click', function () {
        var t = JSON.parse(el.dataset.target);
        showCommentPane(t);
      });
    });
  }

  function renderItem(t) {
    return '<button class="lrp-item" type="button" data-target=\'' + esc(JSON.stringify(t)) + '\'>' +
      buildAvatar(t) +
      '<div class="lrp-info">' +
      '<div class="lrp-name">' + esc(t.name) + '</div>' +
      '<div class="lrp-sub">' + esc(t.subtitle || '') + '</div>' +
      '</div>' +
      '</button>';
  }

  function showCommentPane(target) {
    if (!current) return;
    var pop = current.pop;
    var body = pop.querySelector('.lrp-body');
    var pane = pop.querySelector('.lrp-comment-pane');

    var isFriend = target.type === 'friend';
    var label = isFriend ? 'Отправить в чат: <b>' + esc(target.name) + '</b>' : 'Репост: <b>' + esc(target.name) + '</b>';
    var placeholder = isFriend ? 'Добавить сообщение (опционально)…' : 'Добавить комментарий (опционально)…';
    var btnText = isFriend ? 'Отправить' : 'Опубликовать';
    pane.innerHTML = '' +
      '<div class="lrp-selected">' + buildAvatar(target) + label + '</div>' +
      '<textarea class="lrp-textarea" id="lrpComment" placeholder="' + placeholder + '" maxlength="20000"></textarea>' +
      '<div class="lrp-actions">' +
      '<button class="lrp-btn lrp-btn-sec" type="button" id="lrpCancel">Назад</button>' +
      '<button class="lrp-btn lrp-btn-pri" type="button" id="lrpSubmit">' + btnText + '</button>' +
      '</div>';
    pane.classList.remove('hidden');
    body.style.display = 'none';

    pane.querySelector('#lrpCancel').addEventListener('click', function () {
      pane.classList.add('hidden');
      pane.innerHTML = '';
      body.style.display = '';
    });
    pane.querySelector('#lrpSubmit').addEventListener('click', function () {
      submit(target);
    });

    setTimeout(function () {
      var ta = pane.querySelector('#lrpComment');
      if (ta) ta.focus();
    }, 50);
  }

  async function submit(target) {
    if (!current) return;
    var pop = current.pop;
    var taEl = pop.querySelector('#lrpComment');
    var btn = pop.querySelector('#lrpSubmit');
    var comment = taEl ? taEl.value.trim() : '';

    if (btn) {
      btn.disabled = true;
      btn.textContent = target.type === 'friend' ? 'Отправляем…' : 'Публикуем…';
    }

    // Отправка в личный чат другу — не репост, а сообщение со ссылкой
    if (target.type === 'friend') {
      try {
        // 1) Открыть direct-чат
        var dc = await lastopFetch(API + '/chat/conversations/direct', {
          method: 'POST',
          headers: authHeaders({ 'Content-Type': 'application/json' }),
          body: JSON.stringify({ user_public_id: target.public_id })
        });
        if (!dc.ok) {
          toast('Не удалось открыть чат');
          if (btn) { btn.disabled = false; btn.textContent = 'Отправить'; }
          return;
        }
        var dd = await dc.json();
        var convPID = dd.conversation && dd.conversation.public_id;
        if (!convPID) {
          toast('Не удалось открыть чат');
          if (btn) { btn.disabled = false; btn.textContent = 'Отправить'; }
          return;
        }
        // 2) Сообщение со ссылкой на пост
        var link = location.origin + '/news-detail.html?id=' + encodeURIComponent(current.opts.postPublicID);
        var msg = comment ? (comment + '\\n\\n' + link) : link;
        var mr = await lastopFetch(API + '/chat/conversations/' + encodeURIComponent(convPID) + '/messages', {
          method: 'POST',
          headers: authHeaders({ 'Content-Type': 'application/json' }),
          body: JSON.stringify({ content: msg })
        });
        if (!mr.ok) {
          var md = await mr.json().catch(function () { return {}; });
          toast(md.error || 'Не удалось отправить');
          if (btn) { btn.disabled = false; btn.textContent = 'Отправить'; }
          return;
        }
        toast('Отправлено ✓');
        close();
      } catch (e) {
        toast('Ошибка сети');
        if (btn) { btn.disabled = false; btn.textContent = 'Отправить'; }
      }
      return;
    }

    // Обычный репост в ленту (свою / компании / сообщества)
    var body = {
      comment: comment,
      target_company_id: 0,
      target_community_id: 0
    };
    if (target.type === 'company') body.target_company_id = target.id;
    if (target.type === 'community') body.target_community_id = target.id;

    try {
      var r = await lastopFetch(API + '/posts/' + encodeURIComponent(current.opts.postPublicID) + '/repost', {
        method: 'POST',
        headers: authHeaders({ 'Content-Type': 'application/json' }),
        body: JSON.stringify(body)
      });
      var d = await r.json().catch(function () { return {}; });
      if (!r.ok) {
        toast(d.error || ('Не удалось опубликовать (' + r.status + ')'));
        if (btn) {
          btn.disabled = false;
          btn.textContent = 'Опубликовать';
        }
        return;
      }
      toast('Опубликовано ✓');
      var cb = current.opts.onSuccess;
      var repostObj = d.post;
      close();
      if (typeof cb === 'function') cb(repostObj);
    } catch (e) {
      toast('Ошибка сети');
      if (btn) {
        btn.disabled = false;
        btn.textContent = 'Опубликовать';
      }
    }
  }

  window.lastopRepostPicker = {
    open: open,
    close: close
  };
})();
