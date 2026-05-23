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

// Дерево категорий проектов из словаря (наполняется loadProjectCatTree). Fallback — PROJECT_CATEGORIES.
window.PROJECT_CAT_TREE = [];
window.loadProjectCatTree = async function() {
  try {
    const r = await (window.lastopFetch ? lastopFetch('/api/projects/categories') : fetch('/api/projects/categories'));
    const d = await r.json();
    window.PROJECT_CAT_TREE = d.groups || [];
  } catch (_) { window.PROJECT_CAT_TREE = []; }
};

// Утилита: рендер <optgroup>/<option> внутрь <select>. Источник — дерево словаря (PROJECT_CAT_TREE),
// при пустом дереве — fallback на захардкоженный PROJECT_CATEGORIES (key=label).
// firstOption — текст первого "пустого" пункта. data-parent для подкатегорий (picker строит дерево).
window.renderProjectCategoriesInto = function(selectEl, firstOption) {
  if (!selectEl) return;
  const currentValue = selectEl.value;
  selectEl.innerHTML = '';

  if (firstOption !== null && firstOption !== undefined) {
    const opt = document.createElement('option');
    opt.value = '';
    opt.textContent = firstOption;
    selectEl.appendChild(opt);
  }

  const tree = (window.PROJECT_CAT_TREE && window.PROJECT_CAT_TREE.length) ? window.PROJECT_CAT_TREE : null;
  if (tree) {
    tree.forEach(g => {
      const og = document.createElement('optgroup');
      og.label = g.label;
      const cats = (g.items && g.items.length) ? g.items : [{ key: g.key, label: g.label }];
      cats.forEach(c => {
        const opt = document.createElement('option');
        opt.value = c.key; opt.textContent = c.label;
        og.appendChild(opt);
        (c.items || []).forEach(s => {
          const so = document.createElement('option');
          so.value = s.key; so.textContent = s.label;
          so.setAttribute('data-parent', c.key);
          og.appendChild(so);
        });
      });
      selectEl.appendChild(og);
    });
  } else {
    // fallback: захардкоженное дерево (key=label)
    window.PROJECT_CATEGORIES.forEach(g => {
      const og = document.createElement('optgroup');
      og.label = g.group;
      g.items.forEach(name => {
        const opt = document.createElement('option');
        opt.value = name; opt.textContent = name;
        og.appendChild(opt);
      });
      selectEl.appendChild(og);
    });
  }

  if (currentValue) selectEl.value = currentValue;
  try { selectEl.dispatchEvent(new Event('change',{bubbles:true})); } catch(_){}
};
