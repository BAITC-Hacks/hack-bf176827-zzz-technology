# Разделение работ: Артём и Дамир

Архитектура, правила ролей и формулы — в `.agent/plan.md`. Здесь только кто что делает и в каком порядке.

## Принципы

- Ветки: `master` (фундамент), `feat/analysis` (Артём), `feat/web` (Дамир). Каждый коммитит в свою ветку.
- Файлы не пересекаются. Артём: `internal/analysis/**`, `internal/data/parquet/**`, `internal/graph/**`, `cmd/pipeline/**`, `README.md` (разделы «критерии», «ограничения», «масштабирование»). Дамир: `internal/services/graph/**`, `internal/data/dto/**`, `internal/transport/http/v1/graph/**`, `internal/app/*.go`, `cmd/web/**`, `cmd/check/**`, `web/**`, `Makefile`, `Dockerfile`, `docker-compose.yaml`, `docs/` (swagger), `README.md` (разделы «запуск», «структура», «выходы»).
- Мерж: сначала `feat/web` → `master`, потом Артём делает `git rebase master` и мержит `feat/analysis`. Конфликт возможен только в README — разделы разные, решается руками.
- Контракт между ветками фиксируется в шаге A0 и не меняется без обсуждения: структуры `analysis.Result`, `analysis.NodeResult`, `analysis.ClusterResult` и формат `out/graph.json`.

## Контракт (фиксируется в A0)

```go
// internal/analysis/result.go
type NodeResult struct {
    Gid           int64
    Role          string   // consolidator|transit|distributor|terminal|coordinator|peripheral
    RoleScore     float64  // 0..1
    ClusterID     int
    PriorityScore float64  // 0..1
    Evidence      string   // ≤200 символов, по-русски, с числами
    Features      Features // все метрики узла (in_deg, out_deg, in_kzt, ... см. plan.md)
}
type ClusterResult struct {
    ClusterID       int
    NNodes, NSeed   int
    SumKZTInternal  float64
    TopGids         []int64
    Hypothesis      string
}
type TopNode struct { Rank int; Gid int64; Role string; PriorityScore float64; Why string }
type Result struct {
    Nodes    []NodeResult            // ровно 2248
    Clusters []ClusterResult
    Top      []TopNode               // ≥ 20, отсортирован по убыванию
    Edges    []parquet.Edge          // как в данных
    ByGid    map[int64]*NodeResult
}
func Run(ds parquet.Dataset, opts Options) (*Result, error)
```

`out/graph.json` (пишет pipeline, читает UI напрямую или через API; **gid строкой**):

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

## Артём (тяжёлая часть: фундамент + аналитика)

### A0. Фундамент в `master` (первые 25 минут, Дамир в это время делает D0)

1. `git add -A && git commit -m "skeleton"` — закоммитить каркас как есть.
2. Выпилить Postgres: удалить `internal/app/db.go`, `internal/repo/`, `internal/services/items/`, `internal/data/dto/items.go`, `internal/transport/http/v1/items/`, `migrations/`, `sqlc.yaml`; из `internal/app/internal.go` убрать репозитории и items; из `internal/app/handlers.go` убрать items; из `cmd/web/main.go` убрать `ModuleDB`, `ModuleRepositories`; из `internal/config` убрать `PostgresConfig`; из `config/*.yaml` секцию postgres. `go build ./...` зелёный.
3. `go get github.com/parquet-go/parquet-go gonum.org/v1/gonum`.
4. `internal/data/parquet/loader.go`: структуры `Edge`, `Node`, `Tx` (теги как в plan.md, `date,date`), `Dataset`, `Load(dir)`, `SanityCheck()` (edges == tx по парам, orphans, seed count). Проверено: читается, сумма 365 890 012.01.
5. `internal/graph/graph.go`: `Build(ds)` → `simple.WeightedDirectedGraph` (node id = gid) + `Out/In map[int64][]Edge` + `TxByNode`.
6. `internal/analysis/result.go` — контракт выше. `internal/analysis/analysis.go` — `Run()` с заглушкой: все роли `peripheral`, `role_score 0`, `cluster_id` = номер компоненты, `priority` = нормированный in_kzt, `evidence` = "заглушка", top = 30 по priority.
7. `internal/analysis/export.go`: `WriteCSV(res, dir)` (три файла, схема ТЗ + доп. колонки с фичами) и `WriteGraphJSON(res, dir)` (формат выше, gid строкой).
8. `cmd/pipeline/main.go`: флаги `--data data --out out`, лог таймингов. Положить `data/*.parquet` в репо.
9. Прогнать, убедиться: 2248 строк, три файла, graph.json. Коммит `foundation`, пуш `master`. **Сообщить Дамиру** — он ребейзит `feat/web` на это.
10. `git checkout -b feat/analysis`.

