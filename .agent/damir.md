# План Дамира (для работы через Codex)

Этот файл самодостаточный: его можно целиком отдать Codex как задание. Подробности архитектуры —
`.agent/plan.md`, разделение с Артёмом — `.agent/team.md`. Правила кода — `CLAUDE.md` в корне
(Codex читает `AGENTS.md`: скопировать `cp CLAUDE.md AGENTS.md`, если ещё нет).

## Контекст задачи (кратко)

Хакатон, 5 часов. Кейс «Граф денег»: по графу внутрибанковских переводов (2248 клиентов-узлов,
3119 рёбер, 4840 транзакций, июль 2026, `data/*.parquet`) присвоить каждому узлу роль
(`consolidator | transit | distributor | terminal | coordinator | peripheral`), кластер и приоритет,
выдать три CSV и **экран просмотра**: схема сети со стрелками направления денег, подсветка ролей и
кластеров, поиск по gid, карточка узла с его связями. Жюри на демо называет gid — надо найти его на
схеме и показать связи. Всё должно работать локально, одной командой, без интернета.

Стек: Go 1.25, Fiber v2, Uber FX, zap, Viper, swag. Postgres из каркаса удаляется (делает Артём в
`main`). Аналитику (роли, кластеры, приоритеты) считает Артём в пакете `internal/analysis`.
Моя часть: HTTP API, веб-интерфейс, инфраструктура запуска, валидатор выгрузок, README-разделы.

## Правила

- Ветка `feat/web`. Коммитить часто. Мержу в `main` первым, Артём ребейзится на меня.
- Мои файлы: `internal/services/graph/**`, `internal/data/dto/**`, `internal/transport/http/v1/graph/**`,
  `internal/app/*.go`, `cmd/web/**`, `cmd/check/**`, `web/**`, `Makefile`, `Dockerfile`,
  `docker-compose.yaml`, `docs/` (swagger), README-разделы «Запуск», «Структура», «Что на выходе», «API», «Интерфейс».
- **Не трогать**: `internal/analysis/**`, `internal/data/parquet/**`, `internal/graph/**`, `cmd/pipeline/**`.
  Если там чего-то не хватает — написать Артёму, не править самому.
- Код по образцу каркаса: логика в `internal/services`, хендлеры только парсят и мапят в DTO, ошибки через
  `pkg/httperr`, swagger-аннотации у каждого хендлера, `@Router` без `/v1`, после правок `make swag`.
  Комментарии по-русски. Перед коммитом `make fmt && go build ./...`.
- В JSON для фронта **gid всегда строкой**: значения ≈ 1e17 больше 2^53, JavaScript теряет точность.
- Интерфейс без внешних CDN: библиотеки вендорить в `web/vendor/`.

## Контракт с аналитикой (фиксирует Артём в коммите `foundation`)

```go
// internal/analysis/result.go
type NodeResult struct {
    Gid           int64
    Role          string
    RoleScore     float64
    ClusterID     int
    PriorityScore float64
    Evidence      string
    Features      Features // in_deg, out_deg, in_kzt, out_kzt, in_tx, out_tx, pass_through,
                           // pagerank, betweenness, depth, is_seed, truncated, verified_sink, ...
}
type ClusterResult struct { ClusterID int; NNodes, NSeed int; SumKZTInternal float64; TopGids []int64; Hypothesis string }
type TopNode      struct { Rank int; Gid int64; Role string; PriorityScore float64; Why string }
type Result struct {
    Nodes    []NodeResult
    Clusters []ClusterResult
    Top      []TopNode
    Edges    []parquet.Edge          // Src, Dst int64; SumKZT float64; NTx int64; Depth int8
    ByGid    map[int64]*NodeResult
}
func Run(ds parquet.Dataset, opts Options) (*Result, error)   // internal/analysis
func Load(dir string) (Dataset, error)                        // internal/data/parquet
```

Пайплайн Артёма также пишет `out/graph.json` — им можно пользоваться для UI до готовности API:

```json
{
  "nodes": [{"id":"100000003684369100","role":"consolidator","role_score":0.82,"cluster":3,"priority":0.91,
             "is_seed":false,"depth":2,"evidence":"...","in_deg":11,"out_deg":1,"in_kzt":4200000,"out_kzt":120000}],
  "edges": [{"source":"1000...","target":"1000...","sum_kzt":678000,"n_tx":2}],
  "clusters": [{"id":3,"n_nodes":120,"n_seed":4,"sum_kzt_internal":12000000,"hypothesis":"..."}],
  "top": [{"rank":1,"gid":"1000...","role":"consolidator","priority":0.91,"why":"..."}]
}
```

