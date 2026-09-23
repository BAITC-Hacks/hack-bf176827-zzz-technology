package assistant

import (
	"errors"
	"strconv"

	assistantdto "hackaton/internal/data/dto/assistant"
	graphdto "hackaton/internal/data/dto/graph"
	assistantservice "hackaton/internal/services/assistant"
	"hackaton/internal/transport/http/mixins"
	"hackaton/pkg/httperr"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Status
//
//	@Summary	Доступен ли LLM-ассистент
//	@Tags		assistant
//	@ID			assistant-status
//	@Produce	json
//	@Success	200	{object}	assistantdto.StatusResponse
//	@Router		/assistant/status [get]
func (h *handler) Status(ctx *fiber.Ctx) error {
	return ctx.JSON(assistantdto.StatusResponse{Enabled: h.assistant.Enabled(), Model: h.assistant.Model()})
}

// Ask
//
//	@Summary		Вопрос по графу на естественном языке
//	@Description	Ответ строится через инструменты по результату анализа; в gids — упомянутые узлы для подсветки.
//	@Tags			assistant
//	@ID				assistant-ask
//	@Accept			json
//	@Produce		json
//	@Param			body	body		assistantdto.AskRequest	true	"Вопрос"
//	@Success		200		{object}	assistantdto.AskResponse
//	@Failure		400		{object}	httperr.Response
//	@Failure		422		{object}	httperr.Response
//	@Failure		503		{object}	httperr.Response	"LLM не настроен"
//	@Router			/assistant [post]
func (h *handler) Ask(ctx *fiber.Ctx) error {
	var req assistantdto.AskRequest
	if err := mixins.ParseBody(ctx, h.validate, &req); err != nil {
		return err
	}
	var contextGID int64
	if req.Gid != "" {
		parsed, err := strconv.ParseInt(req.Gid, 10, 64)
		if err != nil {
			return httperr.BadRequest("invalid_gid", "Некорректный gid").WithField("gid")
		}
		contextGID = parsed
	}
	route := make([]int64, 0, len(req.Route))
	for _, raw := range req.Route {
		if gid, err := strconv.ParseInt(raw, 10, 64); err == nil {
			route = append(route, gid)
		}
	}
	answer, err := h.assistant.Ask(ctx.UserContext(), req.Question, contextGID, route)
	if err != nil {
		switch {
		case errors.Is(err, assistantservice.ErrLLMDisabled):
			return httperr.New(fiber.StatusServiceUnavailable, "llm_disabled", "LLM не настроен: задайте OPENAI_API_KEY")
		case errors.Is(err, assistantservice.ErrEmptyQuestion):
			return httperr.BadRequest("empty_question", "Пустой вопрос").WithField("question")
		default:
			h.logger.Error("failed to answer question", zap.Error(err))
			return httperr.New(fiber.StatusBadGateway, "llm_failed", "Ассистент не смог ответить, попробуйте ещё раз")
		}
	}
	return ctx.JSON(assistantdto.AskResponse{Answer: answer.Text, Gids: answer.GIDs, Steps: answer.Steps, Cached: answer.Cached})
}

// Card
//
//	@Summary		Справка по узлу
//	@Description	Четыре блока: роль и почему, потоки, связи, на что обратить внимание. Без ключа — шаблон (by_llm=false).
//	@Tags			assistant
//	@ID				node-card
//	@Produce		json
//	@Param			gid	path		string	true	"GID клиента"
//	@Success		200	{object}	assistantdto.CardResponse
//	@Failure		400	{object}	httperr.Response
//	@Failure		404	{object}	httperr.Response
//	@Router			/nodes/{gid}/card [get]
func (h *handler) Card(ctx *fiber.Ctx) error {
	gid, err := mixins.ParamInt64(ctx, "gid")
	if err != nil {
		return err
	}
	card, err := h.assistant.Card(ctx.UserContext(), gid)
	if err != nil {
		if errors.Is(err, assistantservice.ErrNodeNotFound) {
			return httperr.NotFound("node_not_found", "Узел не найден в выгрузке")
		}
		h.logger.Error("failed to build card", zap.Error(err), zap.Int64("gid", gid))
		return err
	}
	return ctx.JSON(assistantdto.CardResponse{Gid: graphdto.GID(gid), Text: card.Text, ByLLM: card.ByLLM})
}
