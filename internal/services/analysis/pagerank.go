package analysis

import (
	"math"
	"sort"

	"hackaton/internal/data/graph"
)

// weightedPageRank — PageRank с весом = сумма перевода. Своя реализация ради детерминизма:
// gonum обходит узлы в порядке map, и последние знаки плавают между запусками.
func weightedPageRank(txGraph *graph.Graph, damping, tolerance float64, maxIterations int) map[int64]float64 {
	gids := append([]int64(nil), txGraph.GIDs...)
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })
	total := float64(len(gids))

	outWeight := make(map[int64]float64, len(gids))
	for _, gid := range gids {
		for _, edge := range txGraph.Outgoing[gid] {
			outWeight[gid] += edge.SumKZT
		}
	}
	rank := make(map[int64]float64, len(gids))
	for _, gid := range gids {
		rank[gid] = 1 / total
	}
	for iteration := 0; iteration < maxIterations; iteration++ {
		dangling := 0.0
		for _, gid := range gids {
			if outWeight[gid] == 0 {
				dangling += rank[gid]
			}
		}
		base := (1-damping)/total + damping*dangling/total
		next := make(map[int64]float64, len(gids))
		for _, gid := range gids {
			next[gid] = base
		}
		for _, gid := range gids {
			if outWeight[gid] == 0 {
				continue
			}
			share := damping * rank[gid] / outWeight[gid]
			for _, edge := range txGraph.Outgoing[gid] {
				next[edge.Payee] += share * edge.SumKZT
			}
		}
		delta := 0.0
		for _, gid := range gids {
			delta += math.Abs(next[gid] - rank[gid])
		}
		rank = next
		if delta < tolerance {
			break
		}
	}
	return rank
}
