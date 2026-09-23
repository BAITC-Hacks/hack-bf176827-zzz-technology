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
	Node           models.NodeResult
	Incoming       []models.Edge // по убыванию суммы
	Outgoing       []models.Edge
	Percentiles    map[string]float64 // метрика → доля узлов с меньшим значением, 0–100
	NearestSeed    *NearestSeed
	NextCandidates []Candidate
}

type NearestSeed struct {
	GID   int64
	Steps int
}

// Candidate — следующий вероятный ключевой узел (эвристика: доля потока × приоритет).
type Candidate struct {
	GID       int64
	Role      models.Role
	Priority  float64
	Hops      int
	Direction string // down | up
	FlowShare float64
	Score     float64
	ScorePct  int
}

// SeedInfo — исходный участник и его крупнейшие получатели.
type SeedInfo struct {
	Node models.NodeResult
	Next []models.Edge
}

const seedNextLimit = 3
