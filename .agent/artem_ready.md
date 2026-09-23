# Что готово у Артёма и что ожидается от Дамира

Ветка `feat/analysis` (от `main` с foundation). Всё ниже собрано, протестировано и прогнано на реальных данных.

## Готово

### Пайплайн — `go run ./cmd/pipeline --data data --out out [--llm]`

- < 1 с. Пишет `out/nodes_roles.csv` (2248 строк), `out/clusters.csv` (88), `out/top_nodes.csv` (30), `out/graph.json`.
- Детерминирован: повторный прогон даёт байт-в-байт те же файлы.
- `--llm` — дозапросить у LLM гипотезы кластеров, которых нет в кэше. Без флага и без ключа берёт готовые тексты из `out/llm_cache.json` (53 гипотезы уже там, файл в репо).

### Пакет `internal/analysis`

```go
res, err := analysis.Run(ds, analysis.Options{TopN: 30})   // ds из parquet.Load(dir)
```

`*analysis.Result`:

| Поле | Что это |
|---|---|
| `Nodes []NodeResult` | все 2248 узлов: `Gid, Role, RoleScore, ClusterID, PriorityScore, Evidence, Features` |
| `ByGid map[int64]*NodeResult` | быстрый доступ по gid |
| `Clusters []ClusterResult` | `ClusterID, ComponentID, NNodes, NSeed, SumKZTInternal, SumKZTIn, SumKZTOut, RoleCounts, NCycles, TopGids, Hypothesis` |
| `Top []TopNode` | `Rank, Gid, Role, PriorityScore, Why` — уже отсортирован |
| `Edges []parquet.Edge` | `Src, Dst, SumKZT, NTx, Depth` как в данных |
| `Cycles []Cycle` | возвратные потоки ≤ 5 шагов (`Nodes []int64, SumKZT`) |
| `Routes []Route` | устойчивые маршруты A→B→C (`A, B, C, NTx, SumKZT`) |
| `Robustness []RobustnessStep` | что будет при изъятии top-5/10/20 (`LostTurnoverShare, ComponentsAfter, LargestComponent, …`) |

`Features` (все метрики узла, json-теги совпадают с колонками CSV): `in_deg, out_deg, in_kzt, out_kzt, in_tx, out_tx, pass_through,
pagerank, hub, authority, betweenness, n_seed_payers, n_seed_upstream, n_clusters_in, component_id, component_size, depth, is_seed,
truncated, verified_sink, active_days, max_same_day_payers, fast_forward_share, reciprocal, in_cycle, repeat_routes`.

Индекс рёбер для соседей и путей (строится один раз, использовать в сервисе графа):

```go
idx := analysis.BuildIndex(res)
idx.In[gid], idx.Out[gid]            // []parquet.Edge, отсортированы по убыванию суммы
idx.Downstream([]int64{gid}, 2)      // map[gid]расстояние — ego вниз
idx.Upstream([]int64{gid}, 2)        // ego вверх
idx.Path(src, dst, 4)                // []int64 или nil
analysis.Robustness(res, map[int64]bool{gid: true})  // изъятие произвольных узлов
```

JSON для фронта уже готов: `analysis.ToGraphJSON(res)` → структура как в `out/graph.json` (gid строками):
`nodes[] {id, role, role_score, cluster, priority, is_seed, depth, evidence, in_deg, out_deg, in_kzt, out_kzt, truncated, component}`,
`edges[] {source, target, sum_kzt, n_tx}`, `clusters[] {id, n_nodes, n_seed, sum_kzt_internal, top_gids[], hypothesis}`,
`top[] {rank, gid, role, priority, why}`, `robustness[]`. Для `/v1/graph` можно отдавать её целиком или фильтровать по узлам.

### LLM-слой — `internal/llm` + `internal/services/assistant`

Проверено на живом ключе (модель `gpt-5.1`, codex-модели проекту недоступны). Ключ — `OPENAI_API_KEY` в `.env`,
модель — `LLM_MODEL` (дефолт в `config/base.yaml`). Без ключа всё работает на шаблонах и кэше.

