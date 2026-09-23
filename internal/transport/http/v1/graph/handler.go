// Package graph обрабатывает запросы просмотра графа.
package graph

import (
	"hackaton/internal/analysis"
	"hackaton/internal/data/dto"
	graphservice "hackaton/internal/services/graph"
	"hackaton/pkg/httperr"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service graphservice.Service }

func NewHandler(s graphservice.Service) *Handler { return &Handler{service: s} }
func invalid(field string) error {
	return httperr.BadRequest("invalid_parameter", "Некорректный параметр "+field).WithField(field)
}
func number(c *fiber.Ctx, key string, defaultValue, min, max int) (int, error) {
	v := c.Query(key)
	if v == "" {
		return defaultValue, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < min || n > max {
		return 0, invalid(key)
	}
	return n, nil
}
func optional(c *fiber.Ctx, key string) (*int, error) {
	if c.Query(key) == "" {
		return nil, nil
	}
	n, err := number(c, key, 0, 0, 1<<31-1)
	return &n, err
}
func gid(c *fiber.Ctx) (int64, error) {
	v := c.Params("gid")
	if !digits(v) {
		return 0, invalid("gid")
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, invalid("gid")
	}
	return n, nil
}
func digits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// Graph возвращает отфильтрованную сеть.
//
//	@Summary	Граф переводов
//	@Tags		graph
//	@Produce	json
//	@Param		role		query		string	false	"Роль"
//	@Param		cluster		query		int		false	"Кластер"
//	@Param		component	query		int		false	"Компонента"
//	@Param		top			query		int		false	"Топ N и соседи (0 — все)"
//	@Success	200			{object}	dto.GraphResponse
//	@Failure	400			{object}	httperr.Response
//	@Router		/graph [get]
func (h *Handler) Graph(c *fiber.Ctx) error {
	f := graphservice.Filter{Role: c.Query("role")}
	if f.Role != "" {
		ok := false
		for _, r := range analysis.Roles {
			if r == f.Role {
				ok = true
			}
		}
		if !ok {
			return invalid("role")
		}
	}
	var err error
	f.Cluster, err = optional(c, "cluster")
	if err != nil {
		return err
	}
	f.Component, err = optional(c, "component")
	if err != nil {
		return err
	}
	f.TopOnly, err = number(c, "top", 0, 0, 10000)
	if err != nil {
		return err
	}
	return c.JSON(dto.MapGraph(h.service.Graph(f)))
}

// Node возвращает карточку узла.
//
//	@Summary	Карточка узла и соседи
//	@Tags		graph
//	@Produce	json
//	@Param		gid		path		string	true	"gid (int64 строкой)"
//	@Success	200		{object}	dto.NodeCardDTO
//	@Failure	400,404	{object}	httperr.Response
//	@Router		/nodes/{gid} [get]
func (h *Handler) Node(c *fiber.Ctx) error {
	id, err := gid(c)
	if err != nil {
		return err
	}
	card, err := h.service.Node(id)
	if err != nil {
		return err
	}
	return c.JSON(dto.MapCard(card))
}

// Ego возвращает окружение узла в обоих направлениях.
//
//	@Summary	Окружение в один или два шага
//	@Tags		graph
//	@Produce	json
//	@Param		gid		path		string	true	"gid"
//	@Param		depth	query		int		false	"Глубина"	default(1)	Enums(1,2)
//	@Success	200		{object}	dto.GraphResponse
//	@Failure	400,404	{object}	httperr.Response
//	@Router		/nodes/{gid}/ego [get]
func (h *Handler) Ego(c *fiber.Ctx) error {
	id, err := gid(c)
	if err != nil {
		return err
	}
	depth, err := number(c, "depth", 1, 1, 2)
	if err != nil {
		return err
	}
	g, err := h.service.Ego(id, depth)
	if err != nil {
		return err
	}
	return c.JSON(dto.MapGraph(g))
}

// Top возвращает приоритетные узлы.
//
//	@Summary	Топ приоритетов
//	@Tags		graph
//	@Produce	json
//	@Param		n	query		int	false	"Число узлов"	default(30)
//	@Success	200	{array}		dto.TopDTO
//	@Failure	400	{object}	httperr.Response
//	@Router		/top [get]
func (h *Handler) Top(c *fiber.Ctx) error {
	n, err := number(c, "n", 30, 1, 10000)
	if err != nil {
		return err
	}
	return c.JSON(dto.MapTop(h.service.Top(n)))
}

// Clusters возвращает кластеры.
//
//	@Summary	Кластеры сети
//	@Tags		graph
//	@Produce	json
//	@Success	200	{array}	dto.ClusterDTO
//	@Router		/clusters [get]
func (h *Handler) Clusters(c *fiber.Ctx) error { return c.JSON(dto.MapClusters(h.service.Clusters())) }

// Search ищет gid по началу строки.
//
//	@Summary	Поиск по префиксу gid
//	@Tags		graph
//	@Produce	json
//	@Param		q		query		string	false	"Префикс gid"
//	@Param		limit	query		int		false	"Лимит"	default(10)
//	@Success	200		{array}		dto.NodeDTO
//	@Failure	400		{object}	httperr.Response
//	@Router		/search [get]
func (h *Handler) Search(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) > 19 || q != "" && !digits(q) {
		return invalid("q")
	}
	limit, err := number(c, "limit", 10, 1, 100)
	if err != nil {
		return err
	}
	return c.JSON(dto.MapNodes(h.service.Search(q, limit)))
}
