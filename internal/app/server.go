package app

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"hackaton/web"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hackaton/internal/config"
	"hackaton/internal/transport/http/middleware"
	"hackaton/pkg/httperr"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func ModuleWebServer() fx.Option {
	return fx.Options(
		fx.Provide(newFiberApp),
		// порядок важен: request id → лог → recover (паника станет 500 и попадёт в лог) → cors
		fx.Invoke(func(app *fiber.App, cfg *config.Config, log *zap.Logger) {
			app.Use(
				requestid.New(),
				middleware.NewRequestLogger(log),
				recover.New(recover.Config{EnableStackTrace: cfg.App.Debug}),
				cors.New(cors.Config{
					AllowOrigins: strings.Join(cfg.App.CorsOrigins, ","),
					AllowHeaders: strings.Join(cfg.App.CorsHeaders, ","),
				}),
			)
		}),
	)
}

func newFiberApp(cfg *config.Config, log *zap.Logger) *fiber.App {
	return fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		DisableStartupMessage: true, // стартуем через zap
		ErrorHandler:          httperr.Handler(log),
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          2 * time.Minute, // ответ LLM-ассистента до 90 с
		IdleTimeout:           2 * time.Minute,
	})
}

func ModuleRunWebServer() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, app *fiber.App, cfg *config.Config, log *zap.Logger) {
		lc.Append(fx.Hook{
			OnStart: func(context.Context) error {
				addr := fmt.Sprintf("0.0.0.0:%d", cfg.App.Port)
				// Listen синхронно, чтобы занятый порт ронял старт, а не терялся в горутине
				ln, err := net.Listen("tcp", addr)
				if err != nil {
					return fmt.Errorf("listen %s: %w", addr, err)
				}
				go func() {
					if err := app.Listener(ln); err != nil {
						log.Error("http server stopped", zap.Error(err))
					}
				}()
				log.Info("http server started",
					zap.String("addr", addr),
					zap.String("swagger", fmt.Sprintf("http://localhost:%d/swagger/index.html", cfg.App.Port)))
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return app.ShutdownWithContext(ctx)
			},
		})
	})
}

// ModuleStatic регистрируется после API и Swagger.
func ModuleStatic() fx.Option {
	return fx.Invoke(func(app *fiber.App, cfg *config.Config) {
		app.Get("/graph.json", func(c *fiber.Ctx) error { return c.SendFile(filepath.Join(cfg.App.OutDir, "graph.json")) })
		// make demo собирает React в frontend/dist; встроенный D0 остаётся резервным просмотрщиком.
		if _, err := os.Stat("frontend/dist/index.html"); err == nil {
			app.Static("/", "frontend/dist")
		} else {
			app.Use("/", filesystem.New(filesystem.Config{Root: http.FS(web.Files), Index: "index.html"}))
		}
	})
}
