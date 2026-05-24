(function(){
  'use strict';

  const API='/api';
  const tk=()=>{try{return JSON.parse(localStorage.getItem('user')||'{}').id?'cookie':'';}catch(_){return '';}};
  const DEBOUNCE_MS=280;

  function esc(s){return String(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');}
  function letter(s){return((s||'?')[0]||'?').toUpperCase();}
  const C=['#5AB080','#3A90C0','#9060C0','#C07030','#1A8A6A','#B05090','#208090','#3B6D11'];
  function avc(s){return C[(s||'').charCodeAt(0)%C.length]||C[0];}
  function highlight(text,q){
    if(!text)return '';
    const safe=esc(String(text));
    if(!q)return safe;
    const terms=q.toLowerCase().split(/\s+/).filter(Boolean).map(t=>t.replace(/[.*+?^${}()|[\]\\]/g,'\\$&'));
    if(!terms.length)return safe;
    return safe.replace(new RegExp('('+terms.join('|')+')','gi'),'<mark>$1</mark>');
  }

  // CSS dropdown — инжектируем один раз
  let stylesInjected=false;
  function injectStyles(){
    if(stylesInjected)return;
    stylesInjected=true;
    const css=`
      .gss-wrap{position:relative}
      .gss-dd{
        position:absolute;top:calc(100% + 6px);left:0;right:0;
        background:var(--w,#fff);border:1px solid var(--bdr,#DDE8E2);border-radius:14px;
        box-shadow:0 12px 36px rgba(30,138,76,.15);
        max-height:520px;overflow-y:auto;z-index:300;
        opacity:0;pointer-events:none;transform:translateY(-4px);
        transition:opacity .15s,transform .15s
      }
      .gss-dd.open{opacity:1;pointer-events:auto;transform:translateY(0)}
      .gss-group-head{
        padding:8px 14px;font-size:9px;font-weight:700;color:var(--gmt,#5A8A6A);
        text-transform:uppercase;letter-spacing:.1em;background:var(--gp,#F0FAF4);
        border-top:1px solid var(--bdr,#DDE8E2);position:sticky;top:0;z-index:1
      }
      .gss-group-head:first-child{border-top:none}
      .gss-row{
        display:flex;align-items:center;gap:10px;padding:8px 14px;
        cursor:pointer;text-decoration:none;color:inherit;transition:background .12s
      }
      .gss-row:hover,.gss-row.active{background:var(--gp,#F0FAF4)}
      .gss-av{
        width:32px;height:32px;border-radius:9px;display:grid;place-items:center;
        font-size:11px;font-weight:800;color:#fff;flex-shrink:0
      }
      .gss-text{flex:1;min-width:0;display:flex;flex-direction:column;line-height:1.3}
      .gss-title{font-size:13px;font-weight:700;color:var(--t,#1A2A22);white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
      .gss-sub{font-size:11px;color:var(--gmt,#5A8A6A);white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
      .gss-handle{font-size:11px;color:var(--g,#1E8A4C);font-weight:700;margin-left:6px}
      .gss-row mark{background:#FFF6CC;color:inherit;font-weight:700;padding:0 2px;border-radius:3px}
      .gss-foot{
        display:flex;align-items:center;justify-content:space-between;gap:10px;
        padding:11px 14px;background:var(--gp,#F0FAF4);border-top:1px solid var(--bdr,#DDE8E2);
        font-size:12px;font-weight:700;color:var(--g,#1E8A4C);cursor:pointer;
        position:sticky;bottom:0
      }
      .gss-foot:hover{background:var(--gl,#E8F5EE)}
      .gss-foot svg{width:13px;height:13px;stroke:currentColor;fill:none;stroke-width:2.2;stroke-linecap:round;stroke-linejoin:round}
      .gss-empty{padding:32px 14px;text-align:center;font-size:12px;color:var(--gmt,#5A8A6A)}
      .gss-loading{padding:18px;text-align:center;font-size:11px;color:var(--gmt,#5A8A6A)}
    `;
    const tag=document.createElement('style');
    tag.id='gss-styles';
    tag.textContent=css;
    document.head.appendChild(tag);
  }

  function buildDD(input){
    const wrap=input.closest('.search-wrap');
    if(!wrap)return null;
    wrap.classList.add('gss-wrap');
    let dd=wrap.querySelector('.gss-dd');
    if(!dd){
      dd=document.createElement('div');
      dd.className='gss-dd';
      wrap.appendChild(dd);
    }
    return dd;
  }

  function attach(input){
    if(!input||input.dataset.gssReady==='1')return;
    input.dataset.gssReady='1';
    input.setAttribute('autocomplete','off');
    input.setAttribute('spellcheck','false');
    injectStyles();
    const dd=buildDD(input);
    if(!dd)return;

    const wrap=input.closest('.search-wrap');
    let timer=null,lastQ='',activeIdx=-1,items=[];

    function close(){dd.classList.remove('open');activeIdx=-1;}
    function open(){dd.classList.add('open');}

    document.addEventListener('click',e=>{if(!wrap.contains(e.target))close();});

    input.addEventListener('focus',()=>{if(input.value.trim())trigger(input.value);});
    input.addEventListener('input',()=>{
      clearTimeout(timer);
      const v=input.value.trim();
      if(!v){close();return;}
      timer=setTimeout(()=>trigger(v),DEBOUNCE_MS);
    });
    input.addEventListener('keydown',e=>{
      if(e.key==='Enter'){
        e.preventDefault();
        if(activeIdx>=0&&items[activeIdx]){
          location.href=items[activeIdx].url;
        }else{
          const v=input.value.trim();
          if(!v)return;
          location.href='/search.html?q='+encodeURIComponent(v);
        }
        return;
      }
      if(e.key==='Escape'){close();input.blur();return;}
      if(e.key==='ArrowDown'){e.preventDefault();move(1);return;}
      if(e.key==='ArrowUp'){e.preventDefault();move(-1);return;}
    });

    function move(d){
      if(!items.length)return;
      activeIdx=(activeIdx+d+items.length)%items.length;
      [...dd.querySelectorAll('.gss-row')].forEach((el,i)=>el.classList.toggle('active',i===activeIdx));
      const cur=dd.querySelectorAll('.gss-row')[activeIdx];
      if(cur)cur.scrollIntoView({block:'nearest'});
    }

    async function trigger(q){
      if(q===lastQ)return;
      lastQ=q;
      open();
      dd.innerHTML='<div class="gss-loading">Ищем…</div>';
      activeIdx=-1;items=[];

      // @handle — прямой поиск пользователей
      if(q.startsWith('@')){
        try{
          const handle=q.slice(1);
          const r=await fetch(`${API}/users/search?prefix_handle=${encodeURIComponent(handle)}`,{headers:{}});
          const d=r.ok?await r.json():{users:[]};
          render({users:d.users||[],posts:[],communities:[],events:[],companies:[]},q,true);
          return;
        }catch(_){
          render({users:[],posts:[],communities:[],events:[],companies:[]},q,true);
          return;
        }
      }

      // Главный эндпоинт
      try{
        const r=await fetch(`${API}/search?q=${encodeURIComponent(q)}&limit=8`,{headers:{}});
        if(r.ok){
          const d=await r.json();
          render(d,q,false);
          return;
        }
      }catch(_){}

      // Fallback: только пользователи + переход на полный поиск
      try{
        const r=await fetch(`${API}/users/search?q=${encodeURIComponent(q)}&limit=5`,{headers:{}});
        const d=r.ok?await r.json():{users:[]};
        render({users:d.users||[],posts:[],communities:[],events:[],companies:[]},q,false);
      }catch(_){
        render({users:[],posts:[],communities:[],events:[],companies:[]},q,false);
      }
    }

    function render(d,q,handleMode){
      items=[];
      const limits={users:handleMode?8:3,posts:3,communities:2,events:2,companies:2};
      let html='';
      const sections=[
        ['users','Люди',d.users,uRow],
        ['posts','Новости',d.posts,pRow],
        ['communities','Сообщества',d.communities,cRow],
        ['events','Мероприятия',d.events,eRow],
        ['companies','Компании',d.companies,coRow]
      ];
      let any=false;
      for(const[key,title,arr,fn]of sections){
        const a=(arr||[]).slice(0,limits[key]||3);
        if(!a.length)continue;
        any=true;
        html+=`<div class="gss-group-head">${esc(title)}</div>`;
        for(const it of a){
          const{html:row,url}=fn(it,q);
          items.push({url});
          html+=row;
        }
      }
      if(!any){
        html+='<div class="gss-empty">Ничего не найдено</div>';
      }
      html+=`<a class="gss-foot" href="/search.html?q=${encodeURIComponent(q)}">
        <span>Все результаты по запросу «${esc(q.length>30?q.slice(0,30)+'…':q)}»</span>
        <svg viewBox="0 0 24 24"><polyline points="9 18 15 12 9 6"/></svg>
      </a>`;
      items.push({url:'/search.html?q='+encodeURIComponent(q)});
      dd.innerHTML=html;
      activeIdx=-1;
    }

    function uRow(u,q){
      const name=u.full_name||((u.first_name||'')+' '+(u.last_name||'')).trim()||u.email||'?';
      const handle=u.handle?'@'+u.handle:'';
      const sub=[u.position,u.company_name].filter(Boolean).join(' · ');
      const url='/profile_user.html?id='+encodeURIComponent(u.public_id||u.id||'');
      return {url,html:`<a class="gss-row" href="${url}">
        <div class="gss-av" style="background:${avc(name)};overflow:hidden">${u.avatar?`<img src="${esc(u.avatar)}" alt="" style="width:100%;height:100%;object-fit:cover">`:esc(letter(name))}</div>
        <div class="gss-text">
          <div class="gss-title">${highlight(name,q)}${handle?`<span class="gss-handle">${esc(handle)}</span>`:''}</div>
          ${sub?`<div class="gss-sub">${highlight(sub,q)}</div>`:''}
        </div>
      </a>`};
    }
    function pRow(p,q){
      let author = p.author_name || 'LASTOP';
      if (p.author_company_id && p.author_company_name) author = p.author_company_name;
      else if (p.author_community_id && p.author_community_name) author = p.author_community_name;
      const text=p.content||'';
      const preview=text.length>80?text.slice(0,80)+'…':text;
      const url='/news-detail.html?id='+encodeURIComponent(p.public_id||p.id||'');
      return {url,html:`<a class="gss-row" href="${url}">
        <div class="gss-av" style="background:var(--blue,#3A90C0);border-radius:7px">
          <svg viewBox="0 0 24 24" style="width:14px;height:14px;stroke:#fff;fill:none;stroke-width:2;stroke-linecap:round"><path d="M4 6h16M4 12h16M4 18h10"/></svg>
        </div>
        <div class="gss-text">
          <div class="gss-title">${highlight(preview,q)}</div>
          <div class="gss-sub">${esc(author)}${p.category?' · '+esc(p.category):''}</div>
        </div>
      </a>`};
    }
    function cRow(c,q){
      const name=c.name||'';
      const url='/community-detail.html?id='+encodeURIComponent(c.public_id||c.id||'');
      return {url,html:`<a class="gss-row" href="${url}">
        <div class="gss-av" style="background:${c.color||'var(--purple,#9060C0)'};border-radius:7px">${esc(letter(name))}</div>
        <div class="gss-text">
          <div class="gss-title">${highlight(name,q)}</div>
          <div class="gss-sub">${[c.region,c.category].filter(Boolean).map(x=>esc(x)).join(' · ')}</div>
        </div>
      </a>`};
    }
    function eRow(ev,q){
      const url='/event-detail.html?id='+encodeURIComponent(ev.public_id||ev.id||'');
      const date=ev.starts_at?new Date(ev.starts_at).toLocaleDateString('ru-RU',{day:'numeric',month:'short'}):'';
      return {url,html:`<a class="gss-row" href="${url}">
        <div class="gss-av" style="background:${ev.banner_color||'var(--orange,#C07030)'};border-radius:7px">
          <svg viewBox="0 0 24 24" style="width:14px;height:14px;stroke:#fff;fill:none;stroke-width:2;stroke-linecap:round"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
        </div>
        <div class="gss-text">
          <div class="gss-title">${highlight(ev.title||'',q)}</div>
          <div class="gss-sub">${date}${ev.city?' · '+esc(ev.city):''}</div>
        </div>
      </a>`};
    }
    function coRow(co,q){
      const name=co.name||co.company_name||'';
      const url=co.public_id?'/company-detail.html?id='+encodeURIComponent(co.public_id):'/companies.html?q='+encodeURIComponent(name);
      return {url,html:`<a class="gss-row" href="${url}">
        <div class="gss-av" style="background:#208090;border-radius:7px">${esc(letter(name))}</div>
        <div class="gss-text">
          <div class="gss-title">${highlight(name,q)}</div>
          <div class="gss-sub">${[co.industry,co.city].filter(Boolean).map(x=>esc(x)).join(' · ')}</div>
        </div>
      </a>`};
    }
  }

  function boot(){
    document.querySelectorAll('#searchInput').forEach(attach);
  }
  if(document.readyState==='loading'){document.addEventListener('DOMContentLoaded',boot);}
  else{boot();}
})();
