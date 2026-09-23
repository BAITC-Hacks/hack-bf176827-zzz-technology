package app

import (
	"go.uber.org/fx"
	"hackaton/internal/services/graph"
)

// ModuleServices — сервисы (регистрируются по мере появления).
func ModuleServices() fx.Option {
	return fx.Provide(graph.NewService)
}
