# Что готово у Артёма и что ожидается от Дамира

Ветка `main` после рефакторинга архитектуры. Всё ниже собрано, протестировано и прогнано на данных;
CSV байт-в-байт совпадают с версией до рефакторинга. Правила и слои — `CLAUDE.md`.

## Готово

### Пайплайн — `go run ./cmd/pipeline --data data --out out [--llm]`

- < 1 с. Пишет `out/nodes_roles.csv` (2248), `out/clusters.csv` (88), `out/top_nodes.csv` (30), `out/graph.json`.
- Детерминирован (тест `tests/unit/analysis`). `--llm` дозапрашивает у LLM гипотезы, которых нет в кэше;
  без флага и без ключа берёт 53 готовых из `out/llm_cache.json` (файл в репо).

### Один результат на процесс — `*models.AnalysisResult`

Провайдится в `internal/app/internal.go` (`newAnalysisResult` через `pipeline.Service`). Сервис графа Дамира
берёт его из fx как зависимость, второй раз ничего не считает:

```go
type ServiceParams struct {
    services.FxBaseParams
    Result *models.AnalysisResult
}
```

`models.AnalysisResult` (`internal/data/models/analysis.go`):

| Поле / метод | Что это |
|---|---|
| `Nodes []NodeResult` | все 2248: `GID, Role, RoleScore, ClusterID, PriorityScore, Evidence, Features` |
| `Node(gid) (*NodeResult, bool)` | доступ по GID |
| `Clusters []ClusterResult`, `Cluster(id)` | `ClusterID, ComponentID, NodeCount, SeedCount, SumKZTInternal/In/Out, RoleCounts, CycleCount, TopGIDs, Hypothesis` |
| `Top []TopNode` | отсортирован: `Rank, GID, Role, PriorityScore, Why` |
| `Edges []Edge` | `Payer, Payee, SumKZT, TxCount, Depth` |
| `Cycles, Routes, Robustness` | паттерны (см. `models/patterns.go`) |

`models.Features` — все метрики узла (`InDegree, OutDegree, InKZT, OutKZT, PassThrough, PageRank, Betweenness,
SeedPayers, SeedUpstream, ClustersIn, ComponentID, Truncated, VerifiedSink, FastForwardShare, InCycle, RepeatRoutes, …`).

Индекс рёбер для соседей и путей: `graph.NewIndex(result.Edges)` → `Incoming/Outgoing[gid]`, `Downstream`, `Upstream`, `Path`.

### DTO для API уже есть — `internal/data/dto/graph` (`graphdto`)

`NewGraphResponse(result)` (= `graph.json`), `NewNodeResponse`, `NewEdgeResponse`, `NewClusterResponse`, `NewTopNodeResponse`,
`NewRobustnessResponse`, `NewFeaturesResponse`; `graphdto.GID(int64) string`. Для `/v1/graph`, `/v1/top`, `/v1/clusters`
можно отдавать их напрямую. Не хватает только `NodeCardResponse` с соседями — сделать по образцу.

### LLM-слой — работает через HTTP (`internal/transport/http/v1/assistant`)

| Эндпоинт | Ответ | Без ключа |
|---|---|---|
| `GET /v1/assistant/status` | `{enabled, model}` | `enabled:false` — UI прячет поле вопроса |
| `POST /v1/assistant {"question"}` | `{answer, gids[], steps, cached}` | 503 `llm_disabled` |
| `GET /v1/nodes/{gid}/card` | `{gid, text, by_llm}` | шаблон, `by_llm:false` |

Ответ ассистента 8–12 с, карточка ~9 с, всё кэшируется. Сервис: `internal/services/assistant` (`Ask`, `Card`),
гипотезы: `internal/services/hypotheses`, клиент: `pkg/llm`. CLI: `go run ./cmd/ask --q "..."`, `--card <gid>`, `--hypotheses`.

### Документация

`docs/methodology.md` (метрики, правила ролей с порогами, кластеры, приоритет, паттерны, LLM, ограничения, масштабирование),
`docs/demo.md` (сценарий, разбор пяти узлов, вопросы ассистенту). В README дать ссылки, не дублировать.

## Ожидается от Дамира

1. `internal/services/graph`: `Service` с зависимостью `Result *models.AnalysisResult` (не грузить данные самому),
   методы `Graph(filter)`, `Node(gid)`, `Ego(gid, depth)`, `Top(n)`, `Clusters()`, `Search(prefix)`; регистрация в `ModuleServices`.
2. Хендлеры `internal/transport/http/v1/graph/{core.go,handlers.go}`: `GET /graph`, `/nodes/{gid}`, `/nodes/{gid}/ego`,
   `/top`, `/clusters`, `/search`; роуты в `router.go` (`RegisterV1Graph`), регистрация в `app/handlers.go`.
   Роут `/nodes/{gid}/card` уже занят ассистентом — не дублировать.
3. UI (`web/`): цвет по `role`, размер по `priority`, обводка `is_seed`, пунктир `truncated`; карточка с evidence и соседями;
   кнопка «Справка» → `/card`; поле вопроса → `/assistant` (показывать только при `status.enabled`), gid из ответа подсветить;
   панель «Устойчивость» из `graph.json.robustness`; гипотезы кластеров — уже человеческий текст.
4. `cmd/check` на текущих `out/` должен проходить (2248 строк, роли из словаря, evidence ≤ 200, cluster_id у всех, top 30 по убыванию).
5. README и Makefile: `make pipeline` без `--llm` (жюри без ключа), раздел про LLM (опционально, `OPENAI_API_KEY`, модель `gpt-5.1`,
   без ключа те же тексты из кэша), ссылки на `docs/`. В `.dockerignore` не исключать `out/llm_cache.json` и `data/`.
