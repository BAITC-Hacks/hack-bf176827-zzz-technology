SWAG := go run github.com/swaggo/swag/cmd/swag@v1.16.5
.DEFAULT_GOAL := help
.PHONY: help env run build test fmt vet swag pipeline check web demo demo-full docker docker-down frontend frontend-build
help: ## список команд
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
env: ## создать .env, если его нет
	@test -f .env || cp .env.example .env
run: web ## запустить веб-сервер
web: ## запустить веб-сервер
	go run ./cmd/web
pipeline: ## рассчитать аналитику и выгрузить CSV
	go run ./cmd/pipeline --data data --out out
check: ## проверить выгрузки
	go run ./cmd/check --out out
demo: ## pipeline → check → web (Go, без Node; React отдаётся, если собран в frontend/dist, иначе встроенный просмотрщик)
	$(MAKE) pipeline
	$(MAKE) check
	$(MAKE) web
demo-full: frontend-build demo ## то же, но сначала собрать React (нужны Node.js 22+ и npm)
docker: ## собрать и запустить всё в Docker: http://localhost:8080
	docker compose up --build
docker-down: ## остановить контейнер
	docker compose down
build: ## собрать pipeline, check и web
	go build -o bin/pipeline ./cmd/pipeline
	go build -o bin/web ./cmd/web
	go build -o bin/check ./cmd/check
test: ## запустить тесты
	go test ./...
fmt: ## форматирование и статическая проверка
	gofmt -w cmd internal pkg web
	go vet ./...
vet: ## статическая проверка
	go vet ./...
swag: ## обновить Swagger (только сгенерированные файлы; docs/*.md, png, svg не трогаем)
	rm -f docs/docs.go docs/swagger.json docs/swagger.yaml
	$(SWAG) init -g main.go -d cmd/web,internal/transport/http,internal/data/dto,pkg/httperr -o docs
	$(SWAG) fmt -d cmd/web,internal/transport/http

frontend: ## запустить React dev-сервер (:5173; API на :8080)
	@sh frontend/scripts/check-env.sh
	cd frontend && npm run dev
frontend-build: ## установить зависимости и собрать React
	@sh frontend/scripts/check-env.sh
	cd frontend && npm ci && npm run build
