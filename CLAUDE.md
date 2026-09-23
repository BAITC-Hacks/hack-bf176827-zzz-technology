# CLAUDE.md — hackaton API

Каркас на базе morafest-backend (Fiber v2, Uber FX, zap, Viper, pgx/v5 + sqlc, goose, swag).
Устройство, структура и workflow добавления фичи — в `README.md`.

Правила:
- Логика — в `internal/services`, хендлеры только парсят запрос и мапят ответ в DTO.
- Модели `db.*` (sqlc) наружу не отдаём — маппинг в `internal/data/dto`.
- Ошибки — `pkg/httperr` (`httperr.NotFound(...)` и т.д.), не `ctx.Status().JSON()` вручную.
- Каждому хендлеру — swagger-аннотации; `@Router` БЕЗ `/v1` (добавляет `@BasePath`). После — `make swag`.
- Сгенерированное не править: `internal/repo/db/`, `docs/`. Менять SQL → `make sql`.
- Новый fx-провайдер регистрировать в `internal/app/*.go`, модули подключены в `cmd/web/main.go`.
- Секреты только из env/.env, несекретное — `config/*.yaml`.
- Комментарии в коде — по-русски. Перед коммитом: `make fmt && go build ./...`.
