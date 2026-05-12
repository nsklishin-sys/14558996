(() => {
  if(window.__calendarModalInit)return;
  window.__calendarModalInit=true;

  /* ════════════ CSS МОДАЛКИ ════════════ */
  const CSS = `
.lcm-modal-bg{position:fixed;inset:0;background:rgba(15,25,18,.45);backdrop-filter:blur(4px);z-index:200;display:flex;align-items:center;justify-content:center;padding:20px;opacity:0;pointer-events:none;transition:opacity .2s}
.lcm-modal-bg.open{opacity:1;pointer-events:all}
.lcm-modal{background:var(--w);border-radius:var(--rxl);border:1px solid var(--bdr);box-shadow:0 24px 60px rgba(30,138,76,.18);padding:24px;width:100%;max-width:560px;max-height:90vh;overflow-y:auto;transform:translateY(14px);transition:transform .22s}
.lcm-modal-bg.open .lcm-modal{transform:translateY(0)}
.lcm-modal::-webkit-scrollbar{width:4px}
.lcm-modal::-webkit-scrollbar-thumb{background:var(--gb);border-radius:99px}
.lcm-modal-head{display:flex;align-items:center;justify-content:space-between;margin-bottom:18px}
.lcm-modal-head h3{font-size:17px;font-weight:900;color:var(--t)}
.lcm-modal-x{width:30px;height:30px;border-radius:8px;background:var(--gp);border:none;cursor:pointer;display:grid;place-items:center;color:var(--gmt);transition:background .15s;flex-shrink:0}
.lcm-modal-x:hover{background:var(--gl);color:var(--g)}
.lcm-modal-x svg{width:14px;height:14px;fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round}
.lcm-form-row{margin-bottom:13px}
.lcm-form-row.req .lcm-form-label::after{content:' *';color:var(--red)}
.lcm-form-label{font-size:11px;font-weight:800;color:var(--gmt);letter-spacing:.05em;text-transform:uppercase;display:block;margin-bottom:6px}
.lcm-form-input{width:100%;border:1.5px solid var(--bdr);border-radius:12px;padding:9px 12px;font-family:inherit;font-size:13px;color:var(--t);outline:none;transition:border-color .2s;background:var(--gp)}
.lcm-form-input:focus{border-color:var(--gm);background:var(--w)}
.lcm-form-input::placeholder{color:var(--gmt)}
textarea.lcm-form-input{resize:vertical;min-height:70px;line-height:1.5}
.lcm-form-grid2{display:grid;grid-template-columns:1fr 1fr;gap:10px}
.lcm-kind-row,.lcm-color-row{display:flex;gap:6px;flex-wrap:wrap}
.lcm-kind-chip{padding:6px 12px;border-radius:99px;border:1.5px solid var(--bdr);background:var(--gp);color:var(--tm);font-family:inherit;font-size:12px;font-weight:600;cursor:pointer;display:inline-flex;align-items:center;gap:5px;transition:all .15s}
.lcm-kind-chip:hover{border-color:var(--gb);background:var(--gl)}
.lcm-kind-chip.active{background:var(--g);color:#fff;border-color:var(--g)}
.lcm-kind-chip svg{width:11px;height:11px;stroke:currentColor;fill:none;stroke-width:1.8;stroke-linecap:round}
.lcm-color-chip{width:30px;height:30px;border-radius:8px;cursor:pointer;border:3px solid transparent;transition:transform .12s;position:relative}
.lcm-color-chip:hover{transform:scale(1.08)}
.lcm-color-chip.active{border-color:var(--w);box-shadow:0 0 0 2px var(--t)}
.lcm-allday-row{display:flex;align-items:center;gap:8px;margin-top:10px;cursor:pointer;user-select:none}
.lcm-allday-cb{width:18px;height:18px;border:1.8px solid var(--bdr);border-radius:5px;display:grid;place-items:center;background:var(--w);transition:all .12s}
.lcm-allday-cb.on{background:var(--g);border-color:var(--g)}
.lcm-allday-cb.on svg{display:block}
.lcm-allday-cb svg{display:none;width:11px;height:11px;stroke:#fff;fill:none;stroke-width:3;stroke-linecap:round}
.lcm-allday-row label{font-size:13px;color:var(--t);font-weight:600;cursor:pointer}
.lcm-modal-footer{display:flex;gap:8px;justify-content:flex-end;margin-top:18px;padding-top:14px;border-top:1px solid var(--bdr)}
.lcm-btn-sec{padding:9px 18px;border-radius:99px;border:1.5px solid var(--bdr);background:var(--w);font-family:inherit;font-size:13px;font-weight:700;color:var(--tm);cursor:pointer;transition:background .15s}
.lcm-btn-sec:hover{background:var(--gp)}
.lcm-btn-pri{padding:9px 24px;border-radius:99px;border:none;background:var(--g);font-family:inherit;font-size:13px;font-weight:700;color:#fff;cursor:pointer;transition:background .15s}
.lcm-btn-pri:hover{background:var(--gm)}
.lcm-btn-pri:disabled{background:var(--gb);cursor:not-allowed}
.lcm-btn-del{padding:9px 18px;border-radius:99px;border:1.5px solid #F5C2C2;background:#FFF;color:var(--red);font-family:inherit;font-size:13px;font-weight:700;cursor:pointer;transition:all .15s;margin-right:auto}
.lcm-btn-del:hover{background:#FEEAEA}
`;

  /* ════════════ HTML МОДАЛКИ ════════════ */
  const HTML = `
<div class="lcm-modal-bg" id="lcmModalBg">
  <div class="lcm-modal">
    <div class="lcm-modal-head">
      <h3 id="lcmModalTitle">Создать событие</h3>
      <button class="lcm-modal-x" type="button" id="lcmCloseX"><svg viewBox="0 0 24 24"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
    </div>

    <div class="lcm-form-row req">
      <label class="lcm-form-label">Название</label>
      <input class="lcm-form-input" id="lcmTitle" placeholder="Например: Звонок с командой" maxlength="200">
    </div>

    <div class="lcm-form-row">
      <label class="lcm-form-label">Тип</label>
      <div class="lcm-kind-row" id="lcmKindRow">
        <button type="button" class="lcm-kind-chip active" data-kind="personal"><svg viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>Личное</button>
        <button type="button" class="lcm-kind-chip" data-kind="meeting"><svg viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/></svg>Встреча</button>
        <button type="button" class="lcm-kind-chip" data-kind="reminder"><svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>Напоминание</button>
        <button type="button" class="lcm-kind-chip" data-kind="note"><svg viewBox="0 0 24 24"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>Заметка</button>
        <button type="button" class="lcm-kind-chip" data-kind="deadline"><svg viewBox="0 0 24 24"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>Дедлайн</button>
      </div>
    </div>

    <div class="lcm-form-grid2">
      <div class="lcm-form-row req">
        <label class="lcm-form-label">Начало</label>
        <input class="lcm-form-input" type="datetime-local" id="lcmStarts">
      </div>
      <div class="lcm-form-row">
        <label class="lcm-form-label">Окончание</label>
        <input class="lcm-form-input" type="datetime-local" id="lcmEnds">
      </div>
    </div>

    <div class="lcm-allday-row" id="lcmAllDayRow">
      <div class="lcm-allday-cb" id="lcmAllDayCb"><svg viewBox="0 0 12 12"><polyline points="2 6 5 9 10 3"/></svg></div>
      <label>Весь день</label>
    </div>

    <div class="lcm-form-row" style="margin-top:13px">
      <label class="lcm-form-label">Цвет</label>
      <div class="lcm-color-row" id="lcmColorRow">
        <div class="lcm-color-chip active" data-color="#1E8A4C" style="background:#1E8A4C"></div>
        <div class="lcm-color-chip" data-color="#185FA5" style="background:#185FA5"></div>
        <div class="lcm-color-chip" data-color="#B07A00" style="background:#B07A00"></div>
        <div class="lcm-color-chip" data-color="#C04030" style="background:#C04030"></div>
        <div class="lcm-color-chip" data-color="#9060C0" style="background:#9060C0"></div>
        <div class="lcm-color-chip" data-color="#208090" style="background:#208090"></div>
        <div class="lcm-color-chip" data-color="#5A5A5A" style="background:#5A5A5A"></div>
      </div>
    </div>

    <div class="lcm-form-row">
      <label class="lcm-form-label">Место (опционально)</label>
      <input class="lcm-form-input" id="lcmLocation" placeholder="Адрес или ссылка на встречу" maxlength="500">
    </div>

    <div class="lcm-form-row">
      <label class="lcm-form-label">Описание</label>
      <textarea class="lcm-form-input" id="lcmDescription" placeholder="Дополнительные детали…" maxlength="5000"></textarea>
    </div>

    <div class="lcm-form-row">
      <label class="lcm-form-label">Напомнить за</label>
      <select class="lcm-form-input" id="lcmReminder">
        <option value="">Не напоминать</option>
        <option value="5">5 минут</option>
        <option value="15">15 минут</option>
        <option value="30">30 минут</option>
        <option value="60">1 час</option>
        <option value="1440">1 день</option>
      </select>
    </div>

    <div class="lcm-modal-footer">
      <button class="lcm-btn-del" id="lcmDeleteBtn" type="button" style="display:none">Удалить</button>
      <button class="lcm-btn-sec" type="button" id="lcmCancelBtn">Отмена</button>
      <button class="lcm-btn-pri" id="lcmSaveBtn" type="button">Создать</button>
    </div>
  </div>
</div>
`;

  let editingId = null;
  let modalKind = 'personal';
  let modalColor = '#1E8A4C';
  let modalAllDay = false;
  let currentOnSaved = null;

  let __injected = false;
  function ensureInjected(){
    if(__injected)return;
    __injected = true;
    const style = document.createElement('style');
    style.textContent = CSS;
    document.head.appendChild(style);
    const wrap = document.createElement('div');
    wrap.innerHTML = HTML;
    document.body.appendChild(wrap.firstElementChild);
    document.getElementById('lcmCloseX').addEventListener('click', closeModal);
    document.getElementById('lcmCancelBtn').addEventListener('click', closeModal);
    document.getElementById('lcmSaveBtn').addEventListener('click', saveEvent);
    document.getElementById('lcmDeleteBtn').addEventListener('click', deleteEvent);
    document.getElementById('lcmAllDayRow').addEventListener('click', toggleAllDay);
    document.getElementById('lcmKindRow').addEventListener('click', e => {
      const c = e.target.closest('.lcm-kind-chip'); if(!c)return;
      modalKind = c.dataset.kind;
      document.querySelectorAll('#lcmKindRow .lcm-kind-chip').forEach(x => x.classList.toggle('active', x === c));
    });
    document.getElementById('lcmColorRow').addEventListener('click', e => {
      const c = e.target.closest('.lcm-color-chip'); if(!c)return;
      modalColor = c.dataset.color;
      document.querySelectorAll('#lcmColorRow .lcm-color-chip').forEach(x => x.classList.toggle('active', x === c));
    });
    document.getElementById('lcmModalBg').addEventListener('click', e => {
      if(e.target.id === 'lcmModalBg') closeModal();
    });
    document.addEventListener('keydown', e => {
      if(e.key === 'Escape' && document.getElementById('lcmModalBg').classList.contains('open')){
        closeModal();
      }
    });
  }

  function toLocalInputDT(d){
    if(!(d instanceof Date) || isNaN(d))return '';
    const pad = n => String(n).padStart(2,'0');
    return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
  }
  function toggleAllDay(){
    modalAllDay = !modalAllDay;
    document.getElementById('lcmAllDayCb').classList.toggle('on', modalAllDay);
  }
  function getToken(){
    return (typeof tk === 'function' ? tk() : null) || localStorage.getItem('token') || '';
  }

  function openModal(opts){
    ensureInjected();
    opts = opts || {};
    const ev = opts.event || null;
    const defaultDate = opts.defaultDate || null;
    currentOnSaved = opts.onSaved || null;

    editingId = null;
    modalKind = 'personal';
    modalColor = '#1E8A4C';
    modalAllDay = false;

    document.getElementById('lcmModalTitle').textContent = ev ? 'Редактирование события' : 'Создать событие';
    document.getElementById('lcmSaveBtn').textContent = ev ? 'Сохранить' : 'Создать';
    document.getElementById('lcmDeleteBtn').style.display = ev ? '' : 'none';

    document.getElementById('lcmTitle').value = ev?.title || '';
    document.getElementById('lcmDescription').value = ev?.description || '';
    document.getElementById('lcmLocation').value = ev?.location || '';
    document.getElementById('lcmReminder').value = '';

    if(ev){
      editingId = ev.id;
      modalKind = ev.kind || 'personal';
      modalColor = ev.color || '#1E8A4C';
      modalAllDay = !!ev.is_all_day;
      const sa = new Date(ev.starts_at);
      const ea = new Date(ev.ends_at);
      document.getElementById('lcmStarts').value = toLocalInputDT(sa);
      document.getElementById('lcmEnds').value = toLocalInputDT(ea);
    } else {
      const base = defaultDate ? new Date(defaultDate) : new Date();
      const sa = new Date(base); sa.setHours(10,0,0,0);
      const ea = new Date(base); ea.setHours(11,0,0,0);
      document.getElementById('lcmStarts').value = toLocalInputDT(sa);
      document.getElementById('lcmEnds').value = toLocalInputDT(ea);
    }

    document.querySelectorAll('#lcmKindRow .lcm-kind-chip').forEach(c => c.classList.toggle('active', c.dataset.kind === modalKind));
    document.querySelectorAll('#lcmColorRow .lcm-color-chip').forEach(c => c.classList.toggle('active', c.dataset.color === modalColor));
    document.getElementById('lcmAllDayCb').classList.toggle('on', modalAllDay);

    if(window.lastopClearAllErrors)window.lastopClearAllErrors(document.getElementById('lcmModalBg'));
    document.getElementById('lcmModalBg').classList.add('open');
    setTimeout(() => document.getElementById('lcmTitle').focus(), 120);
  }
  function closeModal(){
    const bg = document.getElementById('lcmModalBg');
    if(bg)bg.classList.remove('open');
    editingId = null;
    currentOnSaved = null;
  }

  async function saveEvent(){
    if(window.lastopClearAllErrors)window.lastopClearAllErrors(document.getElementById('lcmModalBg'));
    const titleEl = document.getElementById('lcmTitle');
    const startsEl = document.getElementById('lcmStarts');
    const endsEl = document.getElementById('lcmEnds');

    if(window.lastopValidate){
      const ok = window.lastopValidate([
        {el: titleEl, required: true, message: 'Введите название события'},
        {el: startsEl, required: true, message: 'Укажите дату и время начала'},
      ]);
      if(!ok)return;
    } else {
      if(!titleEl.value.trim()){alert('Введите название');return;}
      if(!startsEl.value){alert('Укажите дату начала');return;}
    }

    const body = {
      title: titleEl.value.trim(),
      description: document.getElementById('lcmDescription').value.trim(),
      kind: modalKind,
      starts_at: startsEl.value,
      ends_at: endsEl.value || startsEl.value,
      is_all_day: modalAllDay,
      color: modalColor,
      location: document.getElementById('lcmLocation').value.trim(),
    };
    const remVal = document.getElementById('lcmReminder').value;
    if(remVal !== '') body.reminder_minutes = parseInt(remVal, 10);
    else if(editingId) body.reminder_minutes = -1;
    if(body.ends_at && body.ends_at < body.starts_at) body.ends_at = body.starts_at;

    const btn = document.getElementById('lcmSaveBtn');
    btn.disabled = true;
    btn.textContent = editingId ? 'Сохранение…' : 'Создание…';
    try{
      const url = editingId ? `/api/calendar/events/${editingId}` : `/api/calendar/events`;
      const method = editingId ? 'PATCH' : 'POST';
      const r = await fetch(url, {
        method,
        headers: {'Content-Type': 'application/json', Authorization: 'Bearer ' + getToken()},
        body: JSON.stringify(body)
      });
      if(!r.ok){
        const d = await r.json().catch(() => ({}));
        if(r.status === 429 && window.lastopAlert){
          await window.lastopAlert({title: 'Слишком много попыток', message: d.error || 'Подождите немного и повторите', type: 'warn'});
        } else if(window.lastopToast){
          window.lastopToast(d.error || 'Не удалось сохранить', 'error');
        } else {
          alert(d.error || 'Ошибка');
        }
        btn.disabled = false;
        btn.textContent = editingId ? 'Сохранить' : 'Создать';
        return;
      }
      if(window.lastopToast)window.lastopToast(editingId ? 'Событие обновлено' : 'Событие создано', 'success');
      const cb = currentOnSaved;
      closeModal();
      if(typeof cb === 'function'){try{cb();}catch{}}
    }catch{
      if(window.lastopToast)window.lastopToast('Ошибка сети', 'error');
    }
    btn.disabled = false;
    btn.textContent = editingId ? 'Сохранить' : 'Создать';
  }

  async function deleteEvent(){
    if(!editingId)return;
    const confirmed = window.lastopConfirm
      ? await window.lastopConfirm({title: 'Удалить событие?', message: 'Это действие нельзя отменить', danger: true})
      : confirm('Удалить событие?');
    if(!confirmed)return;
    try{
      const r = await fetch(`/api/calendar/events/${editingId}`, {
        method: 'DELETE',
        headers: {Authorization: 'Bearer ' + getToken()}
      });
      if(r.ok){
        if(window.lastopToast)window.lastopToast('Удалено', 'success');
        const cb = currentOnSaved;
        closeModal();
        if(typeof cb === 'function'){try{cb();}catch{}}
      } else {
        if(window.lastopToast)window.lastopToast('Не удалось удалить', 'error');
      }
    }catch{
      if(window.lastopToast)window.lastopToast('Ошибка сети', 'error');
    }
  }

  window.openCalendarEventModal = openModal;
  window.closeCalendarEventModal = closeModal;
})();
