LOCALDB   := postgres://hack:hack@localhost:5442/hackaton?sslmode=disable
GOOSE_IMG := kukymbr/goose-docker:3.24.3
SQLC_IMG  := sqlc/sqlc:1.30.0
SWAG      := go run github.com/swaggo/swag/cmd/swag@v1.16.5
DOCKER_U  := --user $(shell id -u):$(shell id -g)

.DEFAULT_GOAL := help
.PHONY: help env run build test fmt vet up down logs db-run db-reset db-psql \
        goose-create goose-migrate goose-status goose-down goose-reset sql swag gen

help: ## список команд
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# ── приложение ─────────────────────────────────────────────
env: ## создать .env из .env.example, если его нет
	@test -f .env || (cp .env.example .env && echo "created .env")

run: env ## запустить API локально (:8080, нужна БД: make db-run)
	go run ./cmd/web

build: ## собрать бинарь в bin/api
	CGO_ENABLED=0 go build -trimpath -o bin/api ./cmd/web

test: ## go test
	go test ./...

fmt: ## gofmt + go vet
	gofmt -l -w .
	go vet ./...

# ── docker compose ─────────────────────────────────────────
up: ## поднять БД + API в docker (для фронтендеров)
	docker compose --profile app up -d --build

down: ## остановить всё
	docker compose --profile app down

logs: ## логи API в docker
	docker compose logs -f api

db-run: ## поднять только Postgres (:5442)
	docker compose up -d postgres

db-reset: ## снести данные БД и поднять заново
	docker compose down -v postgres
	docker compose up -d postgres

db-psql: ## psql в БД
	docker compose exec postgres psql -U hack hackaton

# ── миграции (goose) ───────────────────────────────────────
# Миграции также применяются автоматически при старте API (postgres.auto_migrate).
goose-create: ## новая миграция (спросит имя)
	@read -p "Enter migration name: " name; \
	docker run --rm $(DOCKER_U) -v $(PWD)/migrations/postgres:/migrations \
		-e GOOSE_COMMAND="create" -e GOOSE_COMMAND_ARG="$$name sql" $(GOOSE_IMG)

goose-migrate: ## goose up
	docker run --rm $(DOCKER_U) -v $(PWD)/migrations/postgres:/migrations --network host \
		-e GOOSE_DRIVER=postgres -e GOOSE_DBSTRING=$(LOCALDB) -e GOOSE_COMMAND="up" $(GOOSE_IMG)

goose-status: ## goose status
	docker run --rm $(DOCKER_U) -v $(PWD)/migrations/postgres:/migrations --network host \
		-e GOOSE_DRIVER=postgres -e GOOSE_DBSTRING=$(LOCALDB) -e GOOSE_COMMAND="status" $(GOOSE_IMG)

goose-down: ## откатить последнюю миграцию
	docker run --rm $(DOCKER_U) -v $(PWD)/migrations/postgres:/migrations --network host \
		-e GOOSE_DRIVER=postgres -e GOOSE_DBSTRING=$(LOCALDB) -e GOOSE_COMMAND="down" $(GOOSE_IMG)

goose-reset: ## откатить все миграции
	docker run --rm $(DOCKER_U) -v $(PWD)/migrations/postgres:/migrations --network host \
		-e GOOSE_DRIVER=postgres -e GOOSE_DBSTRING=$(LOCALDB) -e GOOSE_COMMAND="reset" $(GOOSE_IMG)

# ── кодоген ────────────────────────────────────────────────
sql: ## sqlc generate → internal/repo/db
	docker run --rm $(DOCKER_U) -v $(PWD):/src -w /src $(SQLC_IMG) generate

swag: ## swagger → docs/
	rm -rf ./docs
	$(SWAG) init -g main.go -d cmd/web,internal/transport/http,internal/data/dto,pkg/httperr -o docs
	$(SWAG) fmt -d cmd/web,internal/transport/http

gen: sql swag ## sqlc + swagger
