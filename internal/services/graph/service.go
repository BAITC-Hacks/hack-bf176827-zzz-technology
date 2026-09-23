// Package graph — чтение результата анализа для UI: фильтры, окружение узла, топ, кластеры, поиск.
package graph

import (
	"strconv"
	"strings"

	"hackaton/internal/data/graph"
	"hackaton/internal/data/models"
	"hackaton/internal/services"

	"go.uber.org/zap"
)

const (
	minEgoDepth = 1
	maxEgoDepth = 2
)

type Service interface {
	Graph(filter Filter) Subgraph
	Node(gid int64) (*NodeCard, error)
	Ego(gid int64, depth int) (Subgraph, error)
	Top(n int) []models.TopNode
	Clusters() []models.ClusterResult
	Search(prefix string, limit int) []models.NodeResult
	Seeds() []SeedInfo
	Robustness() []models.RobustnessStep
	NodeInfo(gid int64) (*models.NodeResult, bool)
}

type service struct {
	logger *zap.Logger
	result *models.AnalysisResult
	index  *graph.Index
	sorted map[string][]float64 // значения метрик по всем узлам, для перцентилей
}

type ServiceParams struct {
	services.FxBaseParams
	Result *models.AnalysisResult
}

func NewService(params ServiceParams) Service {
	return &service{
		logger: params.Logger.Named("graph_service"),
		result: params.Result,
		index:  graph.NewIndex(params.Result.Edges),
		sorted: buildPercentileIndex(params.Result.Nodes),
	}
}

// Graph — вся сеть или топ-N с соседями, с фильтрами по роли, кластеру и компоненте.
func (s *service) Graph(filter Filter) Subgraph {
	selected := map[int64]bool{}
	if filter.TopOnly > 0 {
		for _, top := range s.Top(filter.TopOnly) {
			selected[top.GID] = true
		}
		s.expand(selected, 1)
	} else {
		for _, node := range s.result.Nodes {
			selected[node.GID] = true
		}
	}
	for _, node := range s.result.Nodes {
		if !filter.matches(node) {
			delete(selected, node.GID)
		}
	}
	return s.subset(selected)
}

func (s *service) Node(gid int64) (*NodeCard, error) {
	node, ok := s.result.Node(gid)
	if !ok {
		return nil, ErrNodeNotFound
	}
	return &NodeCard{
		Node: *node, Incoming: s.index.Incoming[gid], Outgoing: s.index.Outgoing[gid],
		Percentiles: s.percentiles(node.Features), NearestSeed: s.nearestSeed(gid), NextCandidates: s.nextCandidates(gid, nil),
	}, nil
}

// Ego — окружение узла в обе стороны на depth шагов.
func (s *service) Ego(gid int64, depth int) (Subgraph, error) {
	if depth < minEgoDepth || depth > maxEgoDepth {
		return Subgraph{}, ErrInvalidDepth
	}
	if _, ok := s.result.Node(gid); !ok {
		return Subgraph{}, ErrNodeNotFound
	}
	return s.subset(s.expand(map[int64]bool{gid: true}, depth)), nil
}

func (s *service) Top(n int) []models.TopNode {
	if n < 0 {
		n = 0
	}
	if n > len(s.result.Top) {
		n = len(s.result.Top)
	}
	return s.result.Top[:n]
}

func (s *service) Clusters() []models.ClusterResult { return s.result.Clusters }

func (s *service) NodeInfo(gid int64) (*models.NodeResult, bool) { return s.result.Node(gid) }

// Search — узлы, чей GID начинается с prefix (пустой prefix — первые limit узлов).
func (s *service) Search(prefix string, limit int) []models.NodeResult {
	found := []models.NodeResult{}
	if limit <= 0 {
		return found
	}
	for _, node := range s.result.Nodes {
		if strings.HasPrefix(strconv.FormatInt(node.GID, 10), prefix) {
			found = append(found, node)
			if len(found) == limit {
				break
			}
		}
	}
	return found
}

// expand — добавляет соседей (в обе стороны) на depth шагов.
func (s *service) expand(selected map[int64]bool, depth int) map[int64]bool {
	frontier := make([]int64, 0, len(selected))
	for gid := range selected {
		frontier = append(frontier, gid)
	}
	for step := 0; step < depth; step++ {
		var next []int64
		for _, gid := range frontier {
			for _, edge := range s.index.Outgoing[gid] {
				if !selected[edge.Payee] {
					selected[edge.Payee] = true
					next = append(next, edge.Payee)
				}
			}
			for _, edge := range s.index.Incoming[gid] {
				if !selected[edge.Payer] {
					selected[edge.Payer] = true
					next = append(next, edge.Payer)
				}
			}
		}
		frontier = next
	}
	return selected
}

// subset — узлы из selected и рёбра между ними, в порядке результата анализа.
func (s *service) subset(selected map[int64]bool) Subgraph {
	sub := Subgraph{Nodes: []models.NodeResult{}, Edges: []models.Edge{}}
	for _, node := range s.result.Nodes {
		if selected[node.GID] {
			sub.Nodes = append(sub.Nodes, node)
		}
	}
	for _, edge := range s.result.Edges {
		if selected[edge.Payer] && selected[edge.Payee] {
			sub.Edges = append(sub.Edges, edge)
		}
	}
	return sub
}
