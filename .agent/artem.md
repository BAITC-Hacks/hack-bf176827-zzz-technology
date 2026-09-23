# План Артёма

Архитектура и формулы — `.agent/plan.md`, разделение с Дамиром и контракт — `.agent/team.md`.
Здесь — только мои шаги по порядку, с LLM-слоем на OpenAI (Codex-токены).

Правило по LLM из ТЗ: роли и evidence считаются правилами, LLM их **не назначает**. LLM только
формулирует текст по уже посчитанным фактам и отвечает на вопросы по графу через инструменты.
Без ключа всё работает на шаблонах. Результаты LLM кэшируются в репо, чтобы жюри воспроизводило
без сети и без ключа.

---

## A0. Фундамент в `main` (0:00–0:25)

1. `git add -A && git commit -m "skeleton"`.
2. Выпилить Postgres и items: `internal/app/db.go`, `internal/repo/`, `internal/services/items/`,
   `internal/data/dto/items.go`, `internal/transport/http/v1/items/`, `migrations/`, `sqlc.yaml`;
   правки в `internal/app/{internal,handlers}.go`, `cmd/web/main.go`, `internal/config/config.go`
   (убрать `PostgresConfig`, добавить `App.DataDir string` с дефолтом `data`), `config/*.yaml`.
   `go build ./...` зелёный.
3. `go get github.com/parquet-go/parquet-go gonum.org/v1/gonum`.
4. `internal/data/parquet/loader.go`: `Edge`, `Node`, `Tx` (тег `date,date` у даты), `Dataset`,
   `Load(dir)`, `SanityCheck()`.
5. `internal/graph/graph.go`: `Build(ds)` → `*simple.WeightedDirectedGraph` (node id = gid),
   `Out/In map[int64][]Edge`, `TxOut/TxIn map[int64][]Tx`.
6. `internal/analysis/result.go` — контракт из team.md (не менять после коммита).
   `analysis.go` — `Run()` с заглушкой (все `peripheral`, cluster = компонента, priority = in_kzt норм.).
7. `internal/analysis/export.go`: три CSV + `graph.json` (gid строкой).
8. `cmd/pipeline/main.go`, `data/*.parquet` в репо. Прогон: 2248 строк, 3 файла.
9. Коммит `foundation`, пуш `main`, сказать Дамиру. `git checkout -b feat/analysis`.

## A1. Метрики (0:25–0:55) — `internal/analysis/metrics.go`

Все фичи из plan.md. В лог — квантили каждой фичи, по ним ставлю пороги.
Особое: `n_seed_upstream` (BFS от seed по исходящим ≤4), `verified_sink`, `truncated`,
`fast_forward_share` по датам транзакций, `max_same_day_payers`.

## A2. Кластеры (0:55–1:15) — `clusters.go`

`community.Modularize(graph.Undirect{G: g}, 1.0, rand.NewPCG(42, 0))`, изолированным — свои id,
`n_clusters_in` в фичи, `ClusterResult` со статистикой. `hypothesis` пока шаблонная
(состав ролей + суммы), LLM-версия в A5.

## A3. Роли и evidence (1:15–1:55) — `roles.go`

Скоры шести ролей по таблице plan.md, argmax, порядок при равенстве. Запреты: truncated ≠ terminal,
seed ≠ transit, truncated consolidator ×0.7. Evidence — русские шаблоны с числами, ≤200.
Спот-чек пяти узлов (in_deg 24, out_deg 116, pass_through≈1, depth=4, seed без рёбер).

## A4. Приоритет и топ (1:55–2:15) — `priority.go`

Перцентили, веса из plan.md, seed ×0.5, truncated ×0.7, `why`, top 30.
`make pipeline && make check` зелёный. Коммит. Ребейз на `main` после мержа Дамира, мерж `feat/analysis`.
**Контрольная точка 2:15: must-have 1, 2, 4 закрыты.**

## A5. LLM-слой на OpenAI (2:15–3:15) — `internal/llm/`, `internal/services/assistant/`

Проверить доступ и модель до кода:

```bash
curl -s https://api.openai.com/v1/models -H "Authorization: Bearer $OPENAI_API_KEY" | jq -r '.data[].id' | grep -i codex
```

Конфиг: `OPENAI_API_KEY` (только env), `app.llm.model` в yaml (дефолт — та codex-модель, что доступна по
токену; переопределяется `APP_LLM_MODEL`), `app.llm.base_url` (на случай прокси), `app.llm.enabled` =
`key != ""`. Клиент — `github.com/openai/openai-go` (Responses API, function calling) или чистый
`net/http` на `/v1/responses`, если SDK будет сопротивляться; таймаут 30 с, 1 ретрай.

