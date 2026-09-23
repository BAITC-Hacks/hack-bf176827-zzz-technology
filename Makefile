# Go определяет свой SDK по исполняемому файлу; не наследуем GOROOT из Windows/IDE.
GO_CMD := env -u GOROOT go
SWAG := $(GO_CMD) run github.com/swaggo/swag/cmd/swag@v1.16.5
.DEFAULT_GOAL := help
.PHONY: help env run build test fmt vet swag pipeline check web demo frontend frontend-build go-check
help: ## список команд
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
env: ## создать .env, если его нет
	@test -f .env || cp .env.example .env
run: web ## запустить веб-сервер
web: ## запустить веб-сервер
	$(GO_CMD) run ./cmd/web
pipeline: ## рассчитать аналитику и выгрузить CSV
	$(GO_CMD) run ./cmd/pipeline --data data --out out
check: ## проверить выгрузки
	$(GO_CMD) run ./cmd/check --out out
demo: ## собрать React и последовательно выполнить pipeline, check, web
	$(MAKE) go-check
	$(MAKE) frontend-build
	$(MAKE) pipeline
	$(MAKE) check
	$(MAKE) web
build: frontend-build ## собрать pipeline, check и web
	$(GO_CMD) build -o bin/pipeline ./cmd/pipeline
	$(GO_CMD) build -o bin/web ./cmd/web
	$(GO_CMD) build -o bin/check ./cmd/check
test: ## запустить тесты
	$(GO_CMD) test ./...
fmt: ## форматирование и статическая проверка
	gofmt -w cmd internal pkg web
	$(GO_CMD) vet ./...
vet: ## статическая проверка
	$(GO_CMD) vet ./...
swag: ## обновить Swagger, сохранив остальные файлы docs
	$(SWAG) init -g main.go -d cmd/web,internal/transport/http,internal/data/dto,internal/analysis,pkg/httperr -o docs
	$(SWAG) fmt -d cmd/web,internal/transport/http

frontend: ## запустить React dev-сервер (:5173; API на :8080)
	@sh frontend/scripts/check-env.sh
	cd frontend && npm run dev
frontend-build: ## установить зависимости и собрать React
	@sh frontend/scripts/check-env.sh
	cd frontend && npm ci && npm run build

go-check: ## проверить доступность Go до сборки фронтенда
	@$(GO_CMD) version
