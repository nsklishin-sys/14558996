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
`;
  function injectStyle(){
    if(document.getElementById('lc-style'))return;
    const s=document.createElement('style');s.id='lc-style';s.textContent=STYLE;
    document.head.appendChild(s);
  }

  function buildDialog({title='', message='', danger=false, withInput=false, placeholder=''}){
    injectStyle();
    const bg=document.createElement('div');bg.className='lc-bg';
    bg.innerHTML=`<div class="lc-modal" role="dialog">
      ${title?`<div class="lc-title">${escapeHTML(title)}</div>`:''}
      ${message?`<div class="lc-msg">${escapeHTML(message)}</div>`:''}
      ${withInput?`<input class="lc-input" placeholder="${escapeAttr(placeholder)}">`:''}
      <div class="lc-actions">
        <button class="lc-btn lc-btn-secondary" data-act="cancel">Отмена</button>
        <button class="lc-btn ${danger?'lc-btn-danger':'lc-btn-primary'}" data-act="ok">${withInput?'Сохранить':(danger?'Удалить':'OK')}</button>
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
