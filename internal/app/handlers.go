package app

import (
	v1 "hackaton/internal/transport/http/v1"
	graphv1 "hackaton/internal/transport/http/v1/graph"
	healthv1 "hackaton/internal/transport/http/v1/health"

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
			fx.Annotate(healthv1.NewHandler, fx.As(new(healthv1.Handler))),
			graphv1.NewHandler,
		),
		fx.Invoke(
			v1.RegisterV1Health,
			v1.RegisterV1Graph,
		),
	)
}

// ModuleSwagger — UI на /swagger/index.html, спека из docs/ (make swag).
func ModuleSwagger() fx.Option {
	return fx.Invoke(func(app *fiber.App) {
		app.Get("/swagger/*", swagger.WrapHandler)
	})
}
