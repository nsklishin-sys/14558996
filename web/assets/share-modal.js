/* lastopShare({title, url, description}) — модалка «Поделиться» с QR-кодом и копированием ссылки.
   Зависит от бэк-эндпоинта /api/share/qr (PNG). */
(function(){
  if(window.lastopShare)return;
  var MODAL=null,LAST_FOCUS=null;

  function ensure(){
    if(MODAL)return MODAL;
    MODAL=document.createElement('div');
    MODAL.className='lt-share';
    MODAL.setAttribute('role','dialog');
    MODAL.setAttribute('aria-modal','true');
    MODAL.innerHTML=
      '<div class="lt-share-bg"></div>'+
      '<div class="lt-share-box" tabindex="-1">'+
        '<div class="lt-share-head">'+
          '<div class="lt-share-h">'+
            '<div class="lt-share-title">Поделиться</div>'+
            '<div class="lt-share-sub" id="ltShareSub"></div>'+
          '</div>'+
          '<button type="button" class="lt-share-x" aria-label="Закрыть"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>'+
        '</div>'+
        '<div class="lt-share-body">'+
          '<div class="lt-share-qrwrap"><img class="lt-share-qr" id="ltShareQr" alt="QR-код"></div>'+
          '<div class="lt-share-urlrow">'+
            '<input type="text" class="lt-share-url" id="ltShareUrl" readonly>'+
            '<button type="button" class="lt-share-copy" id="ltShareCopyBtn"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>Копировать</button>'+
          '</div>'+
          '<a class="lt-share-dl" id="ltShareDl" download="qr.png"><svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>Скачать QR-код</a>'+
        '</div>'+
      '</div>';
    document.body.appendChild(MODAL);
    MODAL.querySelector('.lt-share-bg').addEventListener('click',close);
    MODAL.querySelector('.lt-share-x').addEventListener('click',close);
    MODAL.querySelector('#ltShareCopyBtn').addEventListener('click',doCopy);
    document.addEventListener('keydown',function(e){
      if(e.key==='Escape' && MODAL.classList.contains('open'))close();
    });
    return MODAL;
  }
  function doCopy(){
    var inp=document.getElementById('ltShareUrl');
    if(!inp)return;
    var done=false;
    if(navigator.clipboard && navigator.clipboard.writeText){
      navigator.clipboard.writeText(inp.value).then(function(){confirmCopy();},function(){fallback();});
      done=true;
    }
    if(!done)fallback();
    function fallback(){
      try{inp.select();inp.setSelectionRange(0,inp.value.length);document.execCommand('copy');confirmCopy();}catch(e){}
    }
  }
  function confirmCopy(){
    var b=document.getElementById('ltShareCopyBtn');
    if(!b)return;
    var orig=b.innerHTML;
    b.classList.add('done');
    b.innerHTML='<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><polyline points="20 6 9 17 4 12"/></svg>Скопировано';
    setTimeout(function(){b.classList.remove('done');b.innerHTML=orig;},1800);
    if(window.lastopToast)try{lastopToast('Ссылка скопирована','success');}catch(e){}
  }
  function open(opts){
    opts=opts||{};
    var url=String(opts.url||location.href);
    var sub=String(opts.description||opts.subtitle||'');
    var node=ensure();
    LAST_FOCUS=document.activeElement;
    document.getElementById('ltShareSub').textContent=sub||url;
    document.getElementById('ltShareUrl').value=url;
    var qrSrc=opts.qrSrc?String(opts.qrSrc):'';
    function qrURL(sz){return qrSrc?(qrSrc+(qrSrc.indexOf('?')>=0?'&':'?')+'size='+sz):('/api/share/qr?size='+sz+'&text='+encodeURIComponent(url));}
    var qr=document.getElementById('ltShareQr');
    qr.src=qrURL(320);
    document.getElementById('ltShareDl').href=qrURL(720);
    node.classList.add('open');
    setTimeout(function(){node.querySelector('.lt-share-box').focus();},40);
  }
  function close(){
    if(!MODAL)return;
    MODAL.classList.remove('open');
    if(LAST_FOCUS && LAST_FOCUS.focus)try{LAST_FOCUS.focus();}catch(e){}
  }
  window.lastopShare=open;
  window.lastopShareClose=close;
})();
