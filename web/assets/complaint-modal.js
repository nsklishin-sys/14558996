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
.complaint-modal{background:#fff;border-radius:18px;padding:22px;width:100%;max-width:440px;max-height:90vh;overflow-y:auto;box-shadow:0 20px 60px rgba(15,25,18,.18);transform:translateY(12px);transition:transform .22s;font-family:inherit}
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
.complaint-textarea{width:100%;border:1.5px solid #DDE8E2;border-radius:10px;padding:10px 12px;font-family:inherit;font-size:13px;color:#1A2A22;outline:none;resize:none;height:80px;background:#F0FAF4;line-height:1.5}
.complaint-textarea:focus{border-color:#22A05A}
.complaint-footer{display:flex;gap:8px;justify-content:flex-end;margin-top:16px;padding-top:12px;border-top:1px solid #DDE8E2}
.complaint-btn-sec{padding:8px 18px;border-radius:10px;border:1.5px solid #DDE8E2;background:#fff;font-family:inherit;font-size:13px;font-weight:600;color:#3A5245;cursor:pointer}
.complaint-btn-sec:hover{background:#F0FAF4}
.complaint-btn-pri{padding:8px 20px;border-radius:10px;border:none;background:#1E8A4C;font-family:inherit;font-size:13px;font-weight:600;color:#fff;cursor:pointer}
.complaint-btn-pri:hover{background:#22A05A}
.complaint-btn-pri:disabled{background:#C0DECA;cursor:not-allowed}
.complaint-toast{position:fixed;bottom:24px;left:50%;transform:translateX(-50%) translateY(20px);background:#1A2A22;color:#fff;padding:11px 20px;border-radius:10px;font-size:13px;font-weight:600;z-index:1100;opacity:0;transition:all .25s;pointer-events:none;max-width:90vw;text-align:center}
.complaint-toast.show{opacity:1;transform:translateX(-50%) translateY(0)}
.complaint-toast.err{background:#E04040}
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
    t.textContent = msg;
    t.classList.toggle('err', !!isError);
    t.classList.add('show');
    setTimeout(() => t.classList.remove('show'), 3200);
  }

  /**
   * Открыть модалку жалобы.
   * @param {string} targetType
   * @param {number} targetID
   * @returns {Promise<boolean>}
   */
  window.lastopComplain = function(targetType, targetID){
    injectStyles();
    return new Promise(resolve => {
      let selectedReason = null;
      const bg = document.createElement('div');
      bg.className = 'complaint-bg';
      bg.innerHTML = `
        <div class="complaint-modal">
          <div class="complaint-head">
            <h3>Пожаловаться</h3>
            <button class="complaint-x" type="button">×</button>
          </div>
          <label class="complaint-label">Причина жалобы</label>
          <div class="complaint-reasons">
            ${REASONS.map(r => `
              <label class="complaint-reason" data-reason="${r.key}">
                <input type="radio" name="complaint-reason" value="${r.key}">
                <span>${r.label}</span>
              </label>
            `).join('')}
          </div>
          <label class="complaint-label">Комментарий (необязательно)</label>
          <textarea class="complaint-textarea" maxlength="1000" placeholder="Опишите подробнее, что не так…"></textarea>
          <div class="complaint-footer">
            <button class="complaint-btn-sec" type="button" data-act="cancel">Отмена</button>
            <button class="complaint-btn-pri" type="button" data-act="submit" disabled>Отправить</button>
          </div>
        </div>`;
      document.body.appendChild(bg);
      requestAnimationFrame(() => bg.classList.add('open'));

      const modal = bg.querySelector('.complaint-modal');
      const submitBtn = bg.querySelector('[data-act="submit"]');
      const textarea = bg.querySelector('.complaint-textarea');

      function close(result){
        bg.classList.remove('open');
        setTimeout(() => { bg.remove(); resolve(result); }, 200);
      }

      bg.addEventListener('click', e => { if (e.target === bg) close(false); });
      modal.querySelector('.complaint-x').addEventListener('click', () => close(false));
      modal.querySelector('[data-act="cancel"]').addEventListener('click', () => close(false));
      bg.querySelectorAll('.complaint-reason').forEach(el => {
        el.addEventListener('click', () => {
          bg.querySelectorAll('.complaint-reason').forEach(x => x.classList.remove('active'));
          el.classList.add('active');
          el.querySelector('input').checked = true;
          selectedReason = el.dataset.reason;
          submitBtn.disabled = false;
        });
      });

      submitBtn.addEventListener('click', async () => {
        if (!selectedReason) return;
        submitBtn.disabled = true;
        submitBtn.textContent = 'Отправка…';
        try {
          const token = localStorage.getItem('token') || '';
          const r = await fetch('/api/complaints', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', 'Authorization': 'Bearer ' + token },
            body: JSON.stringify({
              target_type: targetType,
              target_id: Number(targetID),
              reason: selectedReason,
              comment: textarea.value.trim(),
            }),
          });
          const d = await r.json().catch(() => ({}));
          if (r.ok) {
            showToast(d.message || 'Жалоба отправлена', false);
            close(true);
          } else {
            showToast(d.message || d.error || 'Не удалось отправить', true);
            submitBtn.disabled = false;
            submitBtn.textContent = 'Отправить';
          }
        } catch (e) {
          showToast('Ошибка соединения', true);
          submitBtn.disabled = false;
          submitBtn.textContent = 'Отправить';
        }
      });
    });
  };
})();
