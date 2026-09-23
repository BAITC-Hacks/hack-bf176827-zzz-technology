// Package graph — HTTP-хендлеры просмотра сети: граф с фильтрами, карточка узла, окружение, топ, кластеры, поиск.
package graph

import (
	graphservice "hackaton/internal/services/graph"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Handler interface {
	Graph(ctx *fiber.Ctx) error
	Node(ctx *fiber.Ctx) error
	Ego(ctx *fiber.Ctx) error
	Top(ctx *fiber.Ctx) error
	Clusters(ctx *fiber.Ctx) error
	Search(ctx *fiber.Ctx) error
}

type handler struct {
	logger *zap.Logger
	graph  graphservice.Service
}

type HandlerParams struct {
	fx.In
	Logger *zap.Logger
	Graph  graphservice.Service
}

func NewHandler(params HandlerParams) Handler {
	return &handler{logger: params.Logger.Named("graph_handler"), graph: params.Graph}
}