---

## D0. Сразу, параллельно с фундаментом Артёма (0:00–0:25)

1. `git checkout -b feat/web`.
2. Скачать `cytoscape.min.js` (3.x) в `web/vendor/`. Проверить, что файл не пустой и открывается.
3. `web/index.html` — каркас страницы: слева граф на всю высоту, справа панель карточки узла (300–360 px),
   сверху строка: поле «gid», кнопка «Найти», селект роли, селект кластера, кнопки «Топ-30» и «Вся сеть»,
   легенда цветов ролей. Стили инлайн, без фреймворков. Данные пока из `web/mock/graph.json`
   (10 узлов, 12 рёбер в формате контракта, сделать руками).
4. `cmd/check/main.go` — валидатор выгрузок (`--out out`):
   - `nodes_roles.csv`: ровно 2248 строк, gid уникальны, `role` из словаря шести ролей, `role_score` и
     `priority_score` в [0,1], `evidence` непустой и ≤200 символов (рун), `cluster_id` есть у всех;
   - `clusters.csv`: непустой, содержит каждый `cluster_id` из `nodes_roles.csv`, колонки
     `cluster_id,n_nodes,n_seed,sum_kzt_internal,top_gids,hypothesis`, `hypothesis` непустой;
   - `top_nodes.csv`: ≥20 строк, `rank` 1..N без пропусков, `priority_score` невозрастающий, `why` непустой;
   - при ошибке — понятное сообщение с номером строки и exit-код 1; при успехе — сводка (распределение ролей).
5. `Makefile`: удалить цели goose/sqlc/db-*/up/down/logs про Postgres; добавить
   `pipeline: go run ./cmd/pipeline --data data --out out`, `check: go run ./cmd/check --out out`,
   `web: go run ./cmd/web`, `demo: pipeline check web`. `make help` показывает всё.
6. Коммит `web skeleton, check, makefile`.

Приёмка: страница открывается файлом и рисует мок со стрелками; `go build ./...` зелёный.

## D1. После коммита `foundation` в `main` (~40 мин)

1. `git rebase main`. Убедиться, что `go run ./cmd/pipeline --data data --out out` работает и `out/graph.json` есть.
2. `internal/services/graph/service.go`: интерфейс `Service` и реализация. На старте (fx `OnStart`) —
   `parquet.Load(cfg.App.DataDir)` → `analysis.Run` → хранить `*analysis.Result` в памяти (мьютекс не нужен,
   только чтение). Индексы: `out map[gid][]edge`, `in map[gid][]edge`. Методы:
   - `Graph(f Filter) (nodes, edges)` — фильтр по роли, кластеру, компоненте, `top_only int`
     (top-N и их соседи 1 шага);
   - `Node(gid) (*NodeCard, error)` — узел, фичи, evidence, входящие и исходящие соседи с суммами и числом переводов;
   - `Ego(gid, depth 1..2)` — подграф;
   - `Top(n)`, `Clusters()`, `Search(prefix string, limit)` — по началу строки gid.
   Регистрация в `internal/app/internal.go`.
3. `internal/data/dto/graph.go`: `NodeDTO{Gid string ...}`, `EdgeDTO{Source, Target string ...}`,
   `NodeCardDTO`, `ClusterDTO`, `TopDTO`, `GraphResponse`. Маппинг из `analysis.*`, gid через `strconv.FormatInt`.
4. `internal/transport/http/v1/graph/handler.go` (по образцу удалённого `items`, см. историю git):
   - `GET /graph?role=&cluster=&component=&top=`
   - `GET /nodes/{gid}` — 404 через `httperr.NotFound`, gid парсить `strconv.ParseInt`, 400 при мусоре
   - `GET /nodes/{gid}/ego?depth=1`
   - `GET /top?n=30`
   - `GET /clusters`
   - `GET /search?q=1000&limit=10`
   Роуты в `internal/transport/http/v1/router.go`, регистрация в `internal/app/handlers.go`. `make swag`.
5. Раздача UI: `web/embed.go` с `//go:embed index.html vendor/*`, в `internal/app/server.go` —
   `filesystem` middleware Fiber на `/` (после `/v1` и `/swagger`, чтобы не перехватывать).
6. Коммит `graph api + static ui`.

Приёмка: `make web` → `curl localhost:8080/v1/top` отдаёт JSON с gid-строками; `/v1/nodes/<gid из top>`
отдаёт карточку с соседями; `/swagger/index.html` открывается; `/` отдаёт страницу.

## D2. Интерфейс по-настоящему (~50 мин)

