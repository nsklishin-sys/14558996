(() => {
  if (window.lastopSkel) return;

  // Inject CSS once
  const CSS = `
.lt-skel-pulse {
  background: linear-gradient(90deg,
    var(--bg, #F2F5F3) 0%,
    rgba(0,0,0,0.04) 50%,
    var(--bg, #F2F5F3) 100%);
  background-size: 200% 100%;
  animation: lt-skel-shimmer 1.4s ease-in-out infinite;
  border-radius: 6px;
}
@keyframes lt-skel-shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
.lt-skel-card {
  background: var(--w, #fff);
  border: 1px solid var(--bdr, #DDE8E2);
  border-radius: 18px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 140px;
}
.lt-skel-row { display: flex; gap: 10px; align-items: center; }
.lt-skel-circle { width: 38px; height: 38px; border-radius: 10px; flex-shrink: 0; }
.lt-skel-line { height: 11px; border-radius: 4px; flex: 1; }
.lt-skel-line.short { flex: 0 0 60%; }
.lt-skel-line.mini { flex: 0 0 30%; height: 9px; }
.lt-skel-line.tall { height: 14px; }
.lt-skel-block { height: 100px; border-radius: 10px; }
`;

  function injectCSS() {
    if (document.getElementById('lt-skel-styles')) return;
    const style = document.createElement('style');
    style.id = 'lt-skel-styles';
    style.textContent = CSS;
    document.head.appendChild(style);
  }

  function cards(n) {
    injectCSS();
    n = Math.max(1, Math.min(n || 4, 20));
    let html = '';
    for (let i = 0; i < n; i++) {
      html += `
        <div class="lt-skel-card">
          <div class="lt-skel-row">
            <div class="lt-skel-pulse lt-skel-circle"></div>
            <div style="flex:1;display:flex;flex-direction:column;gap:6px;min-width:0">
              <div class="lt-skel-pulse lt-skel-line tall short"></div>
              <div class="lt-skel-pulse lt-skel-line mini"></div>
            </div>
          </div>
          <div class="lt-skel-pulse lt-skel-line"></div>
          <div class="lt-skel-pulse lt-skel-line short"></div>
        </div>`;
    }
    return html;
  }

  function inject(containerOrId, n) {
    injectCSS();
    const el = typeof containerOrId === 'string'
      ? document.getElementById(containerOrId)
      : containerOrId;
    if (!el) return;
    el.innerHTML = cards(n);
  }

  window.lastopSkel = { cards, inject };
})();
