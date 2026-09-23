package analysis

import (
	"sort"

	"hackaton/internal/data/parquet"
	"hackaton/internal/graph"
)

// Run — полный расчёт: метрики → кластеры → роли → приоритет.
func Run(ds parquet.Dataset, opts Options) (*Result, error) {
	if opts.TopN <= 0 {
		opts.TopN = 30
	}
	g := graph.Build(ds)
	feats := computeFeatures(g)

	res := &Result{
		Edges: ds.Edges,
		ByGid: make(map[int64]*NodeResult, len(g.Gids)),
	}
	res.Nodes = make([]NodeResult, 0, len(g.Gids))
	for _, gid := range g.Gids {
		res.Nodes = append(res.Nodes, NodeResult{Gid: gid, Features: feats[gid]})
	}
	for i := range res.Nodes {
		res.ByGid[res.Nodes[i].Gid] = &res.Nodes[i]
	}

	assignClusters(g, res)
	assignRoles(g, res)
	assignPriority(g, res, opts.TopN)
	describeClusters(g, res)
	return res, nil
}

// sortedByPriority — индексы узлов по убыванию priority (стабильно по gid).
func sortedByPriority(res *Result) []int {
	idx := make([]int, len(res.Nodes))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		na, nb := res.Nodes[idx[a]], res.Nodes[idx[b]]
		if na.PriorityScore != nb.PriorityScore {
			return na.PriorityScore > nb.PriorityScore
		}
		return na.Gid < nb.Gid
	})
	return idx
}
