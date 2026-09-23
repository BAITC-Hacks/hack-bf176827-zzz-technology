package graph

import (
	"hackaton/internal/analysis"
	"hackaton/internal/data/parquet"
	"testing"
)

func testService() *service {
	r := &analysis.Result{Nodes: []analysis.NodeResult{
		{Gid: 100000003684369100, Role: "transit", ClusterID: 0, Features: analysis.Features{ComponentID: 1}},
		{Gid: 100000003684369101, Role: "terminal", ClusterID: 1, Features: analysis.Features{ComponentID: 1}},
		{Gid: 100000003684369102, Role: "peripheral", ClusterID: 1, Features: analysis.Features{ComponentID: 1}},
		{Gid: 100000003684369103, Role: "peripheral", ClusterID: 2, Features: analysis.Features{ComponentID: 2}},
	}, Edges: []parquet.Edge{{Src: 100000003684369100, Dst: 100000003684369101, SumKZT: 123, NTx: 2}, {Src: 100000003684369101, Dst: 100000003684369102, SumKZT: 456, NTx: 3}}, Top: []analysis.TopNode{{Gid: 100000003684369100}}, ByGid: map[int64]*analysis.NodeResult{}}
	for i := range r.Nodes {
		r.ByGid[r.Nodes[i].Gid] = &r.Nodes[i]
	}
	s := &service{}
	s.initialize(r)
	return s
}
func TestGraphFilters(t *testing.T) {
	s := testService()
	zero, one := 0, 1
	for _, tc := range []struct {
		name         string
		filter       Filter
		nodes, edges int
	}{
		{"all", Filter{}, 4, 2}, {"top neighbors", Filter{TopOnly: 1}, 2, 1}, {"zero cluster", Filter{Cluster: &zero}, 1, 0}, {"component", Filter{Component: &one}, 3, 2}, {"combined", Filter{TopOnly: 1, Role: "terminal", Cluster: &one}, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := s.Graph(tc.filter)
			if len(g.Nodes) != tc.nodes || len(g.Edges) != tc.edges {
				t.Fatalf("получено %d узлов/%d рёбер", len(g.Nodes), len(g.Edges))
			}
		})
	}
}
func TestEgoAndCard(t *testing.T) {
	s := testService()
	id := int64(100000003684369100)
	for depth, want := range map[int]int{1: 2, 2: 3} {
		g, e := s.Ego(id, depth)
		if e != nil || len(g.Nodes) != want {
			t.Fatalf("depth %d: %+v %v", depth, g, e)
		}
	}
	card, e := s.Node(id + 1)
	if e != nil || len(card.Incoming) != 1 || len(card.Outgoing) != 1 || card.Incoming[0].SumKZT != 123 {
		t.Fatalf("карточка: %+v %v", card, e)
	}
	if _, e = s.Ego(id, 3); e == nil {
		t.Fatal("принята глубина 3")
	}
	if _, e = s.Node(99); e == nil {
		t.Fatal("неизвестный gid принят")
	}
	g, e := s.Ego(id+3, 2)
	if e != nil || len(g.Nodes) != 1 || len(g.Edges) != 0 {
		t.Fatal("изолированный узел потерян")
	}
	if got := s.Search("10000000368436910", 2); len(got) != 2 {
		t.Fatal(got)
	}
}
