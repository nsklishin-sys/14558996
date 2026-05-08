(function(){
  // Защита от повторного запуска
  if(window.__lastopGuestNavInit)return;
  window.__lastopGuestNavInit=true;

  function hasToken(){
    try{
      var t=localStorage.getItem('token');
      return !!(t&&t.trim());
    }catch{return false;}
  }

  // Список путей доступных только авторизованным
  var BLOCKED=[
    '/chat.html','/profile.html','/profile_user.html',
    '/notifications.html','/saved.html','/tasks.html',
    '/jobs.html','/catalog.html',
    '/my-company.html','/my-community-edit.html',
    '/settings.html','/friends.html'
  ];

  function lockNavItems(){
    var items=document.querySelectorAll('.nav-item[href]');
    var lockSvg='<svg class="nav-lock" viewBox="0 0 24 24" style="width:14px;height:14px;flex-shrink:0;fill:none;stroke:#C8D8D0;stroke-width:1.6;stroke-linecap:round;margin-left:auto"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>';
    items.forEach(function(it){
      var href=it.getAttribute('href')||'';
      // Точное совпадение по пути (без query/hash)
      var path=href.split('?')[0].split('#')[0];
      if(BLOCKED.indexOf(path)===-1)return;
      it.classList.add('locked');
      it.setAttribute('href','javascript:void(0)');
      it.setAttribute('data-guest-locked','1');
      it.addEventListener('click',function(e){
        e.preventDefault();
        e.stopPropagation();
        showGuestModal();
      });
      // Добавить иконку замка если её ещё нет
      if(!it.querySelector('.nav-lock')){
        it.insertAdjacentHTML('beforeend',lockSvg);
      }
    });
  }

  function hideAuthOnlyTopbar(){
    // Заменяем выпадашку профиля на кнопки Войти/Регистрация
    var profile=document.getElementById('topbarProfile');
    if(profile && !document.getElementById('guestTopbarBtns')){
      var div=document.createElement('div');
      div.id='guestTopbarBtns';
      div.style.cssText='display:flex;align-items:center;gap:8px;flex-shrink:0';
      div.innerHTML='<a href="/login.html" style="height:32px;padding:0 14px;border-radius:99px;border:1.5px solid var(--bdr);background:var(--w);font-family:inherit;font-size:12px;font-weight:600;color:var(--tm);text-decoration:none;display:inline-flex;align-items:center;white-space:nowrap">Войти</a>'
        +'<a href="/register.html" style="height:32px;padding:0 16px;border-radius:99px;background:var(--g);color:#fff;font-family:inherit;font-size:12px;font-weight:700;text-decoration:none;display:inline-flex;align-items:center;white-space:nowrap">Регистрация</a>';
      profile.parentNode.insertBefore(div, profile);
      profile.style.display='none';
    }
    var profileDD=document.getElementById('profileDD');
    if(profileDD)profileDD.style.display='none';
    var bell=document.querySelector('.tp-bell, .topbar-bell, [data-bell]');
    if(bell)bell.style.display='none';
  }

  function showGuestModal(){
    // Используем стандартную модалку если она есть, иначе создаём свою
    var bg=document.getElementById('guestNavModalBg');
    if(!bg){
      bg=document.createElement('div');
      bg.id='guestNavModalBg';
      bg.style.cssText='position:fixed;inset:0;background:rgba(15,25,18,.45);z-index:9999;display:flex;align-items:center;justify-content:center;padding:20px;opacity:0;transition:opacity .2s';
      bg.innerHTML='<div style="background:#fff;border-radius:24px;padding:28px 24px;width:100%;max-width:360px;text-align:center;box-shadow:0 24px 60px rgba(30,138,76,.15);transform:translateY(12px);transition:transform .22s">'
        +'<div style="width:52px;height:52px;border-radius:16px;background:#E8F5EE;border:2px solid #C0DECA;display:grid;place-items:center;margin:0 auto 14px"><svg viewBox="0 0 24 24" style="width:22px;height:22px;stroke:#1E8A4C;fill:none;stroke-width:1.7;stroke-linecap:round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg></div>'
        +'<div style="font-size:15px;font-weight:800;color:#1A2A22;margin-bottom:7px">Раздел только для участников</div>'
        +'<div style="font-size:12px;color:#5A8A6A;line-height:1.6;margin-bottom:18px">Зарегистрируйтесь бесплатно, чтобы получить полный доступ к платформе LASTOP.</div>'
        +'<a href="/register.html" style="display:block;padding:10px;border-radius:12px;background:#1E8A4C;color:#fff;font-family:inherit;font-size:13px;font-weight:700;text-decoration:none;margin-bottom:8px">Зарегистрироваться бесплатно</a>'
        +'<a href="/login.html" style="display:block;padding:9px;border-radius:12px;background:transparent;border:1.5px solid #DDE8E2;color:#3A5245;font-family:inherit;font-size:12px;font-weight:600;text-decoration:none;margin-bottom:8px">Уже есть аккаунт — войти</a>'
        +'<button id="guestNavModalClose" style="font-size:11px;color:#5A8A6A;cursor:pointer;border:none;background:none;font-family:inherit">Продолжить просмотр</button>'
        +'</div>';
      document.body.appendChild(bg);
      bg.addEventListener('click',function(e){if(e.target===bg)closeGuestModal();});
      document.getElementById('guestNavModalClose').addEventListener('click',closeGuestModal);
    }
    requestAnimationFrame(function(){
      bg.style.opacity='1';
      bg.firstChild.style.transform='translateY(0)';
    });
  }
  function closeGuestModal(){
    var bg=document.getElementById('guestNavModalBg');
    if(!bg)return;
    bg.style.opacity='0';
    bg.firstChild.style.transform='translateY(12px)';
    setTimeout(function(){bg.style.display='none';},220);
  }
  window.lastopGuestModal=showGuestModal;

  function init(){
    if(hasToken())return;
    lockNavItems();
    hideAuthOnlyTopbar();
    blockActionButtons();
    interceptActionClicks();
  }

  // Скрываем кнопки создания на публичных страницах (Создать, Написать)
  function blockActionButtons(){
    var sel='.btn-create,.btn-write';
    document.querySelectorAll(sel).forEach(function(btn){
      // Просто скрываем — на публичных страницах для гостя они не нужны
      btn.style.display='none';
    });
    // Скрываем все элементы помеченные data-auth-only
    document.querySelectorAll('[data-auth-only]').forEach(function(el){
      el.style.display='none';
    });
  }

  // Перехват кликов на действия в карточках (лайк, сохранить, откликнуться, записаться)
  function interceptActionClicks(){
    var BLOCK_SEL=[
      '.nc-action',           // лайк/комментарий в карточке поста
      '.act-btn',             // news-detail: лайк/коммент/сохранить/репост
      '.btn-save',            // сохранить (мероприятие, вакансия, проект)
      '.btn-save-ex',         // сохранить выставку
      '.btn-share',           // поделиться (event, project)
      '.btn-share-co',        // поделиться сообществом
      '.btn-reg',             // записаться на мероприятие
      '.btn-respond',         // откликнуться на вакансию
      '.btn-follow',          // подписаться (news-detail)
      '.btn-write',           // написать (community/profile/event)
      '.btn-msg',             // написать (вариант)
      '.ci-btn-send',         // отправить комментарий (news-detail)
      '.ci-textarea',         // фокус на поле комментария (news-detail)
      '[data-share]',         // универсальная кнопка share через share.js
      '[data-guest-block]'    // явно помеченные элементы
    ].join(',');
    document.addEventListener('click', function(e){
      var el=e.target.closest(BLOCK_SEL);
      if(!el)return;
      // Не перехватывать кнопки навигации — у них своя логика locked
      if(el.classList.contains('locked'))return;
      e.preventDefault();
      e.stopImmediatePropagation();
      e.stopPropagation();
      showGuestModal();
    }, true); // capture phase — срабатывает до встроенного onclick

    // Перехват фокуса на полях комментариев — гость не должен
    // даже начать набирать текст, сразу модалка
    document.addEventListener('focusin', function(e){
      var el=e.target.closest('.ci-textarea,[data-guest-block-focus]');
      if(!el)return;
      el.blur();
      showGuestModal();
    }, true);
  }

  if(document.readyState==='loading'){
    document.addEventListener('DOMContentLoaded',init);
  }else{
    init();
  }
})();
