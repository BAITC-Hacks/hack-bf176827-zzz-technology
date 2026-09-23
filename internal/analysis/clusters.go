package analysis

import (
	"fmt"
	"math/rand/v2"
	"sort"

	"hackaton/internal/graph"

	gg "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/community"
)

// assignClusters — Louvain на неориентированной проекции (вес = сумма обоих направлений), фиксированный seed.
// Изолированные узлы получают свои cluster_id. id по убыванию размера кластера.
func assignClusters(g *graph.Graph, res *Result) {
	ug := gg.UndirectWeighted{G: g.G, Absent: 0, Merge: func(x, y float64, _, _ gg.Edge) float64 { return x + y }}
	reduced := community.Modularize(ug, 1.0, rand.New(rand.NewPCG(42, 0)))
	comms := reduced.Communities()
	sort.SliceStable(comms, func(i, j int) bool {
		if len(comms[i]) != len(comms[j]) {
			return len(comms[i]) > len(comms[j])
		}
		return minID(comms[i]) < minID(comms[j])
	})
	cluster := map[int64]int{}
	for id, c := range comms {
		for _, n := range c {
			cluster[n.ID()] = id
		}
	}
	next := len(comms)
	for i := range res.Nodes {
		n := &res.Nodes[i]
		id, ok := cluster[n.Gid]
		if !ok {
			id = next
			next++
		}
		n.ClusterID = id
	}
	// сколько разных кластеров среди плательщиков
	for i := range res.Nodes {
		n := &res.Nodes[i]
		seen := map[int]bool{}
		for _, e := range g.In[n.Gid] {
			if src, ok := res.ByGid[e.Src]; ok {
				seen[src.ClusterID] = true
			}
		}
		n.Features.NClustersIn = len(seen)
	}
}

func minID(nodes []gg.Node) int64 {
	m := nodes[0].ID()
	for _, n := range nodes[1:] {
		if n.ID() < m {
			m = n.ID()
		}
	}
	return m
}

// describeClusters — статистика и шаблонная гипотеза по каждому кластеру (после ролей и приоритетов).
func describeClusters(g *graph.Graph, res *Result) {
	byID := map[int]*ClusterResult{}
	for _, n := range res.Nodes {
		c := byID[n.ClusterID]
		if c == nil {
			c = &ClusterResult{ClusterID: n.ClusterID, ComponentID: n.Features.ComponentID, RoleCounts: map[string]int{}}
			byID[n.ClusterID] = c
		}
		c.NNodes++
		if n.Features.IsSeed {
			c.NSeed++
		}
		c.RoleCounts[n.Role]++
	}
	for _, e := range res.Edges {
		s, d := res.ByGid[e.Src], res.ByGid[e.Dst]
		if s == nil || d == nil {
			continue
		}
		if s.ClusterID == d.ClusterID {
			byID[s.ClusterID].SumKZTInternal += e.SumKZT
		} else {
			byID[s.ClusterID].SumKZTOut += e.SumKZT
			byID[d.ClusterID].SumKZTIn += e.SumKZT
		}
	}
	for _, i := range sortedByPriority(res) {
		n := res.Nodes[i]
		c := byID[n.ClusterID]
		if len(c.TopGids) < 5 {
			c.TopGids = append(c.TopGids, n.Gid)
		}
	}
	res.Clusters = res.Clusters[:0]
	for _, c := range byID {
		c.Hypothesis = clusterHypothesis(c)
		res.Clusters = append(res.Clusters, *c)
	}
	sort.Slice(res.Clusters, func(i, j int) bool { return res.Clusters[i].ClusterID < res.Clusters[j].ClusterID })
}

// clusterHypothesis — шаблон по составу ролей; LLM-версия (если включена) перезаписывает поверх.
func clusterHypothesis(c *ClusterResult) string {
	if c.NNodes == 1 {
		if c.NSeed == 1 {
			return "изолированный seed: нет переводов ≥5000 KZT в июле, вне сети"
		}
		return "изолированный узел"
	}
	rc := c.RoleCounts
	s := fmt.Sprintf("%d узлов, %d seed, внутри %s KZT (вход %s, выход %s). ", c.NNodes, c.NSeed, kzt(c.SumKZTInternal), kzt(c.SumKZTIn), kzt(c.SumKZTOut))
	switch {
	case c.NSeed >= 2 && rc[RoleConsolidator] > 0:
		s += fmt.Sprintf("Признаки сбора с %d seed через точки консолидации (%d)", c.NSeed, rc[RoleConsolidator])
		if rc[RoleDistributor] > 0 {
			s += fmt.Sprintf(" и веерного вывода (распределителей: %d)", rc[RoleDistributor])
		}
		s += " — гипотеза: ячейка с общим сборщиком."
	case rc[RoleDistributor] > 0 && rc[RoleTerminal] >= 5:
		s += fmt.Sprintf("Распределителей %d, конечных получателей %d — гипотеза: выплаты/раздача вниз по сети.", rc[RoleDistributor], rc[RoleTerminal])
	case rc[RoleTransit] >= 3:
		s += fmt.Sprintf("Транзитных узлов %d — гипотеза: цепочка прогона средств.", rc[RoleTransit])
	case c.NSeed >= 1 && rc[RoleConsolidator] == 0 && rc[RoleCoordinator] == 0:
		s += "Seed без выраженного сборщика — гипотеза: периферийная группа, деньги рассеиваются."
	default:
		s += "Выраженной структуры не выявлено — требуется ручная проверка топ-узлов."
	}
	return s
}

func kzt(v float64) string {
	switch {
	case v >= 1e6:
		return fmt.Sprintf("%.1f млн", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%.0f тыс", v/1e3)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}
