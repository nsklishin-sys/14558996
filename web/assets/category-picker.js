// Кастомный dropdown с поиском и группировкой.
// Применяется к любому <select> с атрибутом data-category-picker.
// Скрывает оригинальный select, рисует поверх UI, синхронизирует change-события.
//
// Использование (2 уровня — как было):
//   <select data-category-picker>
//     <optgroup label="Группа"><option value="k">Опция</option></optgroup>
//   </select>
//
// Использование (3 уровня — дерево с раскрытием):
//   подкатегории передаются как <option> с атрибутом data-parent="ключ_категории".
//   Компонент сам построит группа → категория → подкатегория с раскрытием.
//   Если ни у одной опции нет data-parent — работает в обычном плоском режиме (полная совместимость).
//
// Если value меняется программно — вызвать select.dispatchEvent(new Event('change')) чтобы UI обновился.

(function() {
  'use strict';

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

      /* ── режим дерева (3 уровня) ── */
      .cp-node{
        display:flex;align-items:center;gap:6px;
        padding:8px 14px;font-size:13px;color:var(--t,#0F172A);
        cursor:pointer;transition:background .1s;user-select:none
      }
      .cp-node:hover{background:var(--gp,#F8FAFC)}
      .cp-node.cp-selected{color:var(--g,#1E8A4C);font-weight:600}
      .cp-node-caret{
        flex-shrink:0;width:18px;height:18px;display:grid;place-items:center;
        color:var(--gmt,#94A3B8);font-size:10px;transition:transform .12s
      }
      .cp-node-caret.cp-open{transform:rotate(90deg);color:var(--g,#1E8A4C)}
      .cp-node-caret-empty{flex-shrink:0;width:18px}
      .cp-node-label{flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
      .cp-node-check{flex-shrink:0;color:var(--g,#1E8A4C);opacity:0}
      .cp-node.cp-selected .cp-node-check{opacity:1}
      .cp-node-check svg{width:14px;height:14px;display:block;stroke:currentColor;fill:none;stroke-width:2.5;stroke-linecap:round;stroke-linejoin:round}
      .cp-lvl-0{font-weight:700}
      .cp-lvl-1{padding-left:32px}
      .cp-lvl-2{padding-left:52px;font-size:12.5px;color:var(--tm,#3A5245)}

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
    const existingWrap = select.closest('.cp-wrap');
    if (existingWrap) {
      existingWrap.parentNode.insertBefore(select, existingWrap);
      existingWrap.remove();
    }
    select.dataset.cpReady = '1';

    // Собираем структуру из optgroup/option.
    // groups: [{ label, items:[{value,text, children:[{value,text}]}] }]
    // children заполняются из <option data-parent="ключ_категории">.
    const groups = [];
    let firstEmptyOpt = null;
    let hasTree = false; // включаем режим дерева только если есть data-parent

    Array.from(select.children).forEach(child => {
      if (child.tagName === 'OPTGROUP') {
        const opts = Array.from(child.querySelectorAll('option'));
        const items = [];
        const byKey = {};
        // сначала родительские категории (без data-parent), затем подкатегории
        opts.forEach(o => {
          const parent = o.getAttribute('data-parent');
          if (!parent) {
            const node = { value: o.value, text: o.textContent, children: [] };
            items.push(node);
            byKey[o.value] = node;
          }
        });
        opts.forEach(o => {
          const parent = o.getAttribute('data-parent');
          if (parent) {
            hasTree = true;
            const node = { value: o.value, text: o.textContent };
            if (byKey[parent]) byKey[parent].children.push(node);
            else items.push({ value: o.value, text: o.textContent, children: [] }); // сирота — как категория
          }
        });
        groups.push({ label: child.label, items });
      } else if (child.tagName === 'OPTION') {
        if (child.value === '' && !firstEmptyOpt) {
          firstEmptyOpt = { value: '', text: child.textContent };
        } else {
          if (!groups.length || groups[groups.length - 1].label !== '__loose__') {
            groups.push({ label: '__loose__', items: [] });
          }
          groups[groups.length - 1].items.push({ value: child.value, text: child.textContent, children: [] });
        }
      }
    });

    // Fallback для значения, которого нет в списке (удалённая категория)
    const currentValue = select.value;
    if (currentValue) {
      let found = false;
      groups.forEach(g => g.items.forEach(it => {
        if (it.value === currentValue) found = true;
        (it.children || []).forEach(ch => { if (ch.value === currentValue) found = true; });
      }));
      if (!found) {
        const ghostText = currentValue + ' (удалена из списка)';
        const ghostOpt = document.createElement('option');
        ghostOpt.value = currentValue;
        ghostOpt.textContent = ghostText;
        select.appendChild(ghostOpt);
        select.value = currentValue;
        groups.push({ label: 'Удалённые категории', items: [{ value: currentValue, text: ghostText, children: [] }] });
      }
    }

    // UI
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
        <input type="text" placeholder="Поиск…" autocomplete="off">
      </div>
      <div class="cp-list"></div>
    `;
    wrap.appendChild(popup);

    const triggerText = trigger.querySelector('.cp-trigger-text');
    const searchInput = popup.querySelector('input');
    const list = popup.querySelector('.cp-list');

    // состояние раскрытия в режиме дерева
    const expanded = {}; // ключ группы/категории -> bool

    // ── РЕНДЕР: плоский режим (как было, для 2-уровневых потребителей) ──
    function renderFlat(query) {
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
          opt.innerHTML = `<span class="cp-option-text"></span><span class="cp-option-check"><svg viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg></span>`;
          opt.querySelector('.cp-option-text').textContent = it.text;
          opt.addEventListener('click', () => { selectOption(it.value); close(); });
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

    // ── РЕНДЕР: дерево (3 уровня, с раскрытием) ──
    function nodeRow(level, label, value, hasChildren, isExpanded, onCaret, onSelect) {
      const row = document.createElement('div');
      row.className = 'cp-node cp-lvl-' + level;
      if (value !== null && select.value === value) row.classList.add('cp-selected');
      const caret = hasChildren
        ? `<span class="cp-node-caret${isExpanded ? ' cp-open' : ''}">▶</span>`
        : `<span class="cp-node-caret-empty"></span>`;
      const check = value !== null
        ? `<span class="cp-node-check"><svg viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg></span>`
        : '';
      row.innerHTML = caret + `<span class="cp-node-label"></span>` + check;
      row.querySelector('.cp-node-label').textContent = label;
      const caretEl = row.querySelector('.cp-node-caret');
      if (caretEl && onCaret) {
        caretEl.addEventListener('click', e => { e.stopPropagation(); onCaret(); });
      }
      row.addEventListener('click', () => {
        if (onSelect) onSelect();
        else if (onCaret) onCaret(); // у группы нет выбора — клик раскрывает
      });
      return row;
    }

    function renderTree(query) {
      query = (query || '').trim().toLowerCase();
      list.innerHTML = '';
      let total = 0;
      const searching = !!query;

      groups.forEach(g => {
        // фильтрация: показываем категорию/подкатегорию, если совпадает её текст
        // или текст подкатегории. При поиске авто-раскрываем.
        const matchedItems = g.items.filter(cat => {
          if (!searching) return true;
          if (cat.text.toLowerCase().includes(query)) return true;
          return (cat.children || []).some(ch => ch.text.toLowerCase().includes(query));
        });
        if (!matchedItems.length) return;

        const gKey = 'g:' + g.label;
        const gExpanded = searching ? true : (expanded[gKey] !== false); // группы по умолчанию раскрыты
        list.appendChild(nodeRow(0, g.label === '__loose__' ? 'Категории' : g.label, null, true, gExpanded,
          () => { expanded[gKey] = !gExpanded; renderTree(searchInput.value); }, null));
        total++;
        if (!gExpanded) return;

        matchedItems.forEach(cat => {
          const hasCh = cat.children && cat.children.length;
          const cKey = 'c:' + cat.value;
          const cExpanded = searching ? true : !!expanded[cKey]; // категории по умолчанию свёрнуты
          list.appendChild(nodeRow(1, cat.text, cat.value, hasCh, cExpanded,
            hasCh ? () => { expanded[cKey] = !cExpanded; renderTree(searchInput.value); } : null,
            () => { selectOption(cat.value); close(); }));
          total++;
          if (hasCh && cExpanded) {
            const subs = cat.children.filter(ch => !searching || ch.text.toLowerCase().includes(query) || cat.text.toLowerCase().includes(query));
            subs.forEach(sub => {
              list.appendChild(nodeRow(2, sub.text, sub.value, false, false, null,
                () => { selectOption(sub.value); close(); }));
              total++;
            });
          }
        });
      });

      if (!total) {
        const empty = document.createElement('div');
        empty.className = 'cp-empty-state';
        empty.textContent = 'Ничего не найдено';
        list.appendChild(empty);
      }
    }

    function renderList(query) {
      if (hasTree) renderTree(query);
      else renderFlat(query);
    }

    function updateTrigger() {
      const v = select.value;
      let label = '';
      groups.forEach(g => g.items.forEach(it => {
        if (it.value === v) label = it.text;
        (it.children || []).forEach(ch => { if (ch.value === v) label = ch.text; });
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

    trigger.addEventListener('click', e => { e.stopPropagation(); toggle(); });
    searchInput.addEventListener('input', () => renderList(searchInput.value));
    searchInput.addEventListener('keydown', e => {
      if (e.key === 'Enter') {
        e.preventDefault();
        const firstSel = list.querySelector('.cp-option, .cp-node');
        // в режиме дерева Enter выбирает первую выбираемую (с data-value) — пропускаем группы
        if (hasTree) {
          // ничего не делаем по Enter в дереве, чтобы не выбрать группу
        } else if (firstSel) {
          selectOption(firstSel.dataset.value);
          close();
        }
      } else if (e.key === 'Escape') {
        close();
      }
    });
    document.addEventListener('click', e => { if (!wrap.contains(e.target)) close(); });

    select.addEventListener('change', () => updateTrigger());

    updateTrigger();
  }

  function init() {
    document.querySelectorAll('select[data-category-picker]').forEach(buildPicker);
  }

  window.initCategoryPickers = init;

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