### A1. Метрики (`internal/analysis/metrics.go`, ~30 мин)

Полный список фич из plan.md: степени, суммы, число переводов, средние чеки, `pass_through`, PageRank
(`network.PageRank` на weighted-графе, damp 0.85), HITS, `network.Betweenness` (невзвешенный), `n_seed_payers`,
`n_seed_upstream` (BFS от каждого seed по исходящим до 4 шагов), `component_id/size`, `truncated`, `verified_sink`.
Из транзакций: `active_days`, `max_same_day_payers`, `fast_forward_share` (доля исходящего, ушедшего в течение
2 дней после входящего), `reciprocal`. Вывести распределения в лог (квантили) — по ним ставить пороги.

### A2. Кластеры (`clusters.go`, ~20 мин)

`community.Modularize(graph.Undirect{G: g}, 1.0, rand.NewPCG(42, 0))` → communities; изолированным узлам —
свои cluster_id. `n_clusters_in` в фичи. `ClusterResult` со статистикой и `hypothesis` по шаблону из состава ролей
(заполняется после A3). Проверить: ~60–70 кластеров, ~9 с >1 seed.

### A3. Роли и evidence (`roles.go`, ~40 мин)

Скоры шести ролей по таблице plan.md, argmax, порядок при равенстве
(coordinator > consolidator > distributor > transit > terminal > peripheral). Truncated не могут быть terminal;
у seed transit запрещён; у truncated consolidator скор ×0.7. Шаблоны evidence по-русски с числами, обрезка до 200.
Спот-чек: узел с in_deg 24, узел с out_deg 116, узел с pass_through≈1, любой depth=4, любой seed без рёбер.

### A4. Приоритет и топ (`priority.go`, ~20 мин)

Перцентильные ранги, веса из plan.md, множители seed ×0.5 и truncated ×0.7, `why`. Top 30.
`go run ./cmd/pipeline` → `go run ./cmd/check` (когда Дамир его сделает) зелёный. Коммит.

### A5. Паттерны (`patterns.go`, ~40 мин, после мержа must-have)

Циклы ≤5 (DFS с ограничением), повторяющиеся маршруты A→B→C (все три узла с ≥2 tx на рёбрах),
устойчивость: удалить top-5/10/20 по priority → сколько компонент и какая доля оборота отвалилась.
Результаты — в evidence (`; в цикле длины 3`), в `clusters.csv` (доп. колонки) и в README.

### A6. README-разделы и демо (~30 мин)

Таблица критериев ролей с порогами и формулами, формула приоритета, кластеризация (оговорить неориентированную
проекцию), учёт ловушек данных, ограничения, масштабирование до 1 млн узлов. Выбрать 3 gid для демо и написать
объяснение каждого «за минуту» в `docs/demo.md`.

---

## Дамир (типовая часть: API + UI + инфраструктура + проверка)

### D0. Пока Артём делает фундамент (первые 25 минут, ветка `feat/web` от текущего `master`)

1. `git checkout -b feat/web`.
2. Скачать cytoscape.js (`cytoscape.min.js`, последний 3.x) в `web/vendor/`. Не CDN — интернета на площадке может не быть.
3. `web/index.html` — скелет: контейнер графа на всю ширину, правая панель (карточка узла), верхняя панель (поиск по gid, фильтр роли, фильтр кластера, кнопка «топ-30»), легенда цветов ролей. Пока грузит `graph.json` по фиксированному пути из мок-файла `web/mock/graph.json` (сделать руками 10 узлов в формате контракта).
4. `cmd/check/main.go`: читает `out/*.csv`, проверяет: `nodes_roles.csv` ровно 2248 строк и уникальные gid; role из словаря; `role_score`, `priority_score` ∈ [0,1]; `evidence` непустой и ≤200; `cluster_id` у всех и каждый есть в `clusters.csv`; `top_nodes.csv` ≥ 20 строк, `rank` 1..N, `priority_score` невозрастающий. Ненулевой exit-код при ошибке, понятные сообщения.
5. Makefile: убрать goose/sqlc/db-цели; добавить `pipeline` (`go run ./cmd/pipeline --data data --out out`), `check`, `web` (`go run ./cmd/web`), `demo` (`pipeline` + `check` + `web`).

### D1. После коммита `foundation` в master (~40 мин)

