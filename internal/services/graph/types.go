package graph

import "hackaton/internal/data/models"

// Filter — параметры выборки для /graph.
type Filter struct {
	Role      string
	Cluster   *int
	Component *int
	TopOnly   int // > 0: топ-N по приоритету и их соседи в один шаг
}

func (f Filter) matches(node models.NodeResult) bool {
	if f.Role != "" && string(node.Role) != f.Role {
		return false
	}
	if f.Cluster != nil && node.ClusterID != *f.Cluster {
		return false
	}
	if f.Component != nil && node.Features.ComponentID != *f.Component {
		return false
	}
	return true
}

type Subgraph struct {
	Nodes []models.NodeResult
	Edges []models.Edge
}

type NodeCard struct {
	Node     models.NodeResult
	Incoming []models.Edge // по убыванию суммы
	Outgoing []models.Edge
}
