(function(){
  'use strict';

  // ───── CSS ─────
  const css = `
.lp-share-wrap{position:relative;display:inline-block;z-index:50}
.lp-share-btn{padding:9px 14px;border-radius:12px;border:1.5px solid #DDE8E2;background:#F0FAF4;color:#5A8A6A;font-family:inherit;font-size:13px;font-weight:700;cursor:pointer;display:inline-flex;align-items:center;gap:6px;transition:all .15s}
.lp-share-btn:hover{border-color:#C0DECA;color:#1E8A4C;background:#E8F5EE}
.lp-share-btn svg{width:14px;height:14px;stroke:currentColor;fill:none;stroke-width:1.8;stroke-linecap:round}
.lp-share-dd{position:fixed;top:0;left:0;width:280px;max-height:60vh;background:#fff;border:1px solid #DDE8E2;border-radius:18px;box-shadow:0 12px 36px rgba(30,138,76,.15);opacity:0;pointer-events:none;transform:translateY(-6px);transition:opacity .18s,transform .18s;z-index:9000;overflow-y:auto}
.lp-share-dd.open{opacity:1;pointer-events:all;transform:translateY(0)}
.lp-share-head{padding:10px 14px 8px;font-size:10px;font-weight:700;color:#5A8A6A;text-transform:uppercase;letter-spacing:.08em;border-bottom:1px solid #DDE8E2}
.lp-share-item{display:flex;align-items:center;gap:10px;padding:10px 14px;cursor:pointer;transition:background .12s}
.lp-share-item:hover{background:#F0FAF4}
.lp-share-av{width:28px;height:28px;border-radius:7px;display:grid;place-items:center;font-size:10px;font-weight:800;color:#fff;flex-shrink:0}
.lp-share-body{flex:1;min-width:0}
.lp-share-name{font-size:12px;font-weight:700;color:#1A2A22;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.lp-share-sub{font-size:10px;color:#5A8A6A}
.lp-share-div{height:1px;background:#DDE8E2}
[data-theme="dark"] .lp-share-btn{background:#1c1c1e;border-color:rgba(84,84,88,.45);color:#aeb1ad}
[data-theme="dark"] .lp-share-dd{background:#2c2c2e;border-color:rgba(84,84,88,.45)}
[data-theme="dark"] .lp-share-head,[data-theme="dark"] .lp-share-sub{color:#aeb1ad}
[data-theme="dark"] .lp-share-name{color:#fff}
[data-theme="dark"] .lp-share-item:hover{background:rgba(255,255,255,.06)}
[data-theme="dark"] .lp-share-div{background:rgba(84,84,88,.45)}
`;
  const style = document.createElement('style');
  style.textContent = css;
  document.head.appendChild(style);

  // ───── helpers ─────
  const esc = s => String(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
  const initials = name => (name||'?').split(/\s+/).map(s=>s[0]||'').join('').slice(0,2).toUpperCase() || '?';

  function toast(msg, ok){
    let t = document.getElementById('lp-share-toast');
    if (!t){
      t = document.createElement('div');
      t.id = 'lp-share-toast';
      t.style.cssText = 'position:fixed;bottom:20px;left:50%;transform:translateX(-50%);background:#1A2A22;color:#fff;padding:10px 18px;border-radius:12px;font-size:13px;font-weight:600;z-index:9999;opacity:0;transition:opacity .25s;pointer-events:none';
      document.body.appendChild(t);
    }
    t.textContent = msg;
    t.style.background = ok ? '#1E8A4C' : '#1A2A22';
    t.style.opacity = '1';
    setTimeout(()=>{ t.style.opacity = '0'; }, 2400);
  }

  // ───── состояние friends-cache ─────
  let _friendsLoaded = false;
  let _friendsHTML = '';

  async function loadFriends(){
    if (_friendsLoaded) return _friendsHTML;
    try {
      const r = await lastopFetch('/api/friends', { headers:{} });
      if (!r.ok) throw 0;
      const d = await r.json();
      const friends = d.friends || [];
      if (!friends.length){
        _friendsHTML = '<div style="padding:14px 8px;text-align:center;font-size:11px;color:#5A8A6A">У вас пока нет друзей</div>';
      } else {
        _friendsHTML = friends.slice(0,20).map(f => {
          const pid = f.public_id || f.friend_id || '';
          const name = f.full_name || f.friend_name || f.name || 'Без имени';
          const sub = f.position || f.company_name || f.email || '';
          return `<div class="lp-share-item" data-friend-pid="${esc(pid)}" data-friend-name="${esc(name)}">
            <div class="lp-share-av" style="background:#3A90C0">${esc(initials(name))}</div>
            <div class="lp-share-body">
              <div class="lp-share-name">${esc(name)}</div>
              ${sub ? `<div class="lp-share-sub">${esc(sub)}</div>` : ''}
            </div>
          </div>`;
        }).join('');
      }
      _friendsLoaded = true;
    } catch {
      _friendsHTML = '<div style="padding:14px 8px;text-align:center;font-size:11px;color:#5A8A6A">Не удалось загрузить друзей</div>';
    }
    return _friendsHTML;
  }

  // ───── действия ─────
  async function repostToFeed(wrap){
    const url = wrap.dataset.shareUrl || location.href;
    const titleAttr = wrap.dataset.shareTitle || document.title || 'Публикация';
    const labelAttr = wrap.dataset.shareLabel || 'публикацией';
    try {
      const r = await lastopFetch('/api/posts', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json'},
        body: JSON.stringify({
          title: 'Делюсь ' + labelAttr + ': ' + titleAttr,
          content: 'Посмотрите: ' + url,
          type: 'news',
          tags: []
        })
      });
      if (!r.ok){ const er = await r.json().catch(()=>({})); toast(er.error || 'Не удалось опубликовать'); return; }
      toast('Опубликовано в вашей ленте ✓', true);
    } catch { toast('Ошибка сети'); }
  }

  async function repostToFriend(wrap, friendPID, friendName){
    if (!friendPID){ toast('Не удалось определить получателя'); return; }
    const url = wrap.dataset.shareUrl || location.href;
    const titleAttr = wrap.dataset.shareTitle || document.title || 'Публикация';
    const emoji = wrap.dataset.shareEmoji || '🔗';
    const message = `${emoji} ${titleAttr}\n${url}`;
    try {
      const dc = await lastopFetch('/api/chat/conversations/direct', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json'},
        body: JSON.stringify({ user_public_id: friendPID })
      });
      if (!dc.ok){ toast('Не удалось открыть чат'); return; }
      const dd = await dc.json();
      const convPID = dd.conversation && dd.conversation.public_id;
      if (!convPID){ toast('Не удалось открыть чат'); return; }
      const r = await lastopFetch(`/api/chat/conversations/${encodeURIComponent(convPID)}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json'},
        body: JSON.stringify({ content: message })
      });
      if (!r.ok){ const er = await r.json().catch(()=>({})); toast(er.error || 'Не удалось отправить'); return; }
      toast('Отправлено ' + friendName + ' ✓', true);
    } catch { toast('Ошибка сети'); }
  }

  // ───── рендер dropdown ─────
  async function buildDD(wrap){
    const friendsHTML = await loadFriends();
    return `
      <div class="lp-share-head">Поделиться в…</div>
      <div class="lp-share-item" data-action="page">
        <div class="lp-share-av" style="background:#1E8A4C">Я</div>
        <div class="lp-share-body">
          <div class="lp-share-name">Моя страница</div>
          <div class="lp-share-sub">Опубликовать в ленте</div>
        </div>
      </div>
      <div class="lp-share-div"></div>
      <div class="lp-share-head" style="padding-top:8px">Отправить в чат</div>
      ${friendsHTML}
    `;
  }

  // ───── авто-инициализация ─────
  function init(){
    document.querySelectorAll('[data-share]:not([data-share-init])').forEach(btn => {
      btn.dataset.shareInit = '1';
      // оборачиваем в .lp-share-wrap, если ещё не обернуто
      let wrap = btn.closest('.lp-share-wrap');
      if (!wrap){
        wrap = document.createElement('div');
        wrap.className = 'lp-share-wrap';
        // переносим data-* атрибуты с кнопки на wrap
        for (const a of Array.from(btn.attributes)){
          if (a.name.startsWith('data-share')) wrap.dataset[a.name.replace(/^data-/,'').replace(/-([a-z])/g,(_,c)=>c.toUpperCase())] = a.value;
        }
        btn.parentNode.insertBefore(wrap, btn);
        wrap.appendChild(btn);
      }
      btn.addEventListener('click', async (e) => {
        e.stopPropagation();
        let dd = wrap._dd;
        if (!dd){
          dd = document.createElement('div');
          dd.className = 'lp-share-dd';
          dd.innerHTML = '<div style="padding:14px 8px;text-align:center;font-size:11px;color:#5A8A6A">Загрузка…</div>';
          document.body.appendChild(dd);
          wrap._dd = dd;
          dd._wrap = wrap;
          dd.innerHTML = await buildDD(wrap);
          dd.addEventListener('click', async (ev) => {
            const item = ev.target.closest('.lp-share-item');
            if (!item) return;
            dd.classList.remove('open');
            if (item.dataset.action === 'page'){
              await repostToFeed(wrap);
            } else if (item.dataset.friendPid){
              await repostToFriend(wrap, item.dataset.friendPid, item.dataset.friendName || '');
            }
          });
        }
        if (dd.classList.contains('open')){
          dd.classList.remove('open');
          return;
        }
        // позиционируем относительно кнопки
        const r = btn.getBoundingClientRect();
        const ddW = 280;
        let left = r.left;
        if (left + ddW > window.innerWidth - 12) left = window.innerWidth - ddW - 12;
        if (left < 12) left = 12;
        dd.style.left = left + 'px';
        dd.style.top = (r.bottom + 8) + 'px';
        dd.classList.add('open');
      });
    });
  }

  // глобальный закрыватель
  document.addEventListener('click', (e) => {
    document.querySelectorAll('.lp-share-dd.open').forEach(dd => {
      const wrap = dd._wrap;
      if (dd.contains(e.target)) return;
      if (wrap && wrap.contains(e.target)) return;
      dd.classList.remove('open');
    });
  });
  window.addEventListener('scroll', () => {
    document.querySelectorAll('.lp-share-dd.open').forEach(dd => dd.classList.remove('open'));
  }, true);
  window.addEventListener('resize', () => {
    document.querySelectorAll('.lp-share-dd.open').forEach(dd => dd.classList.remove('open'));
  });

  if (document.readyState === 'loading'){
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
  // на случай динамического рендера (карточки рендерятся после fetch)
  window.LASTOPShareInit = init;
})();
