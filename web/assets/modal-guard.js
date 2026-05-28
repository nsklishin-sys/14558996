/* modal-guard.js — глобальная защита модальных окон от случайного закрытия.
 *
 * Решает две проблемы (вариант Б):
 *  1. Выделение текста с выходом курсора за пределы окна закрывало модалку
 *     (mousedown внутри модалки, mouseup на фоне → срабатывал клик по фону).
 *     Фикс: закрытие по фону допускается ТОЛЬКО если mousedown тоже был на фоне.
 *  2. Случайный клик по фону терял заполненные поля.
 *     Фикс: если внутри модалки есть заполненные поля — спрашиваем подтверждение
 *     через lastopConfirm перед закрытием.
 *
 * Работает глобально через capture-фазу, не требует правок в самих страницах:
 * перехватывает событие раньше inline-обработчиков onclick="if(event.target===this)close()".
 *
 * Опознавание фонового оверлея: элемент, чьё имя класса содержит "modal-bg",
 * "modal-overlay" или "modal-backdrop" (покрывает modal-bg, mng-modal-bg,
 * nc-modal-bg, mf-overlay-* и т.п. — добавляются по мере необходимости).
 */
(function () {
  'use strict';

  // Класс считается фоновым оверлеем, если содержит один из этих маркеров.
  var OVERLAY_MARKERS = ['modal-bg', 'modal-overlay', 'modal-backdrop'];

  function isOverlay(el) {
    if (!el || el.nodeType !== 1 || typeof el.className !== 'string') return false;
    var cls = el.className;
    for (var i = 0; i < OVERLAY_MARKERS.length; i++) {
      if (cls.indexOf(OVERLAY_MARKERS[i]) !== -1) return true;
    }
    return false;
  }

  // Запоминаем, на каком элементе начался mousedown.
  var downOnOverlay = false;
  document.addEventListener('mousedown', function (e) {
    downOnOverlay = isOverlay(e.target);
  }, true);

  // Есть ли внутри модалки заполненные пользователем поля.
  function hasFilledFields(overlay) {
    var fields = overlay.querySelectorAll('input, textarea, select, [contenteditable="true"]');
    for (var i = 0; i < fields.length; i++) {
      var f = fields[i];
      var tag = f.tagName;
      var type = (f.getAttribute('type') || '').toLowerCase();
      // Пропускаем технические/неинтерактивные поля.
      if (type === 'hidden' || type === 'submit' || type === 'button') continue;
      if (f.disabled || f.readOnly) continue;

      if (f.hasAttribute('contenteditable')) {
        if ((f.textContent || '').trim() !== '') return true;
        continue;
      }
      if (tag === 'SELECT') {
        // Заполнено, если выбрано не первое (дефолтное) значение.
        if (f.selectedIndex > 0) return true;
        continue;
      }
      if (type === 'checkbox' || type === 'radio') {
        if (f.checked) return true;
        continue;
      }
      if ((f.value || '').trim() !== '') return true;
    }
    return false;
  }

  // Флаг: подтверждение уже получено, следующий клик по этому оверлею пропускаем
  // к родному обработчику закрытия (closeModal и т.п.), не перехватывая.
  var bypassNext = null;

  // Перехват клика на фазе capture — раньше inline-onclick на самом оверлее.
  document.addEventListener('click', function (e) {
    var overlay = e.target;
    if (!isOverlay(overlay)) return;            // клик не по фону — не вмешиваемся
    // (e.target === overlay означает клик именно по фону, а не по контенту внутри)

    // Повторный «разрешённый» клик после подтверждения — пропускаем к родному закрытию.
    if (bypassNext === overlay) {
      bypassNext = null;
      return;
    }

    // 1. Выделение текста: mousedown был не на фоне → «протащили» курсор. Не закрываем.
    if (!downOnOverlay) {
      e.stopImmediatePropagation();
      e.preventDefault();
      return;
    }

    // 2. Честный клик по фону. Если есть заполненные поля — подтверждение.
    if (hasFilledFields(overlay)) {
      e.stopImmediatePropagation();
      e.preventDefault();
      var ask = (typeof window.lastopConfirm === 'function')
        ? window.lastopConfirm({ title: 'Закрыть без сохранения?', message: 'Введённые данные будут потеряны.', danger: true, confirmText: 'Закрыть' })
        : Promise.resolve(window.confirm('Закрыть без сохранения? Введённые данные будут потеряны.'));
      Promise.resolve(ask).then(function (ok) {
        if (ok) {
          // Переиспользуем родную логику закрытия страницы: ставим флаг bypass
          // и заново кликаем по фону — на этот раз сработает inline-onclick
          // (closeModal/closeStageModal/...) со всем его сбросом состояния.
          bypassNext = overlay;
          overlay.click();
        }
      });
      return;
    }
    // 3. Полей нет — пропускаем нативное закрытие (inline-onclick сработает сам).
  }, true);
})();
