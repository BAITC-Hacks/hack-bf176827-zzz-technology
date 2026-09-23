package graph

import (
	"errors"
	"strconv"
	"strings"

	graphdto "hackaton/internal/data/dto/graph"
	"hackaton/internal/data/models"
	graphservice "hackaton/internal/services/graph"
	"hackaton/internal/transport/http/mixins"
	"hackaton/pkg/httperr"

	"github.com/gofiber/fiber/v2"
)

const (
	defaultTop     = 30
	maxTop         = 10000
	defaultSearch  = 10
	maxSearch      = 100
	maxGIDLength   = 19
	defaultEgoStep = 1
)

// Graph
//
//	@Summary		Граф переводов
//	@Description	Вся сеть или top-N узлов с соседями; фильтры по роли, кластеру, компоненте.
//	@Tags			graph
//	@ID				graph
//	@Produce		json
//	@Param			role		query		string	false	"Роль"
//	@Param			cluster		query		int		false	"Кластер"
//	@Param			component	query		int		false	"Компонента"
//	@Param			top			query		int		false	"Топ N и соседи (0 — все)"
//	@Success		200			{object}	graphdto.SubgraphResponse
//	@Failure		400			{object}	httperr.Response
//	@Router			/graph [get]
func (h *handler) Graph(ctx *fiber.Ctx) error {
	filter := graphservice.Filter{Role: ctx.Query("role")}
	if filter.Role != "" && !models.IsValidRole(filter.Role) {
		return invalidParam("role")
	}
	var err error
	if filter.Cluster, err = optionalInt(ctx, "cluster"); err != nil {
		return err
	}
	if filter.Component, err = optionalInt(ctx, "component"); err != nil {
		return err
	}
	if filter.TopOnly, err = queryInt(ctx, "top", 0, 0, maxTop); err != nil {
		return err
	}
	sub := h.graph.Graph(filter)
	return ctx.JSON(graphdto.NewSubgraphResponse(sub.Nodes, sub.Edges))
}

// Node
//
//	@Summary	Карточка узла и контрагенты
//	@Tags		graph
//	@ID			node
//	@Produce	json
//	@Param		gid	path		string	true	"GID клиента"
//	@Success	200	{object}	graphdto.NodeCardResponse
//	@Failure	400	{object}	httperr.Response
//	@Failure	404	{object}	httperr.Response
//	@Router		/nodes/{gid} [get]
func (h *handler) Node(ctx *fiber.Ctx) error {
	gid, err := mixins.ParamInt64(ctx, "gid")
	if err != nil {
		return err
	}
	card, err := h.graph.Node(gid)
	if err != nil {
		return h.mapError(err)
	}
	return ctx.JSON(h.nodeCardResponse(card))
}

// Ego
//
//	@Summary	Окружение узла в один или два шага
//	@Tags		graph
//	@ID			node-ego
//	@Produce	json
//	@Param		gid		path		string	true	"GID клиента"
//	@Param		depth	query		int		false	"Глубина"	default(1)	Enums(1,2)
//	@Success	200		{object}	graphdto.SubgraphResponse
//	@Failure	400		{object}	httperr.Response
//	@Failure	404		{object}	httperr.Response
//	@Router		/nodes/{gid}/ego [get]
func (h *handler) Ego(ctx *fiber.Ctx) error {
	gid, err := mixins.ParamInt64(ctx, "gid")
	if err != nil {
		return err
	}
	depth, err := queryInt(ctx, "depth", defaultEgoStep, 1, 2)
	if err != nil {
		return err
	}
	sub, err := h.graph.Ego(gid, depth)
	if err != nil {
		return h.mapError(err)
	}
	return ctx.JSON(graphdto.NewSubgraphResponse(sub.Nodes, sub.Edges))
}

// Top
//
//	@Summary	Топ приоритетов
//	@Tags		graph
//	@ID			top
//	@Produce	json
//	@Param		n	query		int	false	"Число узлов"	default(30)
//	@Success	200	{array}		graphdto.TopNodeResponse
//	@Failure	400	{object}	httperr.Response
//	@Router		/top [get]
func (h *handler) Top(ctx *fiber.Ctx) error {
	n, err := queryInt(ctx, "n", defaultTop, 1, maxTop)
	if err != nil {
		return err
	}
	return ctx.JSON(graphdto.NewTopNodeResponses(h.graph.Top(n)))
}

// Clusters
//
//	@Summary	Кластеры сети
//	@Tags		graph
//	@ID			clusters
//	@Produce	json
//	@Success	200	{array}	graphdto.ClusterResponse
//	@Router		/clusters [get]
func (h *handler) Clusters(ctx *fiber.Ctx) error {
	return ctx.JSON(graphdto.NewClusterResponses(h.graph.Clusters()))
}