1. `git rebase master`.
2. `internal/services/graph/service.go`: на старте (fx `OnStart`) вызывает `parquet.Load` + `analysis.Run` (путь к данным из config `app.data_dir`, дефолт `data`), держит `*analysis.Result` в памяти. Методы: `Graph(filter)`, `Node(gid)`, `Ego(gid, depth)`, `Top(n)`, `Clusters()`, `Search(prefix)`.
3. `internal/data/dto/graph.go`: `NodeDTO` (gid `string`!), `EdgeDTO`, `ClusterDTO`, `TopDTO`, `NodeCardDTO` (роль, скоры, evidence, все фичи, входящие и исходящие соседи с суммами). Маппинг из `analysis.*`.
4. `internal/transport/http/v1/graph/handler.go` по образцу удалённого `items`: `GET /graph?role=&cluster=&component=`, `GET /nodes/{gid}`, `GET /nodes/{gid}/ego?depth=1`, `GET /top?n=30`, `GET /clusters`, `GET /search?q=`. gid парсить `strconv.ParseInt`, 404 через `httperr.NotFound`. Swagger-аннотации, `@Router` без `/v1`. Регистрация в `router.go` и `internal/app/handlers.go`, сервис в `internal/app/internal.go`. `make swag`.
5. Раздача UI: `web/embed.go` с `//go:embed index.html vendor/*`, в `internal/app/server.go` подключить `filesystem` middleware Fiber на `/`. Проверить `curl localhost:8080/v1/top`, swagger открывается, `/` отдаёт страницу.
6. Коммит.

### D2. UI по-настоящему (~50 мин)

1. Источник данных: `/v1/graph` (fallback на `graph.json`, если API недоступен).
2. Cytoscape: направленные рёбра со стрелками (`curve-style: bezier`, `target-arrow-shape: triangle`), ширина ∝ log(sum_kzt), цвет узла по роли (6 цветов + легенда), seed — обводка/ромб, размер ∝ priority.
3. Стартовый вид: топ-30 узлов + их соседи 1 шага (layout `cose`), кнопка «вся сеть» (крупнейшая компонента).
4. Поиск по gid: центрирование, подсветка входящих (один цвет) и исходящих (другой), остальное приглушить. Карточка справа: роль, role_score, priority, cluster, evidence, метрики, списки соседей с суммами (клик по соседу — переход). Кнопка «ego 2 шага» → `/nodes/{gid}/ego?depth=2`.
5. Панель «Топ приоритетов»: таблица из `/v1/top`, клик — фокус на узле. Фильтры по роли и кластеру.
6. Проверка сценария жюри: назвать gid из top → найти на схеме → показать связи. Коммит, **мерж `feat/web` → `master`**, сказать Артёму.

### D3. Инфраструктура и воспроизводимость (~30 мин)

1. Dockerfile: один образ, копирует `data/`, `config/`, бинари pipeline и web; `ENTRYPOINT` запускает pipeline, потом web. `docker-compose.yaml`: только сервис `app`, порт 8080, без Postgres.
2. README, разделы: «Запуск» (одна команда `make demo`, альтернатива `docker compose up --build`, требования: Go 1.25 или Docker), «Структура репозитория», «Что на выходе» (схемы трёх CSV и graph.json), «API» (список эндпоинтов, ссылка на swagger), «Интерфейс» (скриншот).
3. Прогон с чистой машины: `git clone` в `/tmp/x`, `make demo`, замер времени (ожидание: секунды, лимит 5 минут). Затем `docker compose up --build`.
4. Схема решения: Mermaid-диаграмма «данные → метрики → роли → интерфейс» в README + экспорт в PNG `docs/scheme.png` (это «один слайд» из ТЗ).

### D4. Опционально, если есть время

Вкладка «Кластеры» в UI (таблица `clusters.csv`, клик — подсветить кластер). Кнопка «карточка узла» экспортом в текст (для запроса в органы). Тёмная тема не нужна.

---

## Контрольные точки

| Время | Что должно быть |
|---|---|
| 0:25 | `master`: foundation, pipeline пишет 3 CSV с заглушками. `feat/web`: UI-скелет на моке, `cmd/check`, Makefile. |
| 1:30 | `feat/analysis`: реальные роли, кластеры, приоритет, `make check` зелёный. `feat/web`: API + UI на реальном graph.json. |
| 2:00 | Оба смержены в `master`. `make demo` работает end-to-end. Must-have 1, 2, 4, 5 закрыты. |
| 3:00 | README полный (must-have 3), прогон с чистой машины, Docker. |
| 4:00 | Паттерны, устойчивость, полировка UI, `docs/demo.md` с 3 разобранными gid. |
| 4:30 | Freeze. Только правки README и репетиция демо. |
