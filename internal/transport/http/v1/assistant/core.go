// Package assistant — HTTP-хендлеры LLM-ассистента: статус, вопрос по графу, справка по узлу.
package assistant

import (
	assistantservice "hackaton/internal/services/assistant"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Handler interface {
	Status(ctx *fiber.Ctx) error
	Ask(ctx *fiber.Ctx) error
	Card(ctx *fiber.Ctx) error
}

type handler struct {
	logger    *zap.Logger
	validate  *validator.Validate
	assistant assistantservice.Service
}

type HandlerParams struct {
	fx.In
	Logger    *zap.Logger
	Validate  *validator.Validate
	Assistant assistantservice.Service
}

func NewHandler(params HandlerParams) Handler {
	return &handler{logger: params.Logger.Named("assistant_handler"), validate: params.Validate, assistant: params.Assistant}
}
