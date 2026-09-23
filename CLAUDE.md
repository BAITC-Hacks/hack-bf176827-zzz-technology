# CLAUDE.md — «Граф денег» (HackAlem AI)

Go-сервис + React-интерфейс. Без БД: данные — три parquet в `data/`, всё считается в памяти при старте.
Для жюри и запуска — `README.md` (ru/kz/en). Методология — `docs/methodology.md`, демо — `docs/demo.md`.
Внутренние заметки — `.agent/*.md`.

## Слои

```
cmd/{pipeline,ask,web,check}      точки входа: fx.New(app.Module*) + fx.Invoke; check — валидатор CSV без fx
internal/app/                     только DI: ModuleBase, ModuleRepositories, ModuleServices, ModuleWebServer, ModuleV1Handlers, ModuleStatic
internal/config/                  Config{App, LLM}; секреты (OPENAI_API_KEY) только из env, в структуре `mapstructure:"-"`
internal/data/models/             доменные модели без json-тегов (Dataset, AnalysisResult, Features, Role, паттерны)
internal/data/graph/              граф переводов (gonum) и Index рёбер для соседей/путей
internal/data/dto/{graph,assistant}/  JSON-модели API с конструкторами NewXxxResponse(model); gid всегда строкой
internal/repo/dataset/            загрузка parquet → models.Dataset, Validate()
internal/services/analysis/       метрики, роли, кластеры, приоритет, паттерны, дробление сумм (детерминированно)
internal/services/{pipeline,export,hypotheses,assistant,graph}/  оркестрация, CSV/JSON, LLM-гипотезы, ассистент, доступ для API
internal/transport/http/v1/<d>/   core.go (Handler, handler, HandlerParams, NewHandler) + handlers.go; router.go регистрирует
internal/transport/http/mixins/   ParseBody, ParamInt64
pkg/llm                           клиент OpenAI Responses API + файловый кэш (без доменных зависимостей)
pkg/httperr                       формат ошибок {"errors":[{status,msg,field}]} + fiber ErrorHandler
frontend/                         React 19 + Vite + cytoscape; src/lib (чистая логика), components, styles (токены дизайн-системы)
web/                              встроенный резервный просмотрщик, если frontend/dist не собран
tests/unit/<domain>/              black-box тесты на реальных данных (детерминизм, требования ТЗ)
out/                              выгрузки и llm_cache.json — в репо, это артефакты сдачи
```

API: `/v1/graph`, `/v1/nodes/{gid}`, `/v1/nodes/{gid}/ego`, `/v1/top`, `/v1/clusters`, `/v1/seeds`, `/v1/robustness`,
`/v1/search`, `/v1/assistant/status`, `POST /v1/assistant`, `/v1/nodes/{gid}/card`, `/graph.json`. Актуальный список — `v1/router.go` и swagger.

## Правила

- Логика — в `internal/services`; хендлеры только парсят запрос, мапят ошибки сервиса и отдают DTO.
- Сервис: `type Service interface` + приватная `service` + `ServiceParams{services.FxBaseParams; ...}` + `NewService(params) Service`;
  логгер `params.Logger.Named("<домен>_service")`; sentinel-ошибки в `errors.go`, типы параметров в `types.go`.
- Регистрация: `fx.Annotate(x.NewService, fx.As(new(x.Service)))` в `internal/app/internal.go`; хендлеры — в `handlers.go`, роуты — в `v1/router.go`.
- `*models.AnalysisResult` провайдится один раз на процесс (`newAnalysisResult`); сервисы получают его через params, сами данные не грузят.
- Модели `models.*` наружу не отдаём — маппинг в `internal/data/dto`. Модели без json-тегов.
- Ошибки в хендлерах — `pkg/httperr` (`httperr.NotFound(code, msg)` и т.п.), сервисы возвращают sentinel-ошибки, хендлер мапит через `errors.Is`.
- Каждому хендлеру — swagger-аннотации; `@Router` БЕЗ `/v1`. После правок `make swag` (цель удаляет только сгенерированные файлы в `docs/`).
- Именование: `GID` (не Gid), `Payer/Payee` вместо src/dst, receiver'ы одной буквой, приватная реализация = имя интерфейса в нижнем регистре.
- Детерминизм: пайплайн должен давать байт-в-байт одинаковые CSV; всё, что обходит map, сортировать по GID; float из gonum округлять.
- LLM не назначает роли и приоритеты — только формулирует текст по посчитанным фактам; всё кэшируется в `out/llm_cache.json`.
  Ключ кэша гипотез включает evidence топ-узлов: после любой правки текстов evidence — `go run ./cmd/ask --hypotheses`, иначе часть гипотез откатится к шаблонам.
- Пороги ролей и веса приоритета — константы в `services/analysis/{roles,priority}.go`; при изменении обновить таблицы в README (три языка) и `docs/methodology.md`.
- Фронт: цвета только через `var(--apx-*)`/`var(--aml-*)` из `src/styles`, cytoscape читает токены через `readTokens()`; gid никогда не сокращать в списках (на канвасе — последние 6 цифр); бизнес-логика в `src/lib`, не в компонентах.
- README ведётся на трёх языках (`README.md`, `README.en.md`, `README.kz.md`) — правки вносить во все три.
- Комментарии короткие, по-русски. Объёмное — в `.agent/*.md`, не в код. Перед коммитом: `make fmt && go build ./... && go test ./...`; фронт — `docker build --target frontend .` или `cd frontend && npm run build`.
