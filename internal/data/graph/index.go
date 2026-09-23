package graph

import (
	"sort"

	"hackaton/internal/data/models"
)

// Index — рёбра по узлам для запросов соседей и путей (без gonum).
type Index struct {
	Incoming map[int64][]models.Edge // по убыванию суммы
	Outgoing map[int64][]models.Edge
}

func NewIndex(edges []models.Edge) *Index {
	index := &Index{Incoming: map[int64][]models.Edge{}, Outgoing: map[int64][]models.Edge{}}
	for _, edge := range edges {
		index.Outgoing[edge.Payer] = append(index.Outgoing[edge.Payer], edge)
		index.Incoming[edge.Payee] = append(index.Incoming[edge.Payee], edge)
	}
	sortBySum(index.Incoming)
	sortBySum(index.Outgoing)
	return index
}

func sortBySum(byNode map[int64][]models.Edge) {
	for _, edges := range byNode {
		sort.Slice(edges, func(i, j int) bool { return edges[i].SumKZT > edges[j].SumKZT })
	}
}

// Downstream — узлы, достижимые из start по исходящим за ≤ depth шагов, с расстоянием.
func (idx *Index) Downstream(start []int64, depth int) map[int64]int {
	return idx.walk(start, depth, idx.Outgoing, func(e models.Edge) int64 { return e.Payee })
}

// Upstream — узлы, из которых start достижим по исходящим за ≤ depth шагов.
func (idx *Index) Upstream(start []int64, depth int) map[int64]int {
	return idx.walk(start, depth, idx.Incoming, func(e models.Edge) int64 { return e.Payer })
}

func (idx *Index) walk(start []int64, depth int, adjacency map[int64][]models.Edge, next func(models.Edge) int64) map[int64]int {
	distance := map[int64]int{}
	frontier := append([]int64(nil), start...)
	for _, gid := range start {
		distance[gid] = 0
	}
	for step := 1; step <= depth && len(frontier) > 0; step++ {
		var nextFrontier []int64
		for _, gid := range frontier {
			for _, edge := range adjacency[gid] {
				neighbor := next(edge)
				if _, seen := distance[neighbor]; !seen {
					distance[neighbor] = step
					nextFrontier = append(nextFrontier, neighbor)
				}
			}
		}
		frontier = nextFrontier
	}
	return distance
}

// Path — кратчайший по числу шагов маршрут payer→payee (≤ maxLen рёбер), nil если нет.
func (idx *Index) Path(from, to int64, maxLen int) []int64 {
	previous := map[int64]int64{from: from}
	frontier := []int64{from}
	for step := 0; step < maxLen && len(frontier) > 0; step++ {
		var nextFrontier []int64
		for _, gid := range frontier {
			for _, edge := range idx.Outgoing[gid] {
				if _, seen := previous[edge.Payee]; seen {
					continue
				}
				previous[edge.Payee] = gid
				if edge.Payee == to {
					return unwind(previous, from, to)
				}
				nextFrontier = append(nextFrontier, edge.Payee)
			}
		}
		frontier = nextFrontier
	}
	return nil
}

func unwind(previous map[int64]int64, from, to int64) []int64 {
	var path []int64
	for gid := to; gid != from; gid = previous[gid] {
		path = append([]int64{gid}, path...)
	}
	return append([]int64{from}, path...)
}
