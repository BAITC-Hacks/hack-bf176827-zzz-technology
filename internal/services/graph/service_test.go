package graph

import (
	"errors"
	"testing"

	"hackaton/internal/data/graph"
	"hackaton/internal/data/models"

	"go.uber.org/zap"
)

func testService() *service {
	result := &models.AnalysisResult{
		Nodes: []models.NodeResult{
			{GID: 100000003684369100, Role: models.RoleTransit, ClusterID: 0, Features: models.Features{ComponentID: 1}},
			{GID: 100000003684369101, Role: models.RoleTerminal, ClusterID: 1, Features: models.Features{ComponentID: 1}},
			{GID: 100000003684369102, Role: models.RolePeripheral, ClusterID: 1, Features: models.Features{ComponentID: 1}},
			{GID: 100000003684369103, Role: models.RolePeripheral, ClusterID: 2, Features: models.Features{ComponentID: 2}},
		},
		Edges: []models.Edge{
			{Payer: 100000003684369100, Payee: 100000003684369101, SumKZT: 123, TxCount: 2},
			{Payer: 100000003684369101, Payee: 100000003684369102, SumKZT: 456, TxCount: 3},
		},
		Top: []models.TopNode{{GID: 100000003684369100}},
	}
	result.Index()
	return &service{logger: zap.NewNop(), result: result, index: graph.NewIndex(result.Edges)}
}

func TestGraphFilters(t *testing.T) {
	s := testService()
	zero, one := 0, 1
	for _, tc := range []struct {
		name         string
		filter       Filter
		nodes, edges int
	}{
		{"all", Filter{}, 4, 2},
		{"top neighbors", Filter{TopOnly: 1}, 2, 1},
		{"zero cluster", Filter{Cluster: &zero}, 1, 0},
		{"component", Filter{Component: &one}, 3, 2},
		{"combined", Filter{TopOnly: 1, Role: "terminal", Cluster: &one}, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sub := s.Graph(tc.filter)
			if len(sub.Nodes) != tc.nodes || len(sub.Edges) != tc.edges {
				t.Fatalf("получено %d узлов/%d рёбер", len(sub.Nodes), len(sub.Edges))
			}
		})
	}
}

func TestEgoAndCard(t *testing.T) {
	s := testService()
	gid := int64(100000003684369100)
	for depth, want := range map[int]int{1: 2, 2: 3} {
		sub, err := s.Ego(gid, depth)
		if err != nil || len(sub.Nodes) != want {
			t.Fatalf("depth %d: %+v %v", depth, sub, err)
		}
	}
	card, err := s.Node(gid + 1)
	if err != nil || len(card.Incoming) != 1 || len(card.Outgoing) != 1 || card.Incoming[0].SumKZT != 123 {
		t.Fatalf("карточка: %+v %v", card, err)
	}
	if _, err = s.Ego(gid, 3); !errors.Is(err, ErrInvalidDepth) {
		t.Fatalf("принята глубина 3: %v", err)
	}
	if _, err = s.Node(99); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("неизвестный gid принят: %v", err)
	}
	sub, err := s.Ego(gid+3, 2)
	if err != nil || len(sub.Nodes) != 1 || len(sub.Edges) != 0 {
		t.Fatalf("изолированный узел: %+v %v", sub, err)
	}
	if found := s.Search("10000000368436910", 2); len(found) != 2 {
		t.Fatalf("поиск: %d", len(found))
	}
}
