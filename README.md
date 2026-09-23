# hack-bf176827-zzz-technology
Hackathon team repository for Zzz Technology

## Запуск

Самый простой путь — Docker (нужен только Docker с Compose, интернет на время сборки):

```sh
docker compose up --build
```

Через минуту-две откройте http://localhost:8080 — интерфейс, API (`/v1/*`) и Swagger (`/swagger/index.html`)
на одном порту. React собирается внутри образа, Node.js на машине не нужен. Выгрузки `nodes_roles.csv`,
`clusters.csv`, `top_nodes.csv`, `graph.json` появляются в `out/` на хосте. Остановка — `docker compose down`.

Без Docker (нужен Go 1.25+):

```sh
make demo        # pipeline → check → web на :8080 со встроенным просмотрщиком
make demo-full   # то же с React-интерфейсом; дополнительно нужны Node.js 22+ и npm
```

Пайплайн отрабатывает меньше секунды, выгрузки — в `out/`. Опциональный LLM-слой (справка по узлу,
ассистент, гипотезы кластеров) включается ключом `OPENAI_API_KEY` в `.env` или окружении; без ключа
всё работает на шаблонах, а 53 гипотезы берутся из закоммиченного кэша `out/llm_cache.json`.

Для разработки UI: `cd frontend && npm ci && npm run dev` (http://localhost:5173, прокси к API на 8080).
Только Go-сервер — `make web`; на старте он сам считает анализ в памяти. Настройки — `config/base.yaml`;
переопределения: `APP_PORT`, `APP_DATA_DIR`, `APP_OUT_DIR`, `APP_ENVIRONMENT`, `LLM_MODEL`.

## Структура репозитория

| Путь | Назначение |
|---|---|
| `data/` | Исходные Parquet |
| `cmd/pipeline/` | Расчёт и экспорт |
| `cmd/check/` | Проверка трёх CSV |
| `cmd/web/` | HTTP-сервер и Docker entrypoint |
| `internal/data/models/`, `internal/data/graph/` | Доменные модели, граф и индекс рёбер |
| `internal/repo/dataset/` | Загрузка и проверка parquet |
| `internal/services/analysis/` | Метрики, роли, кластеры, приоритет, паттерны |
| `internal/services/{pipeline,export,hypotheses,assistant}/` | Оркестрация, выгрузки, LLM-слой |
| `internal/services/graph/` | Индексы, фильтры, поиск и окружение |
| `internal/data/dto/{graph,assistant}/` | JSON-контракт, gid строками |
| `internal/transport/http/v1/{graph,assistant}/` | HTTP-хендлеры |
| `frontend/` | React + Vite + Cytoscape; собирается в Docker или `make frontend-build` |
| `web/` | Встроенный резервный просмотрщик этапа D0 |
| `docs/` | Swagger, методология (`methodology.md`), сценарий демо (`demo.md`), схема решения (`scheme.png`) |

## Что на выходе

| Файл | Основные колонки / поля |
|---|---|
| `nodes_roles.csv` | `gid,role,role_score,cluster_id,priority_score,evidence` и метрики |
| `clusters.csv` | `cluster_id,n_nodes,n_seed,sum_kzt_internal,top_gids,hypothesis` и агрегаты |
| `top_nodes.csv` | `rank,gid,role,priority_score,why` |
| `graph.json` | `nodes,edges,clusters,top`; идентификаторы строками |

`make check` проверяет 2248 уникальных узлов, допустимые роли, score в [0,1],
непустой evidence до 200 символов, покрытие кластеров, минимум 20 ранжированных
узлов с невозрастающим приоритетом. Ошибка содержит файл и номер строки
для некорректной записи; процесс завершается с кодом 1.

## API

| GET | Назначение |
|---|---|
| `/v1/graph?role=&cluster=&component=&top=30` | Подграф; top-N плюс соседи одного шага, затем фильтры |
| `/v1/nodes/{gid}` | Узел с метриками и списками `incoming`, `outgoing` |
| `/v1/nodes/{gid}/ego?depth=1` | Окружение в обоих направлениях, глубина 1–2 |
| `/v1/top?n=30` | Приоритетные узлы |
| `/v1/clusters` | Кластеры и гипотезы |
| `/v1/search?q=1000&limit=10` | Поиск по началу gid |
| `/v1/health` | Состояние HTTP-сервера |
| `/graph.json` | Выгрузка пайплайна для резервного источника UI |

`top=0` или отсутствие `top` возвращает всю сеть. Пустые выборки — массивы `[]`.
Некорректные параметры дают 400, отсутствующий узел — 404.
Во всех JSON идентификаторы узлов передаются строками без потери точности.

## Интерфейс

Основной интерфейс написан на **React** и находится в `frontend/`.
Production-сборка `frontend/dist/` раздаётся Nginx в Compose либо Go-сервером
после `make demo`. Cytoscape включён в сборку, внешних CDN нет.

Стартовый экран показывает топ-30 с соседями. Полный gid или его префикс
в строке поиска → Enter → карточка и подсветка связей. Входящие стрелки
бирюзовые, исходящие оранжевые; остальные связи приглушены. Клик по соседу
открывает его карточку. «Ego 2 шага» догружает окружение, «Скопировать»
копирует карточку, если браузер разрешает доступ к буферу обмена.

Роль задаёт цвет, приоритет — размер, seed — форму ромба. Наведение
на ребро показывает сумму, на узел — полный gid. Доступны фильтры ролей
и кластеров, список приоритетов с обоснованиями и список кластеров.
«Вся сеть» загружает все узлы и фокусирует крупнейшую компоненту.

UI использует `/v1/*`; при недоступности API читает `graph.json` рядом
со страницей. Добавьте `?demo=1` к адресу React-интерфейса для явного включения
тестового графа из 10 узлов. Реальные ошибки API не подменяются моками.
При отсутствии React-сборки Go использует резервный просмотрщик `web/`.

![Интерфейс графа](docs/ui.png)

```mermaid
flowchart LR
    P[Parquet] --> L[Загрузка]
    L --> M[Метрики]
    M --> R[Правила ролей]
    R --> C[Кластеры]
    C --> Q[Приоритет]
    Q --> CSV[CSV и graph.json]
    Q --> API[HTTP API]
    CSV --> UI[Интерфейс]
    API --> UI
```

![Схема решения](docs/scheme.png)

Проверки: `make test`, `make fmt`, `go build ./...`, `make swag`;
для React — `cd frontend && npm test && npm run build`.
Подробнее о фронтенде: [frontend/README.md](frontend/README.md).

Текущая готовность и результаты проверок: [.agent/damir_ready.md](.agent/damir_ready.md).
Docker-прогон в рабочем окружении пока не выполнен: требуется включить
WSL integration в Docker Desktop. Сборка React и запуск через Go проверены.
