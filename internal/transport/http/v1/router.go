// Package v1 — регистрация роутов /v1. В @Router аннотациях путь пишется БЕЗ /v1 (@BasePath добавит).
package v1

import (
	healthv1 "hackaton/internal/transport/http/v1/health"

	"github.com/gofiber/fiber/v2"
)

type Router fiber.Router

func RegisterV1Health(router Router, handler healthv1.Handler) {
	router.Get("/health", handler.Health)
}
