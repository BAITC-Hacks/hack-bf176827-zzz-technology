// Package v1 — регистрация роутов /v1. В @Router аннотациях путь пишется БЕЗ /v1 (@BasePath добавит).
package v1

import (
	graphv1 "hackaton/internal/transport/http/v1/graph"
	healthv1 "hackaton/internal/transport/http/v1/health"

	"github.com/gofiber/fiber/v2"
)

type Router fiber.Router

func RegisterV1Health(router Router, handler healthv1.Handler) {
	router.Get("/health", handler.Health)
}

func RegisterV1Graph(router Router, handler *graphv1.Handler) {
	router.Get("/graph", handler.Graph)
	router.Get("/nodes/:gid", handler.Node)
	router.Get("/nodes/:gid/ego", handler.Ego)
	router.Get("/top", handler.Top)
	router.Get("/clusters", handler.Clusters)
	router.Get("/search", handler.Search)
}
