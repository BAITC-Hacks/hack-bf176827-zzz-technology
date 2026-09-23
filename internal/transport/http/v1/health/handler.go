package health

import (
	"context"
	"time"

	"hackaton/internal/data/dto"
	"hackaton/pkg/httperr"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler interface {
	Health(ctx *fiber.Ctx) error
}

type handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) Handler {
	return &handler{pool: pool}
}

// Health
//
//	@Summary	Проверка живости (API + БД)
//	@Tags		system
//	@ID			health
//	@Produce	json
//	@Success	200	{object}	dto.HealthResponse
//	@Failure	503	{object}	httperr.Response
//	@Router		/health [get]
func (h *handler) Health(ctx *fiber.Ctx) error {
	c, cancel := context.WithTimeout(ctx.UserContext(), 2*time.Second)
	defer cancel()
	if err := h.pool.Ping(c); err != nil {
		return httperr.New(fiber.StatusServiceUnavailable, "db_unavailable", "База данных недоступна")
	}
	return ctx.JSON(dto.HealthResponse{Status: "ok"})
}
