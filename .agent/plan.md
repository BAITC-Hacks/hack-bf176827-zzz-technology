# HackAlem AI «Граф денег» — план реализации на чистом Go

## Context

Хакатон 5 часов, команда 2–4 бэкендера без фронтендера. Вход: 3 parquet (2248 узлов, 3119 рёбер,
4840 tx). Выход: `nodes_roles.csv`, `clusters.csv`, `top_nodes.csv` + экран со схемой сети.
Решено: **чистый Go** на каркасе `~/GolandProjects/hackaton` (Fiber + FX + zap + Viper, swagger),
**Postgres выкидываем** (данные влезают в память, БД жюри только мешает). Каркас ещё не закоммичен.

Проверено заранее (scratchpad):
- Цифры ТЗ совпадают с данными один в один (компоненты 16: 1877/270/17/13/6/6; in_deg до 24; out_deg до 116; pass_through 0.8–1.2 у 72 узлов; 19 seed без рёбер, 12 seed только получатели).
- `parquet-go/parquet-go` v0.32 читает все три файла: структуры `Edge{Src,Dst int64; SumKZT float64; NTx int64; Depth int8}`, `Node{Gid,Depth int64; IsSeed bool}`, `Tx{Src,Dst int64; Date time.Time \`parquet:"date,date"\`; SumKZT float64}`. Сумма 365 890 012.01 сходится.
- gonum v0.16 (в кэше): `network.PageRank` учитывает веса для `WeightedDirected`; `network.HITS`; `network.Betweenness` (невзвешенный, для денег shortest-path по сумме бессмысленен); `community.Modularize(graph.Undirect{G}, 1.0, src)` = Louvain с фиксированным seed; `topo.ConnectedComponents`. Bounded cycles нет → свой DFS до длины 5.
- gid ≈ 1e17 > 2^53: в JSON для UI отдавать **строкой**. В gonum gid можно использовать напрямую как node ID (int64).
- Ловушка обрыва решается структурой выгрузки: 444 узла depth=4 & out=0 — обрезаны; 1091 узел depth<4 & out=0 — обход от них шёл и ничего не нашёл → *подтверждённые* стоки.

## Правила кода

- Комментарии в коде короткие и компактные: одна строка над функцией, если нужна. Ничего объёмного.
- Архитектура, контракты, обоснования порогов, заметки для памяти — в `.agent/*.md` (plan, team, artem, damir),
  из кода при необходимости ссылка на файл. Код не захламлять.

## Архитектура

```
data/*.parquet ──► cmd/pipeline ──► out/{nodes_roles,clusters,top_nodes}.csv + out/graph.json
                        │
                        └─ internal/analysis (общий пакет) ◄── cmd/web (Fiber API + embedded UI)
```

- `cmd/pipeline` — CLI `go run ./cmd/pipeline --data data --out out`, детерминированный (fixed rand seed), < 10 с.
- `cmd/web` — существующий сервер: на старте прогоняет тот же анализ в памяти (или читает `out/`), отдаёт JSON `/v1/*` и статичный UI на `/`. Swagger как в каркасе.
- Один модуль `hackaton`, два бинаря. Для жюри: `make demo` (= pipeline + web) и `docker compose up` (образ с data внутри).

### Пакеты (после рефакторинга раскладка другая — см. `CLAUDE.md`; ниже исходный план)

| Путь | Содержание |
|---|---|
| `internal/data/parquet/loader.go` | структуры выше + `Load(dir) (Dataset, error)` + sanity-check (edges == tx по парам, orphans) |
| `internal/graph/graph.go` | `Graph`: `simple.WeightedDirectedGraph` (вес = sum_kzt) + adjacency maps `Out/In map[int64][]Edge`, `Nodes map[int64]*NodeInfo`, tx по узлам по датам |
| `internal/analysis/metrics.go` | per-node `Features` (см. ниже) |
| `internal/analysis/roles.go` | правила ролей, `role_score`, `evidence` |
| `internal/analysis/clusters.go` | Louvain + компоненты, `clusters.csv` строки, шаблон `hypothesis` |
| `internal/analysis/priority.go` | `priority_score`, `why`, top-N |
| `internal/analysis/patterns.go` | циклы ≤5 (DFS), повторяющиеся маршруты A→B→C, временные признаки (опционально, час 4) |
| `internal/analysis/export.go` | CSV (encoding/csv) + `graph.json` |
| `internal/analysis/analysis.go` | `Run(ds Dataset, opts) Result` — оркестрация, используется обоими cmd |
| `internal/services/graph/service.go` | держит `Result` в памяти, методы для хендлеров (по CLAUDE.md каркаса: логика в services) |
| `internal/data/dto/graph.go` | DTO: `NodeDTO{Gid string ...}`, `EdgeDTO`, `ClusterDTO`, `TopDTO`; gid строкой |
| `internal/transport/http/v1/graph/handler.go` | `GET /graph?cluster=&role=&limit=`, `GET /nodes/{gid}` (карточка: фичи, роль, evidence, соседи in/out), `GET /nodes/{gid}/ego?depth=1` , `GET /top?n=`, `GET /clusters`, `GET /search?q=` |
| `web/index.html` + `web/vendor/cytoscape.min.js` | UI, вшит через `embed.FS`, отдаётся с `/` (`filesystem` middleware Fiber). Библиотеку **вендорить**, не CDN (интернет на площадке не гарантирован) |
| `cmd/check/main.go` | валидатор выгрузок: 2248 строк, роли из словаря, score в [0,1], evidence непустой и ≤200, cluster_id у всех, top ≥ 20 отсортирован. `make check` |