// Search
//
//	@Summary	Поиск по началу GID
//	@Tags		graph
//	@ID			search
//	@Produce	json
//	@Param		q		query		string	false	"Префикс GID"
//	@Param		limit	query		int		false	"Лимит"	default(10)
//	@Success	200		{array}		graphdto.NodeDetailResponse
//	@Failure	400		{object}	httperr.Response
//	@Router		/search [get]
func (h *handler) Search(ctx *fiber.Ctx) error {
	prefix := strings.TrimSpace(ctx.Query("q"))
	if len(prefix) > maxGIDLength || (prefix != "" && !isDigits(prefix)) {
		return invalidParam("q")
	}
	limit, err := queryInt(ctx, "limit", defaultSearch, 1, maxSearch)
	if err != nil {
		return err
	}
	return ctx.JSON(graphdto.NewNodeDetailResponses(h.graph.Search(prefix, limit)))
}

// Seeds
//
//	@Summary	Исходные участники и куда ушли их деньги
//	@Tags		graph
//	@ID			seeds
//	@Produce	json
//	@Success	200	{array}	graphdto.SeedResponse
//	@Router		/seeds [get]
func (h *handler) Seeds(ctx *fiber.Ctx) error {
	seeds := h.graph.Seeds()
	resp := make([]graphdto.SeedResponse, 0, len(seeds))
	for _, seed := range seeds {
		f := seed.Node.Features
		resp = append(resp, graphdto.SeedResponse{Gid: graphdto.GID(seed.Node.GID), Role: string(seed.Node.Role), Cluster: seed.Node.ClusterID,
			OutKZT: f.OutKZT, OutDeg: f.OutDegree, Priority: seed.Node.PriorityScore,
			Next: graphdto.NewNeighborResponses(seed.Next, func(e models.Edge) int64 { return e.Payee }, h.roleOf)})
	}
	return ctx.JSON(resp)
}

// Robustness
//
//	@Summary	Устойчивость сети при изъятии top-N узлов
//	@Tags		graph
//	@ID			robustness
//	@Produce	json
//	@Success	200	{array}	graphdto.RobustnessResponse
//	@Router		/robustness [get]
func (h *handler) Robustness(ctx *fiber.Ctx) error {
	steps := h.graph.Robustness()
	resp := make([]graphdto.RobustnessResponse, 0, len(steps))
	for _, step := range steps {
		resp = append(resp, graphdto.NewRobustnessResponse(step))
	}
	return ctx.JSON(resp)
}

func (h *handler) roleOf(gid int64) (string, bool) {
	node, ok := h.graph.NodeInfo(gid)
	if !ok {
		return "", false
	}
	return string(node.Role), node.Features.IsSeed
}

func (h *handler) nodeCardResponse(card *graphservice.NodeCard) graphdto.NodeCardResponse {
	resp := graphdto.NodeCardResponse{
		Node:           graphdto.NewNodeDetailResponse(card.Node),
		Incoming:       graphdto.NewNeighborResponses(card.Incoming, func(e models.Edge) int64 { return e.Payer }, h.roleOf),
		Outgoing:       graphdto.NewNeighborResponses(card.Outgoing, func(e models.Edge) int64 { return e.Payee }, h.roleOf),
		Percentiles:    card.Percentiles,
		NextCandidates: make([]graphdto.CandidateResponse, 0, len(card.NextCandidates)),
	}
	if card.NearestSeed != nil {
		resp.NearestSeed = &graphdto.NearestSeedResponse{Gid: graphdto.GID(card.NearestSeed.GID), Steps: card.NearestSeed.Steps}
	}
	for _, c := range card.NextCandidates {
		resp.NextCandidates = append(resp.NextCandidates, graphdto.CandidateResponse{Gid: graphdto.GID(c.GID), Role: string(c.Role), Priority: c.Priority,
			Hops: c.Hops, Direction: c.Direction, FlowShare: c.FlowShare, ScorePct: c.ScorePct})
	}
	return resp
}

func (h *handler) mapError(err error) error {
	switch {
	case errors.Is(err, graphservice.ErrNodeNotFound):
		return httperr.NotFound("node_not_found", "Узел не найден в выгрузке")
	case errors.Is(err, graphservice.ErrInvalidDepth):
		return httperr.BadRequest("invalid_depth", "depth должен быть 1 или 2").WithField("depth")
	default:
		return err
	}
}

func invalidParam(field string) error {
	return httperr.BadRequest("invalid_parameter", "Некорректный параметр "+field).WithField(field)
}

// queryInt — целое из query в [min, max]; отсутствие → fallback.
func queryInt(ctx *fiber.Ctx, key string, fallback, min, max int) (int, error) {
	raw := ctx.Query(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < min || value > max {
		return 0, invalidParam(key)
	}
	return value, nil
}

func optionalInt(ctx *fiber.Ctx, key string) (*int, error) {
	if ctx.Query(key) == "" {
		return nil, nil
	}
	value, err := queryInt(ctx, key, 0, 0, 1<<31-1)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
