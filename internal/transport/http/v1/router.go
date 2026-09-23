// Package v1 — регистрация роутов /v1. В @Router аннотациях путь пишется БЕЗ /v1 (@BasePath добавит).
package v1

import (
	assistantv1handler "hackaton/internal/transport/http/v1/assistant"
	healthv1handler "hackaton/internal/transport/http/v1/health"

	"github.com/gofiber/fiber/v2"
)

type Router fiber.Router

func RegisterV1Health(router Router, handler healthv1handler.Handler) {
	router.Get("/health", handler.Health)
}

func RegisterV1Assistant(router Router, handler assistantv1handler.Handler) {
	assistant := router.Group("/assistant")
	{
		assistant.Get("/status", handler.Status)
		assistant.Post("/", handler.Ask)
	}
	router.Get("/nodes/:gid/card", handler.Card)
}
