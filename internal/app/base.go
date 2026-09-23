// Package app — сборка приложения на Uber FX: конфиг, логгер, валидатор, сервисы, HTTP.
// Модули подключаются в cmd/web (сервер) и cmd/pipeline, cmd/ask (CLI без HTTP).
package app

import (
	"context"
	"reflect"
	"strings"

	"hackaton/internal/config"

	"github.com/go-playground/validator/v10"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func ModuleBase() fx.Option {
	return fx.Options(
		fx.Provide(
			config.New,
			newLogger,
			newValidator,
		),
		// события FX (provide/invoke/start) — в тот же zap, на уровне debug
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			l := &fxevent.ZapLogger{Logger: log.Named("fx")}
			l.UseLogLevel(zapcore.DebugLevel)
			return l
		}),
	)
}

func newLogger(lc fx.Lifecycle, cfg *config.Config) (*zap.Logger, error) {
	zc := zap.NewProductionConfig()
	if cfg.App.IsLocal() {
		zc = zap.NewDevelopmentConfig()
		zc.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	level, err := zap.ParseAtomicLevel(cfg.App.LogLevel)
	if err != nil {
		return nil, err
	}
	zc.Level = level
	zc.DisableStacktrace = true // иначе dev-конфиг цепляет стектрейс к каждому warn (любой 4xx)

	log, err := zc.Build()
	if err != nil {
		return nil, err
	}
	log = log.With(zap.String("env", cfg.App.Environment))

	lc.Append(fx.Hook{OnStop: func(context.Context) error {
		_ = log.Sync() // на stderr/tty Sync всегда возвращает ошибку — игнорируем
		return nil
	}})

	return log, nil
}

// newValidator — имена полей в ошибках берутся из json-тегов (для 422 {field}).
func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.Split(fld.Tag.Get("json"), ",")[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return v
}
