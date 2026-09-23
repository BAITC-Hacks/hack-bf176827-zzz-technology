package app

import (
	"hackaton/internal/repo/db"
	"hackaton/internal/services/items"

	"go.uber.org/fx"
)

func ModuleRepositories() fx.Option {
	return fx.Provide(
		fx.Annotate(db.New, fx.As(new(db.Querier))),
	)
}

func ModuleServices() fx.Option {
	return fx.Provide(
		fx.Annotate(items.NewService, fx.As(new(items.Service))),
	)
}
