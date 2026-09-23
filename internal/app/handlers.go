package app

import (
	v1 "hackaton/internal/transport/http/v1"
	assistantv1handler "hackaton/internal/transport/http/v1/assistant"
	graphv1handler "hackaton/internal/transport/http/v1/graph"
	healthv1handler "hackaton/internal/transport/http/v1/health"

	_ "hackaton/docs"

	"github.com/gofiber/fiber/v2"
	swagger "github.com/swaggo/fiber-swagger"
	"go.uber.org/fx"
)

func ModuleV1Handlers() fx.Option {
	return fx.Options(
		fx.Provide(func(app *fiber.App) v1.Router {
			return app.Group("/v1")
		}),
		fx.Provide(
			fx.Annotate(healthv1handler.NewHandler, fx.As(new(healthv1handler.Handler))),
			fx.Annotate(graphv1handler.NewHandler, fx.As(new(graphv1handler.Handler))),
			fx.Annotate(assistantv1handler.NewHandler, fx.As(new(assistantv1handler.Handler))),
		),
		fx.Invoke(
			v1.RegisterV1Health,
			v1.RegisterV1Graph,
			v1.RegisterV1Assistant,
		),
	)
}

// ModuleSwagger — UI на /swagger/index.html, спека из docs/ (make swag).
func ModuleSwagger() fx.Option {
	return fx.Invoke(func(app *fiber.App) {
		app.Get("/swagger/*", swagger.WrapHandler)
	})
}
