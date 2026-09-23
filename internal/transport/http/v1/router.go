// Package v1 — регистрация роутов /v1. В @Router аннотациях путь пишется БЕЗ /v1 (@BasePath добавит).
package v1

import (
	healthv1 "hackaton/internal/transport/http/v1/health"
	itemsv1 "hackaton/internal/transport/http/v1/items"

	"github.com/gofiber/fiber/v2"
)

type Router fiber.Router

func RegisterV1Health(router Router, handler healthv1.Handler) {
	router.Get("/health", handler.Health)
}

func RegisterV1Items(router Router, handler itemsv1.Handler) {
	items := router.Group("/items")
	{
		items.Get("/", handler.List)
		items.Post("/", handler.Create)
		items.Get("/:id", handler.Get)
		items.Delete("/:id", handler.Delete)
	}
}
