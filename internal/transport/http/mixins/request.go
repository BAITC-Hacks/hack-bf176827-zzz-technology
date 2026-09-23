// Package mixins — общие хелперы хендлеров.
package mixins

import (
	"strconv"

	"hackaton/pkg/httperr"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// ParseBody — JSON body → dst + валидация по тегам `validate` (ошибки → 422 через httperr.Handler).
func ParseBody(c *fiber.Ctx, v *validator.Validate, dst any) error {
	if err := c.BodyParser(dst); err != nil {
		return httperr.BadRequest("invalid_body", "Некорректное тело запроса")
	}
	return v.Struct(dst)
}

// ParamInt64 — int64 из path-параметра (gid клиента).
func ParamInt64(c *fiber.Ctx, name string) (int64, error) {
	value, err := strconv.ParseInt(c.Params(name), 10, 64)
	if err != nil {
		return 0, httperr.BadRequest("invalid_id", "Некорректный идентификатор").WithField(name)
	}
	return value, nil
}
