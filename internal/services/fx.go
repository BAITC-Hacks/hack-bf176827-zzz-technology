// Package services — общее для всех сервисов: базовые fx-зависимости.
package services

import (
	"hackaton/internal/config"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// FxBaseParams — встраивается в XxxParams каждого сервиса.
type FxBaseParams struct {
	fx.In
	Logger *zap.Logger
	Config *config.Config
}
