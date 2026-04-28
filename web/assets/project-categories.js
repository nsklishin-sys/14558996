// Категории проектов LASTOP. Структура: 6 групп, ~34 категории.
// Спринт 6.5: подключить через GET /api/categories/projects (админ-панель).
// До тех пор список захардкожен здесь.

window.PROJECT_CATEGORIES = [
  {
    group: 'Производство и промышленность',
    items: [
      'Машиностроение',
      'Металлургия и металлообработка',
      'Электроника и приборостроение',
      'Химическая промышленность',
      'Пищевая промышленность',
      'Лёгкая промышленность',
      'Деревообработка и мебель',
      'Строительные материалы',
      'Прочее производство'
    ]
  },
  {
    group: 'Логистика и торговля',
    items: [
      'Логистика и перевозки',
      'ВЭД и таможня',
      'Складская логистика',
      'Оптовая торговля',
      'Розничная торговля и e-commerce',
      'Транспортная инфраструктура'
    ]
  },
  {
    group: 'Технологии',
    items: [
      'IT и разработка ПО',
      'Автоматизация и цифровизация',
      'Телекоммуникации',
      'Кибербезопасность',
      'Электронная коммерция (платформы)'
    ]
  },
  {
    group: 'Услуги бизнесу',
    items: [
      'Финансы и инвестиции',
      'Юридические услуги',
      'Консалтинг и аудит',
      'Маркетинг и реклама',
      'HR и подбор персонала',
      'Образование и обучение'
    ]
  },
  {
    group: 'Инфраструктура и ресурсы',
    items: [
      'Строительство и недвижимость',
      'Энергетика и ЖКХ',
      'Сельское хозяйство и АПК',
      'Добыча полезных ископаемых'
    ]
  },
  {
    group: 'Социальная сфера и прочее',
    items: [
      'Медицина и фарма',
      'Туризм и гостеприимство',
      'Медиа и развлечения',
      'Экология и переработка',
      'Социальные проекты и НКО',
      'Другое'
    ]
  }
];

// Утилита: рендер <option> и <optgroup> внутрь существующего <select>.
// firstOption — текст первого "пустого" пункта (например "Все категории" для фильтра, "Выберите категорию" для формы).
// Если firstOption=null или undefined — пустой пункт не добавляется.
window.renderProjectCategoriesInto = function(selectEl, firstOption) {
  if (!selectEl) return;
  // Сохраняем текущее значение чтобы восстановить после перерисовки
  const currentValue = selectEl.value;
  selectEl.innerHTML = '';

  if (firstOption !== null && firstOption !== undefined) {
    const opt = document.createElement('option');
    opt.value = '';
    opt.textContent = firstOption;
    selectEl.appendChild(opt);
  }

  window.PROJECT_CATEGORIES.forEach(g => {
    const og = document.createElement('optgroup');
    og.label = g.group;
    g.items.forEach(name => {
      const opt = document.createElement('option');
      opt.value = name;
      opt.textContent = name;
      og.appendChild(opt);
    });
    selectEl.appendChild(og);
  });

  // Восстанавливаем значение если оно есть в новом списке
  if (currentValue) {
    selectEl.value = currentValue;
  }
};
