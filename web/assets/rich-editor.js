/* rich-editor.js — переиспользуемый rich-text редактор (расширенный набор).
 *
 * Использование:
 *   const ed = LastopRichEditor.create(containerEl, { value: '<p>...</p>', placeholder: 'Текст…' });
 *   ed.getHTML();          // вернуть текущий HTML
 *   ed.setHTML(html);      // установить содержимое
 *   ed.isEmpty();          // true если пусто
 *   ed.destroy();          // снять обработчики
 *
 * Тулбар: жирный, курсив, подчёркнутый, заголовок (H3), маркированный/нумерованный
 * список, цитата, ссылка, выравнивание (лево/центр/право), очистка форматирования.
 *
 * ⚠️ Безопасность: HTML с редактора ОБЯЗАТЕЛЬНО санитизируется на бэке
 * (sanitizeRichHTML). Клиентская очистка тут — только для UX, не для защиты.
 *
 * Хранит/выдаёт HTML с allowlist-тегами: p, br, b, strong, i, em, u, s,
 * ul, ol, li, h3, blockquote, a[href], span[style=text-align] (через div).
 */
(function () {
  'use strict';

  var STYLE_ID = 'lastop-rich-editor-style';
  var CSS = `
.lre-wrap{border:1.5px solid var(--bdr,#DDE8E2);border-radius:var(--rm,12px);background:var(--w,#fff);overflow:hidden;font-family:'Manrope',sans-serif}
.lre-wrap.focused{border-color:var(--gm,#22A05A)}
.lre-toolbar{display:flex;flex-wrap:wrap;gap:2px;padding:6px 8px;border-bottom:1px solid var(--bdr,#DDE8E2);background:var(--gp,#F0FAF4)}
.lre-btn{width:28px;height:28px;border-radius:7px;border:none;background:transparent;color:var(--tm,#3A5245);cursor:pointer;display:grid;place-items:center;transition:background .12s,color .12s;flex-shrink:0;font-family:inherit}
.lre-btn:hover{background:var(--gl,#E8F5EE);color:var(--g,#1E8A4C)}
.lre-btn.active{background:var(--g,#1E8A4C);color:#fff}
.lre-btn svg{width:15px;height:15px;fill:none;stroke:currentColor;stroke-width:1.9;stroke-linecap:round;stroke-linejoin:round;pointer-events:none}
.lre-btn b{font-size:14px;font-weight:800;pointer-events:none}
.lre-btn i{font-size:14px;font-style:italic;font-family:Georgia,serif;pointer-events:none}
.lre-btn u{font-size:13px;text-decoration:underline;pointer-events:none}
.lre-btn s{font-size:13px;text-decoration:line-through;pointer-events:none}
.lre-sep{width:1px;background:var(--bdr,#DDE8E2);margin:3px 4px;align-self:stretch}
.lre-area{min-height:120px;max-height:420px;overflow-y:auto;padding:12px 14px;font-size:13.5px;line-height:1.65;color:var(--t,#1A2A22);outline:none}
.lre-area:empty::before{content:attr(data-placeholder);color:var(--gmt,#5A8A6A);pointer-events:none}
.lre-area p{margin:0 0 10px}
.lre-area p:last-child{margin-bottom:0}
.lre-area h3{font-size:15px;font-weight:800;color:var(--t,#1A2A22);margin:14px 0 8px}
.lre-area h3:first-child{margin-top:0}
.lre-area ul,.lre-area ol{margin:0 0 10px;padding-left:22px}
.lre-area li{margin-bottom:4px}
.lre-area blockquote{margin:0 0 10px;padding:6px 0 6px 14px;border-left:3px solid var(--gb,#C0DECA);color:var(--tm,#3A5245);font-style:italic}
.lre-area a{color:var(--g,#1E8A4C);text-decoration:underline}
.lre-area:focus{outline:none}
`;

  function injectStyle() {
    if (document.getElementById(STYLE_ID)) return;
    var st = document.createElement('style');
    st.id = STYLE_ID;
    st.textContent = CSS;
    document.head.appendChild(st);
  }

  // Кнопки тулбара. cmd — execCommand, value — доп.аргумент, icon — содержимое.
  var BUTTONS = [
    { cmd: 'bold', title: 'Жирный (Ctrl+B)', html: '<b>B</b>' },
    { cmd: 'italic', title: 'Курсив (Ctrl+I)', html: '<i>I</i>' },
    { cmd: 'underline', title: 'Подчёркнутый (Ctrl+U)', html: '<u>U</u>' },
    { cmd: 'strikeThrough', title: 'Зачёркнутый', html: '<s>S</s>' },
    { sep: true },
    { cmd: 'formatBlock', value: 'h3', title: 'Заголовок', html: '<svg viewBox="0 0 24 24"><path d="M6 4v16M18 4v16M6 12h12"/></svg>' },
    { cmd: 'insertUnorderedList', title: 'Маркированный список', html: '<svg viewBox="0 0 24 24"><line x1="9" y1="6" x2="20" y2="6"/><line x1="9" y1="12" x2="20" y2="12"/><line x1="9" y1="18" x2="20" y2="18"/><circle cx="4" cy="6" r="1.4" fill="currentColor" stroke="none"/><circle cx="4" cy="12" r="1.4" fill="currentColor" stroke="none"/><circle cx="4" cy="18" r="1.4" fill="currentColor" stroke="none"/></svg>' },
    { cmd: 'insertOrderedList', title: 'Нумерованный список', html: '<svg viewBox="0 0 24 24"><line x1="10" y1="6" x2="20" y2="6"/><line x1="10" y1="12" x2="20" y2="12"/><line x1="10" y1="18" x2="20" y2="18"/><path d="M4 5h1v3M3.5 11.5h1.5l-1.5 2h1.5M3.5 16.5h1.3v1.2h-1.3v1.3h1.3" stroke-width="1.2"/></svg>' },
    { cmd: 'formatBlock', value: 'blockquote', title: 'Цитата', html: '<svg viewBox="0 0 24 24"><path d="M7 7H4v6h3l-1 4M17 7h-3v6h3l-1 4"/></svg>' },
    { sep: true },
    { action: 'link', title: 'Ссылка', html: '<svg viewBox="0 0 24 24"><path d="M10 13a5 5 0 0 0 7 0l3-3a5 5 0 0 0-7-7l-1 1"/><path d="M14 11a5 5 0 0 0-7 0l-3 3a5 5 0 0 0 7 7l1-1"/></svg>' },
    { sep: true },
    { cmd: 'justifyLeft', title: 'По левому краю', html: '<svg viewBox="0 0 24 24"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="14" y2="12"/><line x1="3" y1="18" x2="18" y2="18"/></svg>' },
    { cmd: 'justifyCenter', title: 'По центру', html: '<svg viewBox="0 0 24 24"><line x1="3" y1="6" x2="21" y2="6"/><line x1="6" y1="12" x2="18" y2="12"/><line x1="4" y1="18" x2="20" y2="18"/></svg>' },
    { cmd: 'justifyRight', title: 'По правому краю', html: '<svg viewBox="0 0 24 24"><line x1="3" y1="6" x2="21" y2="6"/><line x1="10" y1="12" x2="21" y2="12"/><line x1="6" y1="18" x2="21" y2="18"/></svg>' },
    { sep: true },
    { cmd: 'removeFormat', title: 'Очистить форматирование', html: '<svg viewBox="0 0 24 24"><path d="M4 7V4h16v3M9 20h6M12 4v16"/><line x1="3" y1="3" x2="21" y2="21"/></svg>' }
  ];

  function create(container, opts) {
    opts = opts || {};
    injectStyle();

    var wrap = document.createElement('div');
    wrap.className = 'lre-wrap';

    var toolbar = document.createElement('div');
    toolbar.className = 'lre-toolbar';

    var area = document.createElement('div');
    area.className = 'lre-area';
    area.contentEditable = 'true';
    area.setAttribute('data-placeholder', opts.placeholder || 'Введите текст…');
    if (opts.value) area.innerHTML = opts.value;

    // Сборка тулбара
    BUTTONS.forEach(function (b) {
      if (b.sep) {
        var sep = document.createElement('div');
        sep.className = 'lre-sep';
        toolbar.appendChild(sep);
        return;
      }
      var btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'lre-btn';
      btn.title = b.title;
      btn.innerHTML = b.html;
      btn.addEventListener('mousedown', function (e) {
        e.preventDefault(); // не терять выделение/фокус
      });
      btn.addEventListener('click', function (e) {
        e.preventDefault();
        area.focus();
        if (b.action === 'link') {
          doLink();
        } else if (b.cmd === 'formatBlock') {
          // Тоггл: если уже этот блок — вернуть в параграф
          var cur = document.queryCommandValue('formatBlock');
          var want = b.value;
          if (cur && cur.toLowerCase() === want) {
            document.execCommand('formatBlock', false, 'p');
          } else {
            document.execCommand('formatBlock', false, want);
          }
        } else {
          document.execCommand(b.cmd, false, b.value || null);
        }
        updateActive();
        fireChange();
      });
      toolbar.appendChild(btn);
    });

    function doLink() {
      var sel = window.getSelection();
      var hasText = sel && sel.toString().trim() !== '';
      var url = window.prompt('Ссылка (URL):', 'https://');
      if (url === null) return;
      url = url.trim();
      if (!url) {
        document.execCommand('unlink');
      } else {
        // Простейшая валидация схемы
        if (!/^https?:\/\//i.test(url) && !/^mailto:/i.test(url)) url = 'https://' + url;
        if (hasText) {
          document.execCommand('createLink', false, url);
        } else {
          document.execCommand('insertHTML', false, '<a href="' + escAttr(url) + '">' + escHTML(url) + '</a>');
        }
      }
      fireChange();
    }

    function escHTML(s) { return String(s || '').replace(/[&<>"']/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]; }); }
    function escAttr(s) { return escHTML(s); }

    function updateActive() {
      var btns = toolbar.querySelectorAll('.lre-btn');
      var idx = 0;
      BUTTONS.forEach(function (b) {
        if (b.sep) return;
        var btn = btns[idx++];
        if (!btn) return;
        var on = false;
        try {
          if (b.cmd && b.cmd !== 'formatBlock' && b.cmd !== 'removeFormat') {
            on = document.queryCommandState(b.cmd);
          } else if (b.cmd === 'formatBlock' && b.value) {
            var cur = (document.queryCommandValue('formatBlock') || '').toLowerCase();
            on = cur === b.value;
          }
        } catch (e) { on = false; }
        btn.classList.toggle('active', !!on);
      });
    }

    var changeCb = opts.onChange || null;
    function fireChange() {
      if (changeCb) changeCb(getHTML());
    }

    // Очистка вставляемого контента: вставляем как plain text, форматируем уже в редакторе
    area.addEventListener('paste', function (e) {
      e.preventDefault();
      var text = (e.clipboardData || window.clipboardData).getData('text/plain');
      document.execCommand('insertText', false, text);
      fireChange();
    });

    area.addEventListener('input', fireChange);
    area.addEventListener('keyup', updateActive);
    area.addEventListener('mouseup', updateActive);
    area.addEventListener('focus', function () { wrap.classList.add('focused'); });
    area.addEventListener('blur', function () { wrap.classList.remove('focused'); });

    wrap.appendChild(toolbar);
    wrap.appendChild(area);
    container.appendChild(wrap);

    function getHTML() {
      var html = area.innerHTML.trim();
      // Пустой редактор даёт <br> или пустой <p> — нормализуем в ''
      if (html === '<br>' || html === '<p><br></p>' || html === '<div><br></div>' || html === '') return '';
      return html;
    }
    function setHTML(html) { area.innerHTML = html || ''; }
    function isEmpty() { return (area.textContent || '').trim() === '' && area.querySelectorAll('img,hr').length === 0; }
    function destroy() { if (wrap.parentNode) wrap.parentNode.removeChild(wrap); }

    return { getHTML: getHTML, setHTML: setHTML, isEmpty: isEmpty, destroy: destroy, focus: function () { area.focus(); }, element: area };
  }

  window.LastopRichEditor = { create: create };
})();
