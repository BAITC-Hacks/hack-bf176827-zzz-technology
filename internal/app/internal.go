package app

import (
	"go.uber.org/fx"
)

// ModuleServices — сервисы (регистрируются по мере появления).
func ModuleServices() fx.Option {
	return fx.Provide()
}
