# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# миграции и swagger вшиты в бинарь (embed / docs.go)
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/web

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/api /app/api
COPY config ./config
ENV APP_ENVIRONMENT=prod
EXPOSE 8080
ENTRYPOINT ["/app/api"]
