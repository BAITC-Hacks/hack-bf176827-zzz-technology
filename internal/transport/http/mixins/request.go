// Package mixins — общие хелперы хендлеров.
package mixins

import (
	"hackaton/pkg/httperr"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ParseBody — JSON body → dst + валидация по тегам `validate` (ошибки → 422 через httperr.Handler).
func ParseBody(c *fiber.Ctx, v *validator.Validate, dst any) error {
	if err := c.BodyParser(dst); err != nil {
		return httperr.BadRequest("invalid_body", "Некорректное тело запроса")
	}
	return v.Struct(dst)
}

// ParamUUID — uuid из path-параметра.
func ParamUUID(c *fiber.Ctx, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil {
		return uuid.Nil, httperr.BadRequest("invalid_id", "Некорректный идентификатор").WithField(name)
	}
	return id, nil
}
