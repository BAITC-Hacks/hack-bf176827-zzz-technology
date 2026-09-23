package analysis

import (
	"fmt"
	"sort"

	"hackaton/internal/graph"
)

// Веса priority_score (перцентильные ранги, см. README).
var priorityWeights = struct{ inKZT, inDeg, seedUp, pagerank, btw, role float64 }{0.25, 0.20, 0.15, 0.15, 0.10, 0.15}

var roleBonus = map[string]float64{
	RoleCoordinator: 1, RoleConsolidator: 1, RoleDistributor: 0.7, RoleTransit: 0.5, RoleTerminal: 0.4, RolePeripheral: 0,
}

const (
	seedMultiplier      = 0.5 // seed уже известны — фокус на тех, кто выше
	truncatedMultiplier = 0.7 // неполные данные
)

func assignPriority(g *graph.Graph, res *Result, topN int) {
	pInKZT := percentiles(res, func(f Features) float64 { return f.InKZT })
	pInDeg := percentiles(res, func(f Features) float64 { return float64(f.InDeg) })
	pSeedUp := percentiles(res, func(f Features) float64 { return float64(f.NSeedUpstream) })
	pPR := percentiles(res, func(f Features) float64 { return f.PageRank })
	pBtw := percentiles(res, func(f Features) float64 { return f.Betweenness })
	w := priorityWeights

	for i := range res.Nodes {
		n := &res.Nodes[i]
		f := n.Features
		v := w.inKZT*pInKZT[i] + w.inDeg*pInDeg[i] + w.seedUp*pSeedUp[i] + w.pagerank*pPR[i] + w.btw*pBtw[i] + w.role*roleBonus[n.Role]
		if f.IsSeed {
			v *= seedMultiplier
		}
		if f.Truncated {
			v *= truncatedMultiplier
		}
		n.PriorityScore = round4(clamp(v, 0, 1))
	}

	res.Top = res.Top[:0]
	for rank, i := range sortedByPriority(res) {
		if rank >= topN {
			break
		}
		n := res.Nodes[i]
		why := fmt.Sprintf("%s. Приоритет: вход %s KZT (top %.0f%%), %d seed выше по цепочке, PageRank top %.0f%%",
			n.Evidence, kzt(n.Features.InKZT), (1-pInKZT[i])*100, n.Features.NSeedUpstream, (1-pPR[i])*100)
		if n.Features.IsSeed {
			why += "; seed (×0.5)"
		}
		if n.Features.Truncated {
			why += "; обрезан обходом (×0.7)"
		}
		res.Top = append(res.Top, TopNode{Rank: rank + 1, Gid: n.Gid, Role: n.Role, PriorityScore: n.PriorityScore, Why: why})
	}
}

// percentiles — доля узлов со значением меньше (ничьи считаются наполовину), 0..1 по индексу узла.
func percentiles(res *Result, val func(Features) float64) []float64 {
	n := len(res.Nodes)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return val(res.Nodes[idx[a]].Features) < val(res.Nodes[idx[b]].Features) })
	out := make([]float64, n)
	for i := 0; i < n; {
		j := i
		for j < n && val(res.Nodes[idx[j]].Features) == val(res.Nodes[idx[i]].Features) {
			j++
		}
		p := (float64(i) + float64(j-i-1)/2) / float64(n-1)
		for k := i; k < j; k++ {
			out[idx[k]] = p
		}
		i = j
	}
	return out
}

func round4(v float64) float64 { return float64(int(v*10000+0.5)) / 10000 }
