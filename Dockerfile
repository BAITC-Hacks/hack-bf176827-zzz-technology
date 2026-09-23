# syntax=docker/dockerfile:1
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
COPY config ./config
COPY data ./data
COPY cmd/web/entrypoint.sh /app/entrypoint.sh
ENV APP_ENVIRONMENT=prod
EXPOSE 8080
ENTRYPOINT ["/bin/sh", "/app/entrypoint.sh"]