Удалить из каркаса: `internal/app/db.go`, `ModuleRepositories`, `items` (service/dto/handler/queries), `internal/repo`, `migrations`, `sqlc.yaml`, postgres из `docker-compose.yaml`, `PostgresConfig` из `config`. Makefile: убрать goose/sqlc/db-цели, добавить `pipeline`, `check`, `demo`, `web`.

`data/` (87 КБ) кладём в репо, чтобы жюри запускало одной командой.

## Features (per node)

Из графа (по `edges`): `in_deg`, `out_deg`, `in_kzt`, `out_kzt`, `in_tx`, `out_tx`, `pass_through = out_kzt/in_kzt`,
`avg_in_tx`, `avg_out_tx` (сумма/кол-во — стартер напоминает: суммы и число переводов — разные сигналы),
`pagerank` (weighted), `hub`, `authority` (HITS), `betweenness` (unweighted), `n_seed_payers` (сколько seed платят напрямую),
`n_seed_upstream` (сколько seed достигают узла, BFS от seed по исходящим, ≤4 шагов),
`n_clusters_in` (сколько разных кластеров среди плательщиков), `component_id`, `component_size`,
`depth`, `is_seed`, `truncated = depth==4 && out_deg==0`, `verified_sink = depth<4 && out_deg==0 && in_deg>0`.
Из `transactions`: `active_days`, `max_same_day_payers` (макс. число разных плательщиков за день),
`fast_forward_share` (доля исходящего объёма, ушедшего ≤2 дней после входящего), `reciprocal` (есть обратное ребро).

## Правила ролей (документируются в README как таблица с порогами)

Скор каждой роли 0–1, роль = argmax, `role_score` = скор победителя; при равенстве — порядок ниже. Пороги
выведены из распределений (медиана tx 30 000, in_deg ≥ 3 у ~сотни узлов, out_deg ≥ 5 — веерные).

| Роль | Правило (скор) | Заметка |
|---|---|---|
| `consolidator` | `in_deg ≥ 3` и `pass_through < 0.5` (или out_deg ≤ 1); скор = min(1, in_deg/8)·(1 − pass_through/1.0)… с бонусом за `n_seed_payers ≥ 2` | не зависит от исходящих → применим и к обрезанным; для truncated скор ×0.7 + пометка «4-е колено» |
| `distributor` | `out_deg ≥ 5` и `out_deg ≥ 2·in_deg`; скор = min(1, out_deg/30) | seed-статус не мешает (in неизвестен) |
| `transit` | `in_deg ≥ 1`, `out_deg ≥ 1`, `0.7 ≤ pass_through ≤ 1.3`, **не seed** (у seed in занижен); бонус `fast_forward_share ≥ 0.5` | 72 узла с 0.8–1.2 в данных |
| `coordinator` | `in_deg ≥ 2` и `out_deg ≥ 2` и (`n_clusters_in ≥ 2` или `n_seed_upstream ≥ 3`) и betweenness в top-5 % | «связывает группы», кандидат в организаторы; формулируем как гипотезу |
| `terminal` | `verified_sink` и (`in_deg ≥ 2` или `in_kzt ≥ 100 000`); скор = min(1, in_kzt/1 000 000)·0.5 + min(1, in_deg/4)·0.5 | truncated сюда **не попадают** |
| `peripheral` | всё остальное; скор = 1 − max(остальных) | в т.ч. 19 seed без рёбер, truncated без сигналов |

`evidence` (≤200 символов, по-русски, с числами): шаблон на роль, например
`консолидация: получает от 11 плательщиков (из них 3 seed) 4.2 млн, отдаёт дальше 3%; 2 кластера на входе`.
Для truncated добавлять `; исходящие не видны (4-е колено)`.

## Приоритет

`priority_score` = взвешенная сумма перцентильных рангов (устойчиво к выбросам):
`0.25·in_kzt + 0.20·in_deg + 0.15·n_seed_upstream + 0.15·pagerank + 0.10·betweenness + 0.15·role_bonus`
(role_bonus: consolidator/coordinator 1.0, distributor 0.7, transit 0.5, terminal 0.4, peripheral 0).
Множители: seed ×0.5 (уже известны, ТЗ просит сместить фокус), truncated ×0.7 (неполные данные).
`why` = evidence + «приоритет: сумма X, N seed выше по цепочке, PageRank top-K %». `top_nodes.csv` — топ 30.

