package items

import (
	"hackaton/internal/data/dto"
	itemsservice "hackaton/internal/services/items"
	"hackaton/internal/transport/http/mixins"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Handler interface {
	List(ctx *fiber.Ctx) error
	Get(ctx *fiber.Ctx) error
	Create(ctx *fiber.Ctx) error
	Delete(ctx *fiber.Ctx) error
}

type handler struct {
	validate *validator.Validate
	service  itemsservice.Service
}

func NewHandler(v *validator.Validate, s itemsservice.Service) Handler {
	return &handler{validate: v, service: s}
}

// List
//
//	@Summary	Список элементов
//	@Tags		items
//	@ID			list-items
//	@Produce	json
//	@Success	200	{object}	dto.ListResponse[dto.Item]
//	@Router		/items [get]
func (h *handler) List(ctx *fiber.Ctx) error {
	items, err := h.service.List(ctx.UserContext())
	if err != nil {
		return err
	}
	return ctx.JSON(dto.ListResponse[dto.Item]{Items: dto.ItemsFromDB(items)})
}

// Get
//
//	@Summary	Элемент по id
//	@Tags		items
//	@ID			get-item
//	@Produce	json
//	@Param		id	path		string	true	"ID элемента"	format(uuid)
//	@Success	200	{object}	dto.Item
//	@Failure	400	{object}	httperr.Response
//	@Failure	404	{object}	httperr.Response
//	@Router		/items/{id} [get]
func (h *handler) Get(ctx *fiber.Ctx) error {
	id, err := mixins.ParamUUID(ctx, "id")
	if err != nil {
		return err
	}
	item, err := h.service.Get(ctx.UserContext(), id)
	if err != nil {
		return err
	}
	return ctx.JSON(dto.ItemFromDB(item))
}

// Create
//
//	@Summary	Создать элемент
//	@Tags		items
//	@ID			create-item
//	@Accept		json
//	@Produce	json
//	@Param		body	body		dto.CreateItemRequest	true	"Данные элемента"
//	@Success	201		{object}	dto.Item
//	@Failure	400		{object}	httperr.Response
//	@Failure	422		{object}	httperr.Response
//	@Router		/items [post]
func (h *handler) Create(ctx *fiber.Ctx) error {
	var req dto.CreateItemRequest
	if err := mixins.ParseBody(ctx, h.validate, &req); err != nil {
		return err
	}
	item, err := h.service.Create(ctx.UserContext(), req.Title)
	if err != nil {
		return err
	}
	return ctx.Status(fiber.StatusCreated).JSON(dto.ItemFromDB(item))
}

// Delete
//
//	@Summary	Удалить элемент
//	@Tags		items
//	@ID			delete-item
//	@Param		id	path	string	true	"ID элемента"	format(uuid)
//	@Success	204
//	@Failure	400	{object}	httperr.Response
//	@Failure	404	{object}	httperr.Response
//	@Router		/items/{id} [delete]
func (h *handler) Delete(ctx *fiber.Ctx) error {
	id, err := mixins.ParamUUID(ctx, "id")
	if err != nil {
		return err
	}
	if err := h.service.Delete(ctx.UserContext(), id); err != nil {
		return err
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