### A5.1 `internal/llm/client.go` — тонкая обёртка

`Complete(ctx, system, user string, jsonSchema any) (string, error)` со structured output;
`RunTools(ctx, system, user, tools []Tool, exec func(name, argsJSON) (string, error)) (Answer, error)` —
цикл tool-calling до 6 шагов. Все вызовы логируются (модель, токены, время).

### A5.2 Кэш — `internal/llm/cache.go`

`out/llm_cache.json`: ключ = sha256(модель + промпт), значение = ответ. Файл **коммитится**: повторный
`make pipeline` без ключа берёт тексты из кэша, CSV байт-в-байт те же. Без ключа и без кэша — шаблон.

### A5.3 Гипотезы кластеров — `analysis/clusters.go` + флаг `--llm`

Для каждого кластера с `n_nodes ≥ 5` (остальным шаблон) один вызов: на вход структурированные факты
(размер, число seed, оборот внутри/вход/выход, состав ролей, топ-5 узлов с их evidence, число циклов),
на выход JSON `{hypothesis: string ≤ 300}`. Системный промпт: «формулируй как гипотезу для проверки,
не утверждай виновность, только gid, никаких выдуманных атрибутов». Батчить по 10 кластеров в один
вызов, чтобы уложиться в минуту.

### A5.4 Карточка узла — `GET /v1/nodes/{gid}/card`

Сервис собирает факты (роль, скоры, evidence, все фичи, топ входящих/исходящих с суммами, кластер,
циклы, seed выше по цепочке) → LLM пишет справку из 4 блоков: роль и почему; потоки; связи; на что
обратить внимание и какой запрос сделать дальше. Кэш по gid. Без ключа — та же справка шаблоном.
Хендлер и DTO по образцу Дамира, чтобы он подключил кнопку в карточке UI.

### A5.5 Ассистент аналитика — `POST /v1/assistant {question}` → `{answer, gids[]}`

Инструменты для модели (все читают `*analysis.Result` в памяти, ничего не считают заново):
- `get_node(gid)` — карточка узла в JSON;
- `find_nodes(role?, cluster?, min_priority?, is_seed?, limit)` — поиск по фильтрам;
- `who_receives_from(gids[], depth≤3)` — общие получатели вниз по цепочке с суммами;
- `who_pays_to(gid, depth≤3)` — плательщики вверх;
- `path(src, dst, max_len)` — есть ли маршрут и какой;
- `cluster_info(cluster_id)`;
- `top(n, role?)`.
Системный промпт: отвечать по-русски, ссылаться на gid, все выводы — гипотезы, если данных нет — так и
сказать, не выдумывать атрибуты клиентов. Ответ возвращает и список упомянутых gid — UI подсвечивает их.
Отдать Дамиру описание эндпоинта для поля вопроса в UI.

### A5.6 Проверка

Три вопроса на демо: «кто собирает деньги с этих пяти seed?», «через кого проходит больше всего
транзита в кластере N?», «что произойдёт, если заблокировать gid X?». Ответы должны ссылаться на
реальные gid из top. Проверить, что `make pipeline` без `OPENAI_API_KEY` проходит и CSV не меняются.

## A6. Паттерны (3:15–3:55) — `analysis/patterns.go`

Циклы ≤5 (bounded DFS), повторяющиеся маршруты A→B→C с ≥2 tx на каждом ребре, устойчивость
(удалить top-5/10/20 → число компонент, доля отвалившегося оборота). В evidence, в `clusters.csv`,
в README и как инструмент `robustness(n)` для ассистента.

## A7. README и демо (3:55–4:30)

Разделы: критерии ролей (таблица порогов и формул), формула приоритета, кластеризация (неориентированная
проекция оговорена), учёт ловушек данных, LLM-слой (что делает, что не делает, кэш, работа без ключа),
ограничения, масштабирование до 1 млн узлов. `docs/demo.md`: три gid с объяснением «за минуту» и три
вопроса ассистенту.

## 4:30 — freeze

Финальный `make demo` с чистого клона, `llm_cache.json` закоммичен, тег.

---

## Если не успеваю

Порядок жертв: A6 паттерны → A5.5 ассистент (оставить A5.3 гипотезы и A5.4 карточки, они дешёвые) →
A5.4 карточки. A0–A4 и A7 не жертвуются.
