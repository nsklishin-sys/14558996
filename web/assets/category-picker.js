// Кастомный dropdown с поиском и группировкой.
// Применяется к любому <select> с атрибутом data-category-picker.
// Скрывает оригинальный select, рисует поверх UI, синхронизирует change-события.
//
// Использование:
//   <select id="myCat" data-category-picker>
//     <optgroup label="Группа"><option>Опция 1</option></optgroup>
//   </select>
//   <script src="/assets/category-picker.js"></script>
//
// Если value меняется программно — вызвать select.dispatchEvent(new Event('change')) чтобы UI обновился.

(function() {
  'use strict';

  // CSS инжектируется один раз
  if (!document.getElementById('cp-styles')) {
    const css = `
      .cp-wrap{position:relative;font-family:inherit}
      .cp-trigger{
        display:flex;align-items:center;justify-content:space-between;gap:8px;
        width:100%;padding:10px 14px;
        background:#fff;border:1.5px solid var(--bdr,#E2E8F0);border-radius:var(--rm,12px);
        font-family:inherit;font-size:13px;color:var(--t,#0F172A);
        cursor:pointer;transition:border-color .12s;
        text-align:left;line-height:1.3
      }
      .cp-trigger:hover{border-color:var(--gb,#A0D9B8)}
      .cp-trigger.cp-open{border-color:var(--g,#1E8A4C)}
      .cp-trigger-text{flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
      .cp-trigger-text.cp-empty{color:var(--gmt,#94A3B8)}
      .cp-trigger-arrow{
        flex-shrink:0;color:var(--gmt,#94A3B8);
        transition:transform .15s
      }
      .cp-trigger.cp-open .cp-trigger-arrow{transform:rotate(180deg)}
      .cp-trigger-arrow svg{width:14px;height:14px;display:block;stroke:currentColor;fill:none;stroke-width:2;stroke-linecap:round;stroke-linejoin:round}

      .cp-popup{
        position:absolute;top:calc(100% + 4px);left:0;right:0;
        background:#fff;border:1px solid var(--bdr,#E2E8F0);border-radius:var(--rm,12px);
        box-shadow:0 8px 24px rgba(0,0,0,.12);
        z-index:1100;
        display:none;flex-direction:column;
        max-height:340px;overflow:hidden
      }
      .cp-popup.cp-open{display:flex}

      .cp-search{
        flex-shrink:0;padding:10px 12px;border-bottom:1px solid var(--bdr,#E2E8F0);
        position:relative
      }
      .cp-search input{
        width:100%;padding:8px 12px 8px 32px;
        background:var(--gp,#F8FAFC);border:1.5px solid transparent;border-radius:8px;
        font-family:inherit;font-size:12px;color:var(--t,#0F172A);outline:none;
        transition:border-color .12s
      }
      .cp-search input:focus{border-color:var(--gb,#A0D9B8);background:#fff}
      .cp-search-icon{
        position:absolute;left:22px;top:50%;transform:translateY(-50%);
        width:14px;height:14px;color:var(--gmt,#94A3B8);pointer-events:none
      }
      .cp-search-icon svg{width:14px;height:14px;display:block;stroke:currentColor;fill:none;stroke-width:2;stroke-linecap:round}

      .cp-list{
        flex:1;min-height:0;overflow-y:auto;padding:6px 0
      }
      .cp-list::-webkit-scrollbar{width:5px}
      .cp-list::-webkit-scrollbar-thumb{background:var(--gb,#A0D9B8);border-radius:99px}

      .cp-group-title{
        padding:8px 14px 4px;font-size:10px;font-weight:800;
        color:var(--gmt,#94A3B8);text-transform:uppercase;letter-spacing:.04em;
        user-select:none
      }
      .cp-option{
        display:flex;align-items:center;justify-content:space-between;gap:8px;
        padding:8px 14px;font-size:13px;color:var(--t,#0F172A);
        cursor:pointer;transition:background .1s
      }
      .cp-option:hover, .cp-option.cp-active{background:var(--gp,#F8FAFC)}
      .cp-option.cp-selected{color:var(--g,#1E8A4C);font-weight:600}
      .cp-option-check{
        flex-shrink:0;color:var(--g,#1E8A4C);opacity:0
      }
      .cp-option.cp-selected .cp-option-check{opacity:1}
      .cp-option-check svg{width:14px;height:14px;display:block;stroke:currentColor;fill:none;stroke-width:2.5;stroke-linecap:round;stroke-linejoin:round}

      .cp-empty-state{
        padding:24px 14px;text-align:center;
        font-size:12px;color:var(--gmt,#94A3B8)
      }
    `;
    const styleEl = document.createElement('style');
    styleEl.id = 'cp-styles';
    styleEl.textContent = css;
    document.head.appendChild(styleEl);
  }

  function buildPicker(select) {
    if (select.dataset.cpReady === '1') return; // уже обработан
    select.dataset.cpReady = '1';

    // Собираем структуру из optgroup/option
    const groups = [];
    let firstEmptyOpt = null;
    Array.from(select.children).forEach(child => {
      if (child.tagName === 'OPTGROUP') {
        const items = Array.from(child.querySelectorAll('option')).map(o => ({
          value: o.value,
          text: o.textContent
        }));
        groups.push({ label: child.label, items });
      } else if (child.tagName === 'OPTION') {
        if (child.value === '' && !firstEmptyOpt) {
          firstEmptyOpt = { value: '', text: child.textContent };
        } else {
          // Опция вне optgroup — кладём в "разное"
          if (!groups.length || groups[groups.length - 1].label !== '__loose__') {
            groups.push({ label: '__loose__', items: [] });
          }
          groups[groups.length - 1].items.push({ value: child.value, text: child.textContent });
        }
      }
    });

    // Fallback: если у select.value стоит категория, которой нет
    // в собранных группах (старая категория, которую удалили
    // или переименовали в актуальном списке) — добавляем
    // «фантомную» опцию с пометкой. Это позволяет сохранить
    // текущее значение в DOM и показать юзеру понятную метку
    // вместо пустоты.
    const currentValue = select.value;
    if (currentValue) {
      let foundInGroups = false;
      groups.forEach(g => g.items.forEach(it => {
        if (it.value === currentValue) foundInGroups = true;
      }));
      if (!foundInGroups) {
        const ghostText = currentValue + ' (удалена из списка)';
        // 1. Добавляем option в сам <select>, чтобы браузер не
        //    сбросил value на первое доступное значение
        const ghostOpt = document.createElement('option');
        ghostOpt.value = currentValue;
        ghostOpt.textContent = ghostText;
        select.appendChild(ghostOpt);
        select.value = currentValue;
        // 2. Добавляем псевдо-группу в picker, чтобы юзер видел
        //    опцию в UI
        groups.push({ label: 'Удалённые категории', items: [{ value: currentValue, text: ghostText }] });
      }
    }

    // Создаём wrapper и UI
    const wrap = document.createElement('div');
    wrap.className = 'cp-wrap';
    select.parentNode.insertBefore(wrap, select);
    wrap.appendChild(select);
    select.style.display = 'none';

    const trigger = document.createElement('button');
    trigger.type = 'button';
    trigger.className = 'cp-trigger';
    trigger.innerHTML = `
      <span class="cp-trigger-text"></span>
      <span class="cp-trigger-arrow"><svg viewBox="0 0 24 24"><polyline points="6 9 12 15 18 9"/></svg></span>
    `;
    wrap.appendChild(trigger);

    const popup = document.createElement('div');
    popup.className = 'cp-popup';
    popup.innerHTML = `
      <div class="cp-search">
        <span class="cp-search-icon"><svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><line x1="20" y1="20" x2="16.65" y2="16.65"/></svg></span>
        <input type="text" placeholder="Найти категорию…" autocomplete="off">
      </div>
      <div class="cp-list"></div>
    `;
    wrap.appendChild(popup);

    const triggerText = trigger.querySelector('.cp-trigger-text');
    const searchInput = popup.querySelector('input');
    const list = popup.querySelector('.cp-list');

    // Рендер списка опций (с фильтром)
    function renderList(query) {
      query = (query || '').trim().toLowerCase();
      list.innerHTML = '';
      let total = 0;

      groups.forEach(g => {
        const filtered = g.items.filter(it => !query || it.text.toLowerCase().includes(query));
        if (!filtered.length) return;

        if (g.label && g.label !== '__loose__') {
          const title = document.createElement('div');
          title.className = 'cp-group-title';
          title.textContent = g.label;
          list.appendChild(title);
        }

        filtered.forEach(it => {
          const opt = document.createElement('div');
          opt.className = 'cp-option';
          if (select.value === it.value) opt.classList.add('cp-selected');
          opt.dataset.value = it.value;
          opt.innerHTML = `
            <span class="cp-option-text"></span>
            <span class="cp-option-check"><svg viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg></span>
          `;
          opt.querySelector('.cp-option-text').textContent = it.text;
          opt.addEventListener('click', () => {
            selectOption(it.value);
            close();
          });
          list.appendChild(opt);
          total++;
        });
      });

      if (!total) {
        const empty = document.createElement('div');
        empty.className = 'cp-empty-state';
        empty.textContent = 'Ничего не найдено';
        list.appendChild(empty);
      }
    }

    function updateTrigger() {
      const v = select.value;
      let label = '';
      groups.forEach(g => g.items.forEach(it => {
        if (it.value === v) label = it.text;
      }));
      if (!label && firstEmptyOpt) {
        triggerText.textContent = firstEmptyOpt.text;
        triggerText.classList.add('cp-empty');
      } else if (label) {
        triggerText.textContent = label;
        triggerText.classList.remove('cp-empty');
      } else {
        triggerText.textContent = '— выберите —';
        triggerText.classList.add('cp-empty');
      }
    }

    function selectOption(value) {
      select.value = value;
      updateTrigger();
      // Триггерим change для другого JS
      select.dispatchEvent(new Event('change', { bubbles: true }));
    }

    function open() {
      trigger.classList.add('cp-open');
      popup.classList.add('cp-open');
      searchInput.value = '';
      renderList('');
      setTimeout(() => searchInput.focus(), 50);
    }
    function close() {
      trigger.classList.remove('cp-open');
      popup.classList.remove('cp-open');
    }
    function toggle() {
      if (popup.classList.contains('cp-open')) close();
      else open();
    }

    trigger.addEventListener('click', e => {
      e.stopPropagation();
      toggle();
    });
    searchInput.addEventListener('input', () => renderList(searchInput.value));
    searchInput.addEventListener('keydown', e => {
      if (e.key === 'Enter') {
        e.preventDefault();
        const first = list.querySelector('.cp-option');
        if (first) {
          selectOption(first.dataset.value);
          close();
        }
      } else if (e.key === 'Escape') {
        close();
      }
    });
    document.addEventListener('click', e => {
      if (!wrap.contains(e.target)) close();
    });

    // Если value <select>а меняется снаружи — слушаем
    select.addEventListener('change', () => updateTrigger());

    updateTrigger();
  }

  function init() {
    document.querySelectorAll('select[data-category-picker]').forEach(buildPicker);
  }

  // Экспортируем для повторной инициализации после динамической вставки
  window.initCategoryPickers = init;

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
