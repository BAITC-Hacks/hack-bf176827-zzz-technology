# React-интерфейс «Граф денег»

React 19 + Vite 7 + cytoscape.js на токенах дизайн-системы (`src/styles/design-system.css`).
Все зависимости входят в локальную сборку, внешних CDN нет.

## Как отдаётся в проде

Отдельного фронт-сервера нет. В Docker-образе React собирается на node-стадии, а Go-сервер раздаёт
`frontend/dist` и API на одном порту: http://localhost:8080. Из корня репозитория:

```sh
docker compose up --build
```

Локально без Docker: `make demo-full` (нужны Node.js 22+ и npm) собирает интерфейс и запускает сервер.
Если `frontend/dist` нет, сервер отдаёт встроенный резервный просмотрщик из `web/`.

## Разработка

Node.js 22.12+ (рекомендуется 24), npm; Go API запускается отдельно.

```sh
# первый терминал, из корня:
make web
# второй терминал:
cd frontend
npm ci
npm run dev
```

Откройте http://localhost:5173. Vite проксирует `/v1`, `/swagger` и `/graph.json` на http://127.0.0.1:8080;
другой адрес задаётся переменной `API_PROXY_TARGET`.

Проверки: `npm test` (операции с графом), `npm run build`.

## Структура

```
src/
  App.jsx                 состояние экрана, загрузка данных, выбор узла, маршрут
  api.js                  fetch с разбором ошибок API
  graph.js                чистые операции над графом (фильтр, окружение, слияние)
  lib/amlLogic.js         роли, форматирование, сигналы по метрикам, маршрут
  lib/graphStyle.js       стили cytoscape по токенам, элементы графа
  hooks/useRoute.js       маршрут просмотра в localStorage
  components/             Toolbar, SelectedStrip, GraphCanvas, RouteBar, SidePanel, NodeTab, Assistant
  styles/                 design-system.css (библиотека), tokens.css (цвета ролей)
  styles.css              раскладка экрана
```

Режим демо с тестовыми данными без бэкенда: http://localhost:5173/?demo.