```go
client := llm.New(llm.Config{APIKey: cfg.LLM.APIKey, Model: cfg.LLM.Model, BaseURL: cfg.LLM.BaseURL})
cache  := llm.OpenCache(filepath.Join(cfg.App.OutDir, "llm_cache.json"))
svc    := assistant.New(res, client, cache)

svc.Enabled() bool                                   // есть ли ключ
svc.Ask(ctx, question) (assistant.Answer, error)     // Answer{Text, Gids []string, Steps, Cached}; без ключа → llm.ErrDisabled
svc.Card(ctx, gidStr) (text string, byLLM bool, err error)  // без ключа шаблон; err ≠ nil и text ≠ "" → шаблон + причина
svc.EnrichHypotheses(ctx, 5, 10) (n int, err error)  // уже вызывается пайплайном
```

Ответ ассистента 8–12 с, карточка ~9 с. Всё кэшируется в `out/llm_cache.json`.

CLI для ручной проверки: `go run ./cmd/ask --q "..."`, `--card <gid>`, `--hypotheses`.

### Документация

- `docs/methodology.md` — ловушки данных, метрики, таблица правил ролей с порогами, кластеры, формула приоритета, паттерны, LLM, ограничения, масштабирование до 1 млн. **В README дать ссылку**, не дублировать.
- `docs/demo.md` — сценарий демо, разбор пяти узлов, вопросы ассистенту, ответы жюри.

## Ожидается от Дамира

### 1. Сервис графа держит один `Result` и один `Index`

В `internal/services/graph`: на старте `parquet.Load(cfg.App.DataDir)` → `analysis.Run` → `analysis.BuildIndex`.
Сервис ассистента должен получить **тот же** `*analysis.Result` (не считать второй раз). Проще всего: fx-провайдер
`*analysis.Result` в `internal/app/internal.go`, от него зависят и graph-сервис, и `assistant.New(...)`.

### 2. Два эндпоинта LLM (хендлер по образцу остальных, в `internal/transport/http/v1/graph` или отдельным `assistant`)

| Эндпоинт | Вход | Выход | Без ключа |
|---|---|---|---|
| `GET /v1/nodes/{gid}/card` | — | `{"gid":"…","text":"…","by_llm":true}` | `by_llm:false`, текст шаблона, статус 200 |
| `POST /v1/assistant` | `{"question":"…"}` | `{"answer":"…","gids":["…"],"steps":2,"cached":false}` | 503 через `httperr.New(503, "llm_disabled", "LLM не настроен")` |
| `GET /v1/assistant/status` | — | `{"enabled":true,"model":"gpt-5.1"}` | `enabled:false` — UI по этому прячет поле вопроса |

Таймаут хендлера ассистента — 90 с (`context.WithTimeout`), у fiber `WriteTimeout` в `server.go` сейчас 30 с — поднять до 120.

### 3. UI: что показывать из готовых данных

- Цвет по `role`, размер по `priority`, обводка `is_seed`, пунктир/серый для `truncated`.
- Карточка: `evidence` целиком, `role_score`, `priority`, `cluster`, метрики из `Features`, соседи из `Index`.
- Кнопка «Справка» → `/card`; поле вопроса → `/assistant`, gid из ответа подсветить на графе.
- Вкладка/панель «Устойчивость»: `robustness[]` из graph.json — «изъятие топ-5: −12 % оборота, 16→51 компонент».
- Гипотезы кластеров — уже человеческий текст со ссылками на gid, показывать в панели кластера.

### 4. `cmd/check` — на текущих выгрузках должен проходить

Проверить на `out/` из этой ветки: 2248 строк, роли из словаря, evidence ≤ 200, cluster_id у всех и все в `clusters.csv`, top 30 по убыванию.

### 5. README и Makefile

- `make pipeline` = `go run ./cmd/pipeline --data data --out out` (без `--llm`, чтобы жюри не нуждалось в ключе).
- Раздел про LLM: «опционально, `OPENAI_API_KEY` в `.env`, модель `gpt-5.1`; без ключа те же тексты из кэша».
- Ссылки на `docs/methodology.md` и `docs/demo.md`.
- В `.dockerignore` не исключать `out/llm_cache.json` и `data/`.

## Порядок мержа

1. Дамир: `git rebase main` → мерж `feat/web` → `main`.
2. Артём: `git rebase main` в `feat/analysis` → мерж. Пересечений по файлам нет; `docs/`, `internal/analysis`, `internal/llm`, `internal/services/assistant`, `cmd/pipeline`, `cmd/ask` — только у Артёма.
3. После мержа Артём подключает эндпоинты из п. 2, если Дамир их не сделал.
