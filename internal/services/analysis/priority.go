package analysis

import (
	"fmt"
	"math"
	"sort"

	"hackaton/internal/data/models"
)

// Веса priority_score (перцентильные ранги). Формула — docs/methodology.md, раздел 5.
const (
	weightInKZT        = 0.25
	weightInDegree     = 0.20
	weightSeedUpstream = 0.15
	weightPageRank     = 0.15
	weightBetweenness  = 0.10
	weightRole         = 0.15

	seedMultiplier      = 0.5 // seed уже известны — фокус на тех, кто выше
	truncatedMultiplier = 0.7 // неполные данные
)

var roleBonus = map[models.Role]float64{
	models.RoleCoordinator: 1, models.RoleConsolidator: 1, models.RoleDistributor: 0.7,
	models.RoleTransit: 0.5, models.RoleTerminal: 0.4, models.RolePeripheral: 0,
}

func assignPriority(result *models.AnalysisResult, topN int) {
	inKZT := percentiles(result, func(f models.Features) float64 { return f.InKZT })
	inDegree := percentiles(result, func(f models.Features) float64 { return float64(f.InDegree) })
	seedUpstream := percentiles(result, func(f models.Features) float64 { return float64(f.SeedUpstream) })
	pageRank := percentiles(result, func(f models.Features) float64 { return f.PageRank })
	betweenness := percentiles(result, func(f models.Features) float64 { return f.Betweenness })

	for i := range result.Nodes {
		node := &result.Nodes[i]
		score := weightInKZT*inKZT[i] + weightInDegree*inDegree[i] + weightSeedUpstream*seedUpstream[i] +
			weightPageRank*pageRank[i] + weightBetweenness*betweenness[i] + weightRole*roleBonus[node.Role]
		if node.Features.IsSeed {
			score *= seedMultiplier
		}
		if node.Features.Truncated {
			score *= truncatedMultiplier
		}
		node.PriorityScore = math.Round(clamp01(score)*10000) / 10000
	}

	result.Top = result.Top[:0]
	for rank, i := range sortedByPriority(result) {
		if rank >= topN {
			break
		}
		node := result.Nodes[i]
		why := fmt.Sprintf("%s. Приоритет: вход %s KZT (top %.0f%%), %d seed выше по цепочке, PageRank top %.0f%%",
			node.Evidence, formatKZT(node.Features.InKZT), (1-inKZT[i])*100, node.Features.SeedUpstream, (1-pageRank[i])*100)
		if node.Features.IsSeed {
			why += "; seed (×0.5)"
		}
		if node.Features.Truncated {
			why += "; обрезан обходом (×0.7)"
		}
		result.Top = append(result.Top, models.TopNode{Rank: rank + 1, GID: node.GID, Role: node.Role, PriorityScore: node.PriorityScore, Why: why})
	}
}

// percentiles — доля узлов со значением меньше (ничьи считаются наполовину), по индексу узла.
func percentiles(result *models.AnalysisResult, value func(models.Features) float64) []float64 {
	count := len(result.Nodes)
	order := make([]int, count)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return value(result.Nodes[order[a]].Features) < value(result.Nodes[order[b]].Features)
	})
	ranks := make([]float64, count)
	for start := 0; start < count; {
		end := start
		for end < count && value(result.Nodes[order[end]].Features) == value(result.Nodes[order[start]].Features) {
			end++
		}
		rank := (float64(start) + float64(end-start-1)/2) / float64(count-1)
		for k := start; k < end; k++ {
			ranks[order[k]] = rank
		}
		start = end
	}
	return ranks
}
