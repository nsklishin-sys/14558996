// ── Active Context Helpers ──────────────────────────────────────
// Глобальная утилита для прокидывания активного контекста (компания/сообщество)
// в любые fetch-запросы. Подключается на каждой странице через htmlInject.
//
// API:
//   withActiveContextHeaders(headers={})  — возвращает копию headers с X-Active-*
//   getActiveContext()                    — { kind: 'user'|'company'|'community', id: number }
//   isActiveCompanyContext()              — bool
//   isActiveCommunityContext()            — bool

(function () {
  if (window.withActiveContextHeaders) return;

  function getActiveContext() {
    try {
      const co = localStorage.getItem('active_company_id');
      if (co && Number(co) > 0) return { kind: 'company', id: Number(co) };
      const cm = localStorage.getItem('active_community_id');
      if (cm && Number(cm) > 0) return { kind: 'community', id: Number(cm) };
    } catch {}
    return { kind: 'user', id: 0 };
  }

  function withActiveContextHeaders(base) {
    const h = Object.assign({}, base || {});
    const ctx = getActiveContext();
    if (ctx.kind === 'company') h['X-Active-Company-Id'] = String(ctx.id);
    else if (ctx.kind === 'community') h['X-Active-Community-Id'] = String(ctx.id);
    return h;
  }

  window.withActiveContextHeaders = withActiveContextHeaders;
  window.getActiveContext = getActiveContext;
  window.isActiveCompanyContext = function () { return getActiveContext().kind === 'company'; };
  window.isActiveCommunityContext = function () { return getActiveContext().kind === 'community'; };
})();
