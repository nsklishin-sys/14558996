// LASTOP — стилизованный dropdown для <select data-lt-select> и <select data-category-picker>.
// Превращает любой нативный <select> с этим атрибутом в кастомный popup с группами и поиском.
// Нативный <select> остаётся в DOM, скрыт визуально, через него работает форма и change event.
(function(){
  const STYLE = `
.lt-sel-wrap{position:relative;width:100%}
.lt-sel-native{position:absolute !important;width:1px !important;height:1px !important;padding:0 !important;border:0 !important;clip:rect(0 0 0 0) !important;overflow:hidden !important;opacity:0 !important;pointer-events:none !important}
.lt-sel-btn{width:100%;height:38px;padding:0 36px 0 12px;border:1.5px solid #DDE8E2;border-radius:12px;background:#F0FAF4;font-family:'Manrope',sans-serif;font-size:13px;color:#1A2A22;cursor:pointer;text-align:left;display:flex;align-items:center;position:relative;transition:border-color .15s}
.lt-sel-btn:hover{border-color:#C0DECA}
.lt-sel-btn.open,.lt-sel-btn:focus{border-color:#22A05A;background:#fff;outline:none}
.lt-sel-btn.empty{color:#5A8A6A}
.lt-sel-btn::after{content:position:absolute;right:14px;top:50%;width:8px;height:8px;border-right:1.8px solid #5A8A6A;border-bottom:1.8px solid #5A8A6A;transform:translateY(-65%) rotate(45deg);transition:transform .15s}
.lt-sel-btn.open::after{transform:translateY(-30%) rotate(-135deg)}
.lt-sel-pop{position:absolute;left:0;right:0;top:calc(100% + 4px);background:#fff;border:1px solid #DDE8E2;border-radius:12px;box-shadow:0 12px 32px rgba(30,138,76,.15);z-index:1000;max-height:320px;overflow:hidden;display:none;flex-direction:column;font-family:'Manrope',sans-serif}
.lt-sel-pop.open{display:flex}
.lt-sel-search{padding:8px;border-bottom:1px solid #DDE8E2;flex-shrink:0}
.lt-sel-search input{width:100%;height:32px;padding:0 12px;border:1.5px solid #DDE8E2;border-radius:8px;font-family:inherit;font-size:12px;color:#1A2A22;outline:none;background:#F0FAF4}
.lt-sel-search input:focus{border-color:#22A05A;background:#fff}
.lt-sel-list{flex:1;overflow-y:auto;padding:4px}
.lt-sel-list::-webkit-scrollbar{width:6px}
.lt-sel-list::-webkit-scrollbar-thumb{background:#C0DECA;border-radius:99px}
.lt-sel-group{font-size:10px;font-weight:800;color:#5A8A6A;text-transform:uppercase;letter-spacing:.06em;padding:8px 10px 4px}
.lt-sel-item{padding:8px 10px;font-size:13px;color:#1A2A22;cursor:pointer;border-radius:6px;transition:background .1s,color .1s}
.lt-sel-item:hover{background:#F0FAF4;color:#1E8A4C}
.lt-sel-item.active{background:#E8F5EE;color:#1E8A4C;font-weight:700}
.lt-sel-item.placeholder{color:#5A8A6A}
.lt-sel-empty{padding:14px;text-align:center;font-size:12px;color:#5A8A6A}
`;
  function inject(){
    if(document.getElementById('lt-sel-style'))return;
    const s=document.createElement('style');s.id='lt-sel-style';s.textContent=STYLE;document.head.appendChild(s);
  }
  function esc(s){return String(s||'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));}

  function build(nativeSel){
    if(nativeSel.dataset.ltSelBound)return;
    nativeSel.dataset.ltSelBound='1';
    inject();
    const wrap=document.createElement('div');
    wrap.className='lt-sel-wrap';
    nativeSel.parentNode.insertBefore(wrap,nativeSel);
    wrap.appendChild(nativeSel);
    nativeSel.classList.add('lt-sel-native');
    const btn=document.createElement('button');
    btn.type='button';btn.className='lt-sel-btn empty';
    btn.innerHTML='<span class="lt-sel-btn-label"></span>';
    const pop=document.createElement('div');pop.className='lt-sel-pop';
    pop.innerHTML='<div class="lt-sel-search"><input type="text" placeholder="Поиск…" autocomplete="off"></div><div class="lt-sel-list"></div>';
    wrap.appendChild(btn);wrap.appendChild(pop);

    const search=pop.querySelector('input');
    const list=pop.querySelector('.lt-sel-list');
    const label=btn.querySelector('.lt-sel-btn-label');

    function syncLabel(){
      const sel=nativeSel.options[nativeSel.selectedIndex];
      const txt=sel?sel.textContent:
      const isPlaceholder=!nativeSel.value || (sel && !sel.value);
      label.textContent=txt||'Выбрать…';
      btn.classList.toggle('empty',isPlaceholder);
    }

    function renderList(filter){
      const f=(filter||'').trim().toLowerCase();
      let html=
      let total=0;
      // optgroups
      const groups=nativeSel.querySelectorAll('optgroup');
      if(groups.length){
        groups.forEach(g=>{
          const opts=Array.from(g.querySelectorAll('option')).filter(o=>{
            if(!f)return true;
            return o.textContent.toLowerCase().includes(f);
          });
          if(!opts.length)return;
          html+=`<div class="lt-sel-group">${esc(g.label)}</div>`;
          opts.forEach(o=>{
            const isAct=o.value===nativeSel.value;
            const isPh=!o.value;
            html+=`<div class="lt-sel-item${isAct?' active':''}${isPh?' placeholder':''}" data-val="${esc(o.value)}">${esc(o.textContent)}</div>`;
            total++;
          });
        });
        // плоские опции вне групп
        const flatTopOpts=Array.from(nativeSel.children).filter(c=>c.tagName==='OPTION');
        flatTopOpts.forEach(o=>{
          if(f && !o.textContent.toLowerCase().includes(f))return;
          const isAct=o.value===nativeSel.value;
          const isPh=!o.value;
          html=`<div class="lt-sel-item${isAct?' active':''}${isPh?' placeholder':''}" data-val="${esc(o.value)}">${esc(o.textContent)}</div>`+html;
          total++;
        });
      } else {
        // нет групп — просто список
        Array.from(nativeSel.options).forEach(o=>{
          if(f && !o.textContent.toLowerCase().includes(f))return;
          const isAct=o.value===nativeSel.value;
          const isPh=!o.value;
          html+=`<div class="lt-sel-item${isAct?' active':''}${isPh?' placeholder':''}" data-val="${esc(o.value)}">${esc(o.textContent)}</div>`;
          total++;
        });
      }
      if(!total)html='<div class="lt-sel-empty">Ничего не найдено</div>';
      list.innerHTML=html;
    }

    function open(){
      // Закрыть остальные
      document.querySelectorAll('.lt-sel-pop.open').forEach(p=>{if(p!==pop){p.classList.remove('open');const b=p.previousSibling;if(b&&b.classList)b.classList.remove('open');}});
      pop.classList.add('open');btn.classList.add('open');
      search.value=
      renderList('');
      setTimeout(()=>search.focus(),20);
    }
    function close(){pop.classList.remove('open');btn.classList.remove('open');}

    btn.addEventListener('click',e=>{
      e.stopPropagation();
      if(pop.classList.contains('open'))close();else open();
    });
    search.addEventListener('input',()=>renderList(search.value));
    search.addEventListener('keydown',e=>{if(e.key==='Escape')close();});
    list.addEventListener('click',e=>{
      const it=e.target.closest('.lt-sel-item');if(!it)return;
      const val=it.dataset.val;
      nativeSel.value=val;
      // диспатч change чтобы существующий код страницы реагировал как на обычный select
      nativeSel.dispatchEvent(new Event('change',{bubbles:true}));
      syncLabel();
      close();
    });
    // Внешний клик — закрыть
    document.addEventListener('click',e=>{
      if(!wrap.contains(e.target))close();
    });
    // Слежение за внешними изменениями value (например при загрузке существующего event для редактирования)
    const mo=new MutationObserver(()=>syncLabel());
    mo.observe(nativeSel,{attributes:true,attributeFilter:['value']});
    nativeSel.addEventListener('change',syncLabel);
    // Также — если страница после рендера набивает options и устанавливает value программно
    setTimeout(syncLabel,50);
    syncLabel();
  }

  function scan(root){
    if(!root||!root.querySelectorAll)return;
    root.querySelectorAll('select[data-lt-select],select[data-category-picker]').forEach(build);
  }

  function init(){
    scan(document);
    if(window.MutationObserver){
      const obs=new MutationObserver(muts=>{
        muts.forEach(m=>{
          m.addedNodes.forEach(n=>{
            if(n.nodeType!==1)return;
            if(n.matches&&n.matches('select[data-lt-select],select[data-category-picker]'))build(n);
            scan(n);
          });
        });
      });
      obs.observe(document.body,{childList:true,subtree:true});
    }
  }
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',init);else init();

  window.LASTOP_initSelect=build;
})();
