// LASTOP — простые промисы для confirm и prompt вместо браузерных диалогов.
// Использование:
//   if (await lastopConfirm({title:'Удалить?', message:'Действие необратимо'})) { ... }
//   const text = await lastopPrompt({title:'Название задачи'});  // null если отменено
(function(){
  const STYLE = `
.lc-bg{position:fixed;inset:0;background:rgba(15,25,18,.45);backdrop-filter:blur(4px);
  z-index:9999;display:flex;align-items:center;justify-content:center;padding:20px;
  opacity:0;pointer-events:none;transition:opacity .18s}
.lc-bg.open{opacity:1;pointer-events:auto}
.lc-modal{background:#fff;border-radius:18px;border:1px solid #DDE8E2;
  box-shadow:0 24px 60px rgba(30,138,76,.15);padding:24px;width:100%;max-width:420px;
  font-family:'Manrope',sans-serif;transform:translateY(10px);transition:transform .18s}
.lc-bg.open .lc-modal{transform:translateY(0)}
.lc-title{font-size:16px;font-weight:800;color:#1A2A22;margin-bottom:8px}
.lc-msg{font-size:13px;color:#3A5245;line-height:1.55;margin-bottom:18px}
.lc-input{width:100%;border:1.5px solid #DDE8E2;border-radius:12px;padding:10px 12px;
  font-family:inherit;font-size:13px;color:#1A2A22;outline:none;background:#F0FAF4;
  transition:border-color .15s;margin-bottom:18px}
.lc-input:focus{border-color:#22A05A}
.lc-actions{display:flex;gap:8px;justify-content:flex-end}
.lc-btn{padding:8px 18px;border-radius:12px;font-family:inherit;font-size:13px;
  font-weight:600;cursor:pointer;border:none;transition:opacity .15s}
.lc-btn-secondary{background:#fff;border:1.5px solid #DDE8E2;color:#3A5245}
.lc-btn-secondary:hover{background:#F0FAF4}
.lc-btn-primary{background:#1E8A4C;color:#fff}
.lc-btn-primary:hover{opacity:.88}
.lc-btn-danger{background:#E04040;color:#fff}
.lc-btn-danger:hover{opacity:.88}
/* ── Universal toast ── */
.lt-toast{position:fixed;bottom:24px;left:50%;transform:translateX(-50%) translateY(20px);
  background:#1A2A22;color:#fff;padding:11px 20px;border-radius:12px;font-family:'Manrope',sans-serif;
  font-size:13px;font-weight:600;z-index:9998;opacity:0;transition:all .25s;pointer-events:none;
  max-width:calc(100vw - 32px);box-shadow:0 8px 24px rgba(15,25,18,.2);display:flex;align-items:center;gap:8px}
.lt-toast.show{opacity:1;transform:translateX(-50%) translateY(0)}
.lt-toast.success{background:#1E8A4C}
.lt-toast.error{background:#E04040}
.lt-toast.warn{background:#B07A00}
.lt-toast svg{width:14px;height:14px;fill:none;stroke:currentColor;stroke-width:2.2;stroke-linecap:round;flex-shrink:0}
/* ── Universal alert dialog ── */
.lc-modal.lc-alert .lc-title{display:flex;align-items:center;gap:10px}
.lc-alert-icon{width:24px;height:24px;border-radius:50%;display:grid;place-items:center;flex-shrink:0}
.lc-alert-icon svg{width:14px;height:14px;fill:none;stroke:#fff;stroke-width:2.4;stroke-linecap:round;stroke-linejoin:round}
.lc-alert.type-error .lc-alert-icon{background:#E04040}
.lc-alert.type-warn .lc-alert-icon{background:#B07A00}
.lc-alert.type-info .lc-alert-icon{background:#1E8A4C}
.lc-alert.type-success .lc-alert-icon{background:#1E8A4C}
/* ── Form field invalid state ── */
.lt-invalid,input.lt-invalid,select.lt-invalid,textarea.lt-invalid{
  border-color:#E04040 !important;
  box-shadow:0 0 0 3px rgba(224,64,64,.12) !important;
  animation:lt-shake .35s cubic-bezier(.36,.07,.19,.97);
}
@keyframes lt-shake{
  10%,90%{transform:translateX(-1px)}
  20%,80%{transform:translateX(2px)}
  30%,50%,70%{transform:translateX(-3px)}
  40%,60%{transform:translateX(3px)}
}
.lt-field-error{display:block;font-size:11px;font-weight:600;color:#E04040;margin-top:5px;line-height:1.4}
.lt-field-error::before{content:"⚠ ";font-size:11px}
`;
  function injectStyle(){
    if(document.getElementById('lc-style'))return;
    const s=document.createElement('style');s.id='lc-style';s.textContent=STYLE;
    document.head.appendChild(s);
  }

  function buildDialog({title='', message='', danger=false, withInput=false, placeholder='', confirmText=''}){
    injectStyle();
    const bg=document.createElement('div');bg.className='lc-bg';
    bg.innerHTML=`<div class="lc-modal" role="dialog">
      ${title?`<div class="lc-title">${escapeHTML(title)}</div>`:''}
      ${message?`<div class="lc-msg">${escapeHTML(message)}</div>`:''}
      ${withInput?`<input class="lc-input" placeholder="${escapeAttr(placeholder)}">`:''}
      <div class="lc-actions">
        <button class="lc-btn lc-btn-secondary" data-act="cancel">Отмена</button>
        <button class="lc-btn ${danger?'lc-btn-danger':'lc-btn-primary'}" data-act="ok">${confirmText||(withInput?'Сохранить':(danger?'Удалить':'OK'))}</button>
      </div></div>`;
    document.body.appendChild(bg);
    requestAnimationFrame(()=>bg.classList.add('open'));
    return bg;
  }
  function escapeHTML(s){return String(s||'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));}
  function escapeAttr(s){return escapeHTML(s);}
  function close(bg){bg.classList.remove('open');setTimeout(()=>bg.remove(),200);}

  window.lastopConfirm=function(opts={}){
    return new Promise(resolve=>{
      const bg=buildDialog({...opts,withInput:false});
      const onClick=e=>{
        const act=e.target.closest('[data-act]')?.dataset.act;
        if(!act)return;
        close(bg);resolve(act==='ok');document.removeEventListener('keydown',onKey);
      };
      const onKey=e=>{if(e.key==='Escape'){close(bg);resolve(false);document.removeEventListener('keydown',onKey);}};
      bg.addEventListener('click',e=>{if(e.target===bg){close(bg);resolve(false);document.removeEventListener('keydown',onKey);}});
      bg.addEventListener('click',onClick);
      document.addEventListener('keydown',onKey);
    });
  };

  // ── Universal toast: window.lastopToast(msg, type) ──
  // type: 'info' | 'success' | 'error' | 'warn'
  // Также экспортируется как window.toast / window.showToast если те не определены.
  const TOAST_ICONS = {
    success: '<svg viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg>',
    error: '<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12" y2="16"/></svg>',
    warn: '<svg viewBox="0 0 24 24"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12" y2="17"/></svg>',
    info: ''
  };
  let _lt_toast_el = null;
  let _lt_toast_timer = null;
  window.lastopToast = function(msg, type){
    injectStyle();
    if(!_lt_toast_el){
      _lt_toast_el = document.createElement('div');
      _lt_toast_el.className = 'lt-toast';
      document.body.appendChild(_lt_toast_el);
    }
    const t = (type==='ok'||type===true) ? 'success' : (type||'info');
    _lt_toast_el.className = 'lt-toast ' + t;
    const icon = TOAST_ICONS[t] || '';
    _lt_toast_el.innerHTML = icon + '<span>'+escapeHTML(String(msg||''))+'</span>';
    requestAnimationFrame(()=>_lt_toast_el.classList.add('show'));
    clearTimeout(_lt_toast_timer);
    _lt_toast_timer = setTimeout(()=>_lt_toast_el && _lt_toast_el.classList.remove('show'), 3000);
  };

  // Fallback для страниц без своего toast() / showToast()
  if(typeof window.toast !== 'function')     window.toast     = (m,ok)=>window.lastopToast(m, ok?'success':'info');
  if(typeof window.showToast !== 'function') window.showToast = (m,ok)=>window.lastopToast(m, ok?'success':'info');

  // ── Universal alert: window.lastopAlert({title,message,type}) ──
  // type: 'info' | 'success' | 'error' | 'warn'
  window.lastopAlert = function(opts={}){
    return new Promise(resolve=>{
      injectStyle();
      const type = opts.type || 'info';
      const iconSvg = TOAST_ICONS[type] || TOAST_ICONS.info;
      const bg = document.createElement('div');
      bg.className = 'lc-bg';
      bg.innerHTML = `<div class="lc-modal lc-alert type-${type}" role="dialog">
        <div class="lc-title">
          ${iconSvg?`<div class="lc-alert-icon">${iconSvg}</div>`:''}
          <span>${escapeHTML(opts.title||'Внимание')}</span>
        </div>
        ${opts.message?`<div class="lc-msg">${escapeHTML(opts.message)}</div>`:''}
        <div class="lc-actions">
          <button class="lc-btn lc-btn-primary" data-act="ok">${escapeHTML(opts.okText||'Понятно')}</button>
        </div></div>`;
      document.body.appendChild(bg);
      requestAnimationFrame(()=>bg.classList.add('open'));
      const onClick=e=>{
        if(e.target.closest('[data-act="ok"]') || e.target===bg){
          close(bg); resolve(true);
          document.removeEventListener('keydown', onKey);
        }
      };
      const onKey=e=>{if(e.key==='Escape'||e.key==='Enter'){close(bg);resolve(true);document.removeEventListener('keydown',onKey);}};
      bg.addEventListener('click', onClick);
      document.addEventListener('keydown', onKey);
    });
  };

  // ── Field-level validation helpers ──
  // lastopMarkError(inputEl, errorText) — подсветить поле + текст ошибки под ним
  // lastopClearError(inputEl) — снять подсветку
  // lastopValidate([{el, required, message}, ...]) — массовая проверка, фокус и скролл на первое невалидное; возвращает true/false
  window.lastopMarkError = function(el, message){
    if(!el) return;
    injectStyle();
    el.classList.add('lt-invalid');
    // Удалить предыдущий .lt-field-error если есть
    const next = el.nextElementSibling;
    if(next && next.classList && next.classList.contains('lt-field-error')) next.remove();
    if(message){
      const err = document.createElement('div');
      err.className = 'lt-field-error';
      err.textContent = message;
      el.parentNode && el.parentNode.insertBefore(err, el.nextSibling);
    }
    // Снимаем подсветку при редактировании
    const clear = ()=>{
      el.classList.remove('lt-invalid');
      const n = el.nextElementSibling;
      if(n && n.classList && n.classList.contains('lt-field-error')) n.remove();
      el.removeEventListener('input', clear);
      el.removeEventListener('change', clear);
    };
    el.addEventListener('input', clear);
    el.addEventListener('change', clear);
  };
  window.lastopClearError = function(el){
    if(!el) return;
    el.classList.remove('lt-invalid');
    const n = el.nextElementSibling;
    if(n && n.classList && n.classList.contains('lt-field-error')) n.remove();
  };
  window.lastopClearAllErrors = function(scope){
    const root = scope || document;
    root.querySelectorAll('.lt-invalid').forEach(el=>el.classList.remove('lt-invalid'));
    root.querySelectorAll('.lt-field-error').forEach(el=>el.remove());
  };
  // checks: [{ id?:string, el?:HTMLElement, required?:boolean, validate?:function(value)=>string|null, message?:string }]
  // Возвращает true если всё валидно, иначе false (и подсвечивает поля).
  window.lastopValidate = function(checks){
    injectStyle();
    let firstBad = null;
    checks.forEach(c=>{
      const el = c.el || (c.id ? document.getElementById(c.id) : null);
      if(!el) return;
      window.lastopClearError(el);
      const val = (el.value||'').trim();
      let err = null;
      if(c.required && !val) err = c.message || 'Это поле обязательно';
      if(!err && typeof c.validate === 'function'){
        const r = c.validate(val, el);
        if(r) err = r;
      }
      if(err){
        window.lastopMarkError(el, err);
        if(!firstBad) firstBad = el;
      }
    });
    if(firstBad){
      try { firstBad.scrollIntoView({behavior:'smooth', block:'center'}); } catch(_){}
      setTimeout(()=>{ try{ firstBad.focus(); }catch(_){ } }, 200);
      return false;
    }
    return true;
  };

  window.lastopPrompt=function(opts={}){
    return new Promise(resolve=>{
      const bg=buildDialog({...opts,withInput:true});
      const inp=bg.querySelector('.lc-input');
      setTimeout(()=>inp.focus(),50);
      const onClick=e=>{
        const act=e.target.closest('[data-act]')?.dataset.act;
        if(!act)return;
        if(act==='ok'){const v=inp.value.trim();close(bg);resolve(v||null);}
        else{close(bg);resolve(null);}
        document.removeEventListener('keydown',onKey);
      };
      const onKey=e=>{
        if(e.key==='Escape'){close(bg);resolve(null);document.removeEventListener('keydown',onKey);}
        if(e.key==='Enter'){const v=inp.value.trim();close(bg);resolve(v||null);document.removeEventListener('keydown',onKey);}
      };
      bg.addEventListener('click',e=>{if(e.target===bg){close(bg);resolve(null);document.removeEventListener('keydown',onKey);}});
      bg.addEventListener('click',onClick);
      document.addEventListener('keydown',onKey);
    });
  };
})();
