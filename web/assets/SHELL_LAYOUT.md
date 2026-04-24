# LASTOP GROUP — Shell Layout Standard

Все страницы сайта используют единый каркас (shell). Эталон — `home-auth.html`.

## Размеры

| Область | Размер |
|---|---|
| Левый сайдбар (`.nav`) | 240px (1200px: 220px, 1000px: 220px, 740px: 52px иконки) |
| Верхний топбар (`.topbar`) | высота 56px |
| Правый сайдбар (`.right`) | 300px (1200px: 260px, 1000px: скрыт) |
| Gap между ячейками | 10px |
| Padding вокруг shell | 10px |

## Брейкпоинты

- `default` (≥1201px): 240 / 1fr / 300
- `@media (max-width:1200px)`: 220 / 1fr / 260
- `@media (max-width:1000px)`: 220 / 1fr (правая панель скрыта)
- `@media (max-width:740px)`: 52 / 1fr (иконочный nav)
- `@media (min-width:1921px) and (min-height:1081px)`: zoom 1.18, 250 / 1fr / 340

## Как использовать на новой странице

1. В разметке — стандартный скелет:
```html
   <div class="shell">
     <a class="logo-cell">...</a>
     <header class="topbar">...</header>
     <nav class="nav">...</nav>
     <main class="main">...</main>
     <aside class="right">...</aside>
   </div>
```

2. В `<head>` — две строки подключения:
```html
   <link rel="stylesheet" href="/assets/shell-layout.css">
   <script>document.documentElement.classList.add('lt-shell');</script>
```

3. Для страниц без правой панели (чат, мессенджер-подобные) — заменить `lt-shell` на `lt-shell-2col`.

## Что нельзя делать

- Не определяй `.shell{}` в inline `<style>` страницы с отличными значениями (240, 300, 56, 10 менять нельзя). Внешний файл — единственный источник размеров.
- Не добавляй промежуточные брейкпоинты (1400px, 900px, 840px, 520px) для `.shell{}`. Система из пяти точек (1200, 1000, 740, 1921+) покрывает все нужные случаи.
- Не меняй `grid-template-areas` — имена ячеек фиксированы: `logo`, `topbar`, `nav`, `main`, `right`.
- Не оборачивай содержимое `.shell` в дополнительные контейнеры — это ломает grid.
- Не меняй порядок класса на `<html>` — именно `lt-shell` должен быть, скрипт его ставит синхронно.
