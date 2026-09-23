package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// NewRequestLogger — access-лог через zap: метод, путь, статус, длительность; 5xx → error, 4xx → warn, остальное → info.
func NewRequestLogger(log *zap.Logger) fiber.Handler {
	log = log.Named("http")
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// ошибку сразу прогоняем через ErrorHandler, чтобы в логе был итоговый статус
		if err := c.Next(); err != nil {
			if herr := c.App().ErrorHandler(c, err); herr != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		status := c.Response().StatusCode()
		fields := []zap.Field{
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.IP()),
			zap.String("request_id", c.GetRespHeader(fiber.HeaderXRequestID)),
		}
		switch {
		case status >= 500:
			log.Error("request", fields...)
		case status >= 400:
			log.Warn("request", fields...)
		default:
			log.Info("request", fields...)
		}
		return nil
	}
}
