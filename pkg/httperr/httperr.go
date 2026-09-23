// Package httperr — единый формат ошибок API: {"errors":[{"status","msg","field"}]}.
package httperr

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Error struct {
	HTTPStatus int    `json:"-"`
	Status     string `json:"status" example:"not_found"` // машинный код для фронта
	Msg        string `json:"msg" example:"Не найдено"`   // человекочитаемое сообщение
	Field      string `json:"field,omitempty"`            // поле запроса, если ошибка про него
}

type Response struct {
	Errors []Error `json:"errors"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d %s: %s", e.HTTPStatus, e.Status, e.Msg)
}

func (e *Error) WithField(field string) *Error {
	c := *e
	c.Field = field
	return &c
}

func New(httpStatus int, status, msg string) *Error {
	return &Error{HTTPStatus: httpStatus, Status: status, Msg: msg}
}

func BadRequest(status, msg string) *Error   { return New(http.StatusBadRequest, status, msg) }
func Unauthorized(status, msg string) *Error { return New(http.StatusUnauthorized, status, msg) }
func Forbidden(status, msg string) *Error    { return New(http.StatusForbidden, status, msg) }
func NotFound(status, msg string) *Error     { return New(http.StatusNotFound, status, msg) }
func Conflict(status, msg string) *Error     { return New(http.StatusConflict, status, msg) }

// Handler — fiber.ErrorHandler: *Error → как есть, ошибки валидатора → 422,
// *fiber.Error → его код, всё остальное → 500 с логом.
func Handler(log *zap.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var (
			he *Error
			ve validator.ValidationErrors
			fe *fiber.Error
		)
		switch {
		case errors.As(err, &he):
			return c.Status(he.HTTPStatus).JSON(Response{Errors: []Error{*he}})

		case errors.As(err, &ve):
			errs := make([]Error, 0, len(ve))
			for _, f := range ve {
				msg := "failed on '" + f.Tag() + "'"
				if f.Param() != "" {
					msg += " (" + f.Param() + ")"
				}
				errs = append(errs, Error{Status: "validation_failed", Msg: msg, Field: f.Field()})
			}
			return c.Status(http.StatusUnprocessableEntity).JSON(Response{Errors: errs})

		case errors.As(err, &fe):
			return c.Status(fe.Code).JSON(Response{Errors: []Error{{Status: "http_error", Msg: fe.Message}}})

		default:
			log.Error("unhandled error",
				zap.Error(err),
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.String("request_id", c.GetRespHeader(fiber.HeaderXRequestID)))
			return c.Status(http.StatusInternalServerError).JSON(Response{
				Errors: []Error{{Status: "internal_error", Msg: "Внутренняя ошибка сервера"}},
			})
		}
	}
}