## Кластеры

- `cluster_id` = Louvain-сообщество (`Modularize` на `graph.Undirect`, resolution 1.0, `rand.NewPCG(42,0)`); ожидаем ~60–70 сообществ, ~9 с >1 seed. В README явно оговорить, что кластеризация на неориентированной проекции.
- Узлы вне рёбер (19 seed) — каждому свой cluster_id (отдельные строки в `clusters.csv`, hypothesis «изолирован: нет переводов ≥5000 в июле»).
- `clusters.csv`: `cluster_id, n_nodes, n_seed, sum_kzt_internal` (рёбра внутри), `top_gids` (top-5 по priority через `;`), `hypothesis` — шаблон по составу ролей: «N seed → K consolidator (X KZT), далее M distributor: признаки сбора с курьеров и веерного вывода» и т.п. Доп. колонки: `component_id`, `n_consolidators`, `n_transit`, `sum_kzt_in/out` (внешние потоки).

## UI (`web/index.html`, cytoscape.js вендорно)

- Загружает `/v1/graph` (все узлы/рёбра; gid строки). Стрелки на рёбрах, ширина ∝ log(sum_kzt), цвет узла = роль, форма/обводка = seed, фильтр по кластеру и роли, легенда.
- По умолчанию показывать крупнейшую компоненту с layout `cose` (2248 узлов ок), либо «топ-30 + их соседи» как стартовый вид (быстрее и понятнее на демо).
- Поиск по gid → центрирование, подсветка входящих/исходящих, боковая карточка: роль, role_score, evidence, метрики, соседи со суммами, кластер. Кнопка «ego-граф 2 шага».
- Панель «Топ приоритетов» (таблица из `/v1/top`), клик → фокус на узле. Это закрывает проверку must-have 5 («жюри называет gid — находим и показываем связи»).

## README (обязателен, 25 баллов)

Разделы: запуск одной командой (`make demo` / `docker compose up`), схема данные → метрики → роли → интерфейс
(Mermaid + PNG в `docs/scheme.png` — это и есть «слайд»), таблица критериев ролей с порогами, формула
приоритета, кластеризация, что на выходе (схемы CSV), учёт ловушек данных (обрыв, seed in занижен, порог 5000,
компоненты), ограничения, масштабирование до ~1 млн узлов (текст: колоночное хранилище, igraph/GraphX/Neo4j GDS,
approximate betweenness по сэмплу, Leiden вместо Louvain, инкрементальный пересчёт per-node фич, индексы),
осторожность формулировок (гипотезы, не обвинения). Инструкция для демо: 2–3 gid для разбора.

## Порядок работ и разделение

Команда: Артём (фундамент + аналитика, ветка `feat/analysis`) и Дамир (API + UI + инфраструктура + проверка,
ветка `feat/web`). Пошаговые задачи каждого, контракт между ветками (`analysis.Result`, `graph.json`),
порядок мержа и контрольные точки по времени — в `.agent/team.md`.

Кратко: A0 фундамент в `main` (25 мин, Дамир параллельно делает UI-скелет на моке, `cmd/check`, Makefile) →
параллельно A1–A4 (метрики, кластеры, роли, приоритет) и D1–D2 (сервис, DTO, хендлеры, UI) → мерж `feat/web`
первым, `feat/analysis` ребейзом → к 2:00 must-have закрыты → A5 паттерны + D3 Docker/README/чистая машина →
A6 README-критерии и `docs/demo.md` → freeze в 4:30.

LLM-слой (Anthropic API, флаг `LLM_API_KEY`): только если после 3:30 есть время и ключ; пайплайн без него не меняется.

## Верификация

1. `go build ./... && make pipeline` — три файла в `out/`, время < 1 мин (ожидание: секунды).
2. `make check` — 2248 строк, все роли из словаря, score ∈ [0,1], evidence ≤ 200 и непустой, cluster_id у всех, top ≥ 20 и отсортирован по убыванию, `clusters.csv` покрывает все cluster_id из `nodes_roles.csv`.
3. Сверка распределения ролей: consolidator ~ десятки, distributor ~ 8–15, transit ~ 50–80, terminal — сотни, peripheral — остальное, truncated не помечены terminal. Спот-чек 3 узлов вручную (top in_deg=24, top out_deg=116, pass_through≈1).
4. `make run` → `curl localhost:8080/v1/top`, `/v1/nodes/<gid>` (gid строкой в JSON), swagger открывается, UI на `/` находит gid из top и подсвечивает связи.
5. Чистая машина: `git clone` в `/tmp/x && cd /tmp/x && make demo` (только Go 1.25) и `docker compose up --build`.
6. Детерминизм: два прогона pipeline дают одинаковые CSV (`diff`).

## Открытое

- LLM-слой (Anthropic API, флаг `LLM_API_KEY`): по умолчанию выключен; включаем только если после 3:30 есть время и ключ.
