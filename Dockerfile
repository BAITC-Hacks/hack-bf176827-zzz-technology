# syntax=docker/dockerfile:1
# Один образ: React-интерфейс собирается на node-стадии, Go-бинари — на go-стадии,
# в рантайме web отдаёт API и статику на :8080. Node на машине не нужен.

FROM node:24-alpine AS frontend
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/pipeline ./cmd/pipeline \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/check ./cmd/check \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/web ./cmd/web

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/ /app/bin/
COPY --from=frontend /app/dist /app/frontend/dist
COPY config ./config
COPY data ./data
COPY out/llm_cache.json ./out/llm_cache.json
COPY cmd/web/entrypoint.sh /app/entrypoint.sh
ENV APP_ENVIRONMENT=prod
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=20s --retries=12 \
  CMD wget -q --spider http://127.0.0.1:8080/v1/health || exit 1
ENTRYPOINT ["/bin/sh", "/app/entrypoint.sh"]
