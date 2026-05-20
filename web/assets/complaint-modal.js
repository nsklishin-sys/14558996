(function(){
  if (window.lastopComplain) return;

  const REASONS = [
    { key: 'spam',      label: 'Спам или реклама' },
    { key: 'abuse',     label: 'Оскорбления, угрозы' },
    { key: 'illegal',   label: 'Запрещённый контент' },
    { key: 'fraud',     label: 'Мошенничество' },
    { key: 'disinfo',   label: 'Дезинформация' },
    { key: 'copyright', label: 'Нарушение авторских прав' },
    { key: 'other',     label: 'Другое' },
  ];

  function injectStyles(){
    if (document.getElementById('complaintModalStyles')) return;
    const css = `
.complaint-bg{position:fixed;inset:0;background:rgba(15,25,18,.5);backdrop-filter:blur(4px);z-index:1000;display:flex;align-items:center;justify-content:center;padding:20px;opacity:0;transition:opacity .2s;pointer-events:none}
.complaint-bg.open{opacity:1;pointer-events:auto}
.complaint-modal{background:#fff;border-radius:18px;padding:22px;width:100%;max-width:460px;max-height:90vh;overflow-y:auto;box-shadow:0 20px 60px rgba(15,25,18,.18);transform:translateY(12px);transition:transform .22s;font-family:inherit}
.complaint-bg.open .complaint-modal{transform:translateY(0)}
.complaint-head{display:flex;align-items:center;justify-content:space-between;margin-bottom:16px}
.complaint-head h3{font-size:16px;font-weight:800;color:#1A2A22;margin:0}
.complaint-x{background:#F0FAF4;border:none;width:30px;height:30px;border-radius:8px;cursor:pointer;color:#5A8A6A;font-size:18px;display:grid;place-items:center}
.complaint-x:hover{background:#E8F5EE;color:#1E8A4C}
.complaint-label{font-size:11px;font-weight:700;color:#5A8A6A;text-transform:uppercase;letter-spacing:.05em;margin-bottom:6px;display:block}
.complaint-reasons{display:flex;flex-direction:column;gap:6px;margin-bottom:14px}
.complaint-reason{display:flex;align-items:center;gap:9px;padding:9px 11px;border-radius:10px;border:1.5px solid #DDE8E2;cursor:pointer;font-size:13px;color:#3A5245;transition:all .15s}
.complaint-reason:hover{border-color:#C0DECA;background:#F0FAF4}
.complaint-reason input{margin:0;accent-color:#1E8A4C}
.complaint-reason.active{border-color:#1E8A4C;background:#E8F5EE;color:#1A2A22;font-weight:600}
.complaint-textarea{width:100%;border:1.5px solid #DDE8E2;border-radius:10px;padding:10px 12px;font-family:inherit;font-size:13px;color:#1A2A22;outline:none;resize:none;height:80px;background:#F0FAF4;line-height:1.5;margin-bottom:14px}
.complaint-textarea:focus{border-color:#22A05A}
.complaint-screenshot-area{margin-bottom:14px}
.complaint-screenshot-btn{display:flex;align-items:center;justify-content:center;gap:8px;width:100%;padding:11px;border:1.5px dashed #C0DECA;border-radius:10px;background:#F0FAF4;color:#5A8A6A;font-family:inherit;font-size:13px;font-weight:600;cursor:pointer;transition:all .15s}
.complaint-screenshot-btn:hover{border-color:#1E8A4C;background:#E8F5EE;color:#1E8A4C}
.complaint-screenshot-btn svg{width:16px;height:16px;fill:none;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round}
.complaint-screenshot-preview{position:relative;border-radius:10px;overflow:hidden;border:1.5px solid #DDE8E2}
.complaint-screenshot-preview img{display:block;width:100%;max-height:200px;object-fit:cover}
.complaint-screenshot-remove{position:absolute;top:8px;right:8px;background:rgba(15,25,18,.7);color:#fff;border:none;width:26px;height:26px;border-radius:6px;cursor:pointer;display:grid;place-items:center;font-size:16px;line-height:1}
.complaint-screenshot-remove:hover{background:#E04040}
.complaint-footer{display:flex;gap:8px;justify-content:flex-end;margin-top:16px;padding-top:12px;border-top:1px solid #DDE8E2}
.complaint-btn-sec{padding:8px 18px;border-radius:10px;border:1.5px solid #DDE8E2;background:#fff;font-family:inherit;font-size:13px;font-weight:600;color:#3A5245;cursor:pointer}
.complaint-btn-sec:hover{background:#F0FAF4}
.complaint-btn-pri{padding:8px 20px;border-radius:10px;border:none;background:#1E8A4C;font-family:inherit;font-size:13px;font-weight:600;color:#fff;cursor:pointer}
.complaint-btn-pri:hover{background:#22A05A}
.complaint-btn-pri:disabled{background:#C0DECA;cursor:not-allowed}
.complaint-toast{position:fixed;bottom:24px;left:50%;transform:translateX(-50%) translateY(20px);background:#fff;color:#1A2A22;padding:13px 18px 13px 14px;border-radius:12px;font-size:13px;font-weight:600;z-index:1100;opacity:0;transition:all .25s;pointer-events:none;max-width:90vw;display:flex;align-items:center;gap:10px;box-shadow:0 8px 28px rgba(15,25,18,.18);border:1.5px solid #1E8A4C}
.complaint-toast.show{opacity:1;transform:translateX(-50%) translateY(0)}
.complaint-toast .ct-ico{width:24px;height:24px;border-radius:50%;background:#1E8A4C;color:#fff;display:grid;place-items:center;flex-shrink:0}
.complaint-toast .ct-ico svg{width:14px;height:14px;fill:none;stroke:currentColor;stroke-width:2.5;stroke-linecap:round;stroke-linejoin:round}
.complaint-toast.err{border-color:#E04040}
.complaint-toast.err .ct-ico{background:#E04040}
`;
    const style = document.createElement('style');
    style.id = 'complaintModalStyles';
    style.textContent = css;
    document.head.appendChild(style);
  }

  function showToast(msg, isError){
    let t = document.querySelector('.complaint-toast');
    if (!t) {
      t = document.createElement('div');
      t.className = 'complaint-toast';
      document.body.appendChild(t);
    }
    t.classList.toggle('err', !!isError);
    const icoSVG = isError
      ? '<svg viewBox="0 0 24 24"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>'
      : '<svg viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg>';
    t.innerHTML = '<span class="ct-ico">' + icoSVG + '</span><span>' + String(msg) + '</span>';
    t.classList.add('show');
    setTimeout(() => t.classList.remove('show'), 3500);
  }

  window.lastopComplain = function(targetType, targetID){
    injectStyles();
    return Promise.resolve(true);
  };
})();