1. Источник данных — `/v1/graph`; если API недоступен, fallback на `graph.json` рядом со страницей.
2. Cytoscape-стили: рёбра направленные (`curve-style: bezier`, `target-arrow-shape: triangle`),
   ширина ∝ `log(sum_kzt)`, подпись суммы при наведении; узлы: цвет по роли (шесть цветов, легенда),
   seed — ромб или толстая обводка, размер ∝ `priority`, подпись — последние 6 цифр gid (полный в tooltip).
3. Стартовый вид: `?top=30` (топ-30 плюс соседи), layout `cose`. Кнопка «Вся сеть» грузит всё и
   раскладывает крупнейшую компоненту (2248 узлов cose раскладывает за несколько секунд — показать спиннер).
4. Поиск: ввод gid (или префикса через `/v1/search` с подсказками) → узел центрируется и увеличивается,
   входящие рёбра одним цветом, исходящие другим, всё остальное приглушено. Если узла нет на экране —
   догрузить `/v1/nodes/{gid}/ego?depth=1` и добавить в граф.
5. Правая карточка: роль, role_score, priority, cluster, evidence, таблица метрик, списки входящих и
   исходящих с суммами (клик по соседу — переход к нему), кнопка «Ego 2 шага».
6. Панель «Топ приоритетов»: таблица из `/v1/top` (rank, gid, role, priority, why), клик — фокус на узле.
   Фильтры по роли и кластеру перерисовывают граф.
7. Репетиция сценария жюри: назвать gid из top → найти → показать связи, за 10 секунд.
8. Коммит, **мерж `feat/web` → `main`** (`git checkout main && git merge feat/web`), сказать Артёму.

Приёмка: сценарий жюри проходит; страница работает без интернета (отключить сеть и перезагрузить).

## D3. Инфраструктура и воспроизводимость (~30 мин)

1. `Dockerfile`: multi-stage, собрать `pipeline`, `check`, `web`; в runtime-образ скопировать `data/`, `config/`;
   `ENTRYPOINT` — shell-скрипт: pipeline → check → web. `docker-compose.yaml`: один сервис `app`, порт 8080,
   Postgres удалить.
2. README-разделы (остальные пишет Артём, не затирать его текст):
   - «Запуск»: требования (Go 1.25 **или** Docker), `make demo`, альтернатива `docker compose up --build`,
     где открыть (`http://localhost:8080`, swagger), где лежат выгрузки (`out/`);
   - «Структура репозитория»;
   - «Что на выходе»: схемы `nodes_roles.csv`, `clusters.csv`, `top_nodes.csv`, `graph.json`;
   - «API»: таблица эндпоинтов;
   - «Интерфейс»: что умеет, скриншот в `docs/ui.png`.
3. Прогон с чистой машины: `git clone <repo> /tmp/x && cd /tmp/x && make demo`, засечь время (лимит ТЗ 5 минут,
   ожидание — секунды). То же через `docker compose up --build`.
4. Схема решения: Mermaid в README «parquet → загрузка → метрики → правила ролей → кластеры → приоритет →
   CSV + API → UI» и её PNG в `docs/scheme.png` (это «один слайд» из ТЗ).

## D4. Подключение LLM-фич Артёма (когда его эндпоинты появятся, ~20 мин)

Артём делает на бэкенде: `GET /v1/nodes/{gid}/card` (текстовая справка по узлу) и
`POST /v1/assistant {"question": "..."}` → `{"answer": "...", "gids": ["..."]}`. Моя часть в UI:
1. В карточке узла кнопка «Справка» → показать текст из `/card` (с индикатором загрузки).
2. Внизу правой панели поле «Вопрос ассистенту» → ответ текстом, все gid из `gids` подсветить на графе,
   клик по gid в ответе — фокус на узле.
3. Если эндпоинты отвечают 503 (нет ключа) — кнопки скрыть, не ломать страницу.

## D5. Если остаётся время

Вкладка «Кластеры»: таблица из `/v1/clusters`, клик — подсветить кластер на графе. Экспорт карточки узла
в текст (кнопка «Скопировать»). Хоткей Enter в поиске.

---

## Контрольные точки

| Время | Готово |
|---|---|
| 0:25 | UI-скелет на моке, `cmd/check`, Makefile — в `feat/web` |
| 1:05 | API + раздача UI на реальном `graph.json` |
| 2:00 | UI полный, `feat/web` смержен в `main`, сценарий жюри проходит |
| 2:30 | Docker, README-разделы, прогон с чистого клона |
| 3:00 | LLM-кнопки в UI, схема-слайд |
| 4:30 | Freeze |
