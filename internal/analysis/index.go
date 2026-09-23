package analysis

import (
	"sort"

	"hackaton/internal/data/parquet"
)

// Index — рёбра по узлам для быстрых запросов (соседи, пути). Строится один раз по Result.
type Index struct {
	In  map[int64][]parquet.Edge // отсортированы по убыванию суммы
	Out map[int64][]parquet.Edge
}

func BuildIndex(res *Result) *Index {
	idx := &Index{In: map[int64][]parquet.Edge{}, Out: map[int64][]parquet.Edge{}}
	for _, e := range res.Edges {
		idx.Out[e.Src] = append(idx.Out[e.Src], e)
		idx.In[e.Dst] = append(idx.In[e.Dst], e)
	}
	bySum := func(m map[int64][]parquet.Edge) {
		for _, es := range m {
			sort.Slice(es, func(i, j int) bool { return es[i].SumKZT > es[j].SumKZT })
		}
	}
	bySum(idx.In)
	bySum(idx.Out)
	return idx
}

// Downstream — узлы, достижимые из start по исходящим за ≤ depth шагов, с расстоянием.
func (idx *Index) Downstream(start []int64, depth int) map[int64]int {
	return idx.walk(start, depth, idx.Out, func(e parquet.Edge) int64 { return e.Dst })
}

// Upstream — узлы, из которых достижим start по исходящим за ≤ depth шагов.
func (idx *Index) Upstream(start []int64, depth int) map[int64]int {
	return idx.walk(start, depth, idx.In, func(e parquet.Edge) int64 { return e.Src })
}

func (idx *Index) walk(start []int64, depth int, adj map[int64][]parquet.Edge, next func(parquet.Edge) int64) map[int64]int {
	dist := map[int64]int{}
	frontier := append([]int64(nil), start...)
	for _, s := range start {
		dist[s] = 0
	}
	for d := 1; d <= depth && len(frontier) > 0; d++ {
		var nf []int64
		for _, u := range frontier {
			for _, e := range adj[u] {
				v := next(e)
				if _, ok := dist[v]; !ok {
					dist[v] = d
					nf = append(nf, v)
				}
			}
		}
		frontier = nf
	}
	return dist
}

// Path — кратчайший по числу шагов маршрут src→dst (≤ maxLen рёбер), nil если нет.
func (idx *Index) Path(src, dst int64, maxLen int) []int64 {
	prev := map[int64]int64{src: src}
	frontier := []int64{src}
	for d := 0; d < maxLen && len(frontier) > 0; d++ {
		var nf []int64
		for _, u := range frontier {
			for _, e := range idx.Out[u] {
				if _, seen := prev[e.Dst]; seen {
					continue
				}
				prev[e.Dst] = u
				if e.Dst == dst {
					var path []int64
					for v := dst; v != src; v = prev[v] {
						path = append([]int64{v}, path...)
					}
					return append([]int64{src}, path...)
				}
				nf = append(nf, e.Dst)
			}
		}
		frontier = nf
	}
	return nil
}
