# CLAUDE.md — «Граф денег» (HackAlem AI)

Go-сервис: Fiber v2, Uber FX, zap, Viper, swag. Без БД: данные — три parquet в `data/`,
всё считается в памяти. Архитектура и решения — `.agent/plan.md`, разделение работ — `.agent/team.md`,
методология — `docs/methodology.md`.

## Слои

```
cmd/{pipeline,ask,web}            точки входа: fx.New(app.Module*) + fx.Invoke
internal/app/                     только DI: ModuleBase, ModuleRepositories, ModuleServices, ModuleWebServer, ModuleV1Handlers
internal/config/                  Config{App, LLM}; секреты (OPENAI_API_KEY) только из env, в структуре `mapstructure:"-"`
internal/data/models/             доменные модели без json-тегов (Dataset, AnalysisResult, Features, Role, паттерны)
internal/data/graph/              граф переводов (gonum) и Index рёбер для соседей/путей
internal/data/dto/<domain>/       JSON-модели API с конструкторами NewXxxResponse(model); gid всегда строкой
internal/repo/dataset/            загрузка parquet → models.Dataset, Validate()
internal/services/<domain>/       логика: analysis, export, hypotheses, pipeline, assistant
internal/transport/http/v1/<d>/   core.go (Handler, handler, HandlerParams, NewHandler) + handlers.go
internal/transport/http/mixins/   ParseBody, ParamInt64
pkg/llm                           клиент OpenAI Responses API + файловый кэш (без доменных зависимостей)
pkg/httperr                       формат ошибок {"errors":[{status,msg,field}]} + fiber ErrorHandler
tests/unit/<domain>/              black-box тесты на реальных данных
```

## Правила

- Логика — в `internal/services`; хендлеры только парсят запрос, мапят ошибки сервиса и отдают DTO.
- Сервис: `type Service interface` + приватная `service` + `ServiceParams{services.FxBaseParams; ...}` + `NewService(params) Service`;
  логгер `params.Logger.Named("<домен>_service")`; sentinel-ошибки в `errors.go`, типы параметров в `types.go`.
- Регистрация: `fx.Annotate(x.NewService, fx.As(new(x.Service)))` в `internal/app/internal.go`; хендлеры — в `handlers.go`, роуты — в `v1/router.go`.
- Модели `models.*` наружу не отдаём — маппинг в `internal/data/dto`. Модели без json-тегов.
- Ошибки в хендлерах — `pkg/httperr` (`httperr.NotFound(code, msg)` и т.п.), сервисы возвращают sentinel-ошибки, хендлер мапит через `errors.Is`.
- Каждому хендлеру — swagger-аннотации; `@Router` БЕЗ `/v1`. После правок `make swag`. `docs/` не править руками.
- Именование: `GID` (не Gid), `Payer/Payee` вместо src/dst, receiver'ы одной буквой, приватная реализация = имя интерфейса в нижнем регистре.
- Детерминизм: пайплайн должен давать байт-в-байт одинаковые CSV; всё, что обходит map, сортировать по GID.
- LLM не назначает роли и приоритеты — только формулирует текст по посчитанным фактам; всё кэшируется в `out/llm_cache.json` (в репо).
- Комментарии короткие, по-русски. Объёмное — в `.agent/*.md`, не в код. Перед коммитом: `make fmt && go build ./... && go test ./...`.
