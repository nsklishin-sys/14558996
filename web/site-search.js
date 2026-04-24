(function initGlobalSiteSearch(){
  const SEARCH_SECTIONS = [
    { title: 'Главная', url: '/home-auth.html', keywords: ['главная','home','лента','новости'] },
    { title: 'Профиль', url: '/profile.html', keywords: ['профиль','profile','аккаунт','страница'] },
    { title: 'Чат', url: '/chat.html', keywords: ['чат','сообщения','диалоги','chat'] },
    { title: 'Настройки', url: '/settings.html', keywords: ['настройки','settings','параметры'] },
    { title: 'Новости', url: '/dashboard.html', keywords: ['новости','news','обновления'] },
    { title: 'Проекты', url: '/projects.html', keywords: ['проекты','project','кейсы'] },
    { title: 'Мероприятия', url: '/events.html', keywords: ['мероприятия','events','ивенты','конференции'] },
    { title: 'Выставки', url: '/exhibitions.html', keywords: ['выставки','exhibitions','expo'] },
    { title: 'Резюме', url: '/jobs.html', keywords: ['резюме','вакансии','работа','jobs'] },
    { title: 'Товары и услуги', url: '/catalog.html', keywords: ['товары','услуги','каталог','catalog'] },
    { title: 'Форум', url: '/forum.html', keywords: ['форум','forum','обсуждения'] },
    { title: 'Компании', url: '/companies.html', keywords: ['компании','бизнес','организации','companies'] },
    { title: 'Сообщества', url: '/communities.html', keywords: ['сообщества','groups','communities'] }
  ];

  function normalize(text){
    return (text || '')
      .toString()
      .toLowerCase()
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .replace(/ё/g, 'е')
      .replace(/[^a-zа-я0-9\s-]/gi, ' ')
      .replace(/\s+/g, ' ')
      .trim();
  }

  function getBestMatch(query){
    const normalizedQuery = normalize(query);
    if (!normalizedQuery) return null;

    const terms = normalizedQuery.split(' ').filter(Boolean);
    let best = null;

    for (const section of SEARCH_SECTIONS){
      const haystack = normalize([section.title, ...(section.keywords || [])].join(' '));
      let score = 0;

      for (const term of terms){
        if (!term) continue;
        if (haystack.includes(term)) score += term.length > 3 ? 2 : 1;
      }

      if (haystack.includes(normalizedQuery)) score += 3;

      if (!best || score > best.score){
        best = { section, score };
      }
    }

    return best && best.score > 0 ? best.section : null;
  }

  async function runSearch(query){
    const cleaned = (query || '').trim();
    if (!cleaned) return;
    if (cleaned.startsWith('@')){
      try{
        const token = localStorage.getItem('token') || '';
        const prefix = cleaned.slice(1);
        const r = await fetch(`/api/users/search?prefix_handle=${encodeURIComponent(prefix)}`, { headers: token ? { Authorization: `Bearer ${token}` } : {} });
        const d = await r.json();
        const u = (d.users || [])[0];
        if (u?.id){
          window.location.href = `/profile_user.html?id=${encodeURIComponent(u.id)}`;
          return;
        }
      }catch(_){}
    }

    const match = getBestMatch(cleaned);
    if (match){
      const url = new URL(match.url, window.location.origin);
      url.searchParams.set('q', cleaned);
      window.location.href = url.toString();
      return;
    }

    const fallbackUrl = new URL('/home-auth.html', window.location.origin);
    fallbackUrl.searchParams.set('q', cleaned);
    window.location.href = fallbackUrl.toString();
  }

  function attachSearch(input){
    if (!input || input.dataset.searchReady === '1') return;
    input.dataset.searchReady = '1';
    input.style.background = '#FFFFFF';

    input.addEventListener('keydown', (event) => {
      if (event.key !== 'Enter') return;
      event.preventDefault();
      runSearch(input.value);
    });

    input.addEventListener('search', () => {
      runSearch(input.value);
    });
  }

  function boot(){
    document.querySelectorAll('#searchInput').forEach(attachSearch);
  }

  if (document.readyState === 'loading'){
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }
})();
