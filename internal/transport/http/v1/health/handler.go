package health

import (
	"hackaton/internal/data/dto"

	"github.com/gofiber/fiber/v2"
)

type Handler interface {
	Health(ctx *fiber.Ctx) error
}

type handler struct{}

func NewHandler() Handler {
	return &handler{}
}

// Health
//
//	@Summary	Проверка живости
//	@Tags		system
//	@ID			health
//	@Produce	json
//	@Success	200	{object}	dto.HealthResponse
//	@Router		/health [get]
func (h *handler) Health(ctx *fiber.Ctx) error {
	return ctx.JSON(dto.HealthResponse{Status: "ok"})
}
