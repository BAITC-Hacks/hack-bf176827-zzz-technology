package analysis

import (
	"sort"

	"hackaton/internal/data/parquet"
	"hackaton/internal/graph"
)

const (
	maxCycleLen    = 5 // возвратные потоки: циклы длиной ≤
	repeatRouteMin = 2 // устойчивый маршрут A→B→C: на обоих рёбрах переводов ≥
)

// Cycle — возвратный поток: деньги возвращаются к отправителю.
type Cycle struct {
	Nodes  []int64
	SumKZT float64 // минимальная сумма по рёбрам цикла (сколько реально «прокрутилось»)
}

// Route — устойчивая цепочка A→B→C с повторными переводами на обоих рёбрах.
type Route struct {
	A, B, C int64
	NTx     int64 // min(n_tx) по двум рёбрам
	SumKZT  float64
}

// RobustnessStep — что происходит с сетью при изъятии top-N узлов по приоритету.
type RobustnessStep struct {
	Removed             int      `json:"removed"`
	LostTurnoverShare   float64  `json:"lost_turnover_share"`
	ComponentsAfter     int      `json:"components_after"`
	LargestComponent    int      `json:"largest_component_after"`
	NodesLostAllPayers  int      `json:"nodes_lost_all_incoming"`
	LargestBefore       int      `json:"largest_component_before"`
	ComponentsBefore    int      `json:"components_before"`
	SeedsDisconnected   int      `json:"seeds_disconnected"` // seed, чьи все исходящие цепочки оборваны
	RemovedGids         []int64  `json:"-"`
	RemovedGidsAsString []string `json:"removed_gids"`
}

// findPatterns — циклы и маршруты; помечает Features.InCycle / RepeatRoutes.
func findPatterns(g *graph.Graph, res *Result) {
	res.Cycles = findCycles(g, maxCycleLen)
	inCycle := map[int64]bool{}
	for _, c := range res.Cycles {
		for _, n := range c.Nodes {
			inCycle[n] = true
		}
	}
	res.Routes = findRoutes(g)
	routesVia := map[int64]int{}
	for _, r := range res.Routes {
		routesVia[r.B]++
	}
	for i := range res.Nodes {
		n := &res.Nodes[i]
		n.Features.InCycle = inCycle[n.Gid]
		n.Features.RepeatRoutes = routesVia[n.Gid]
	}
}

// findCycles — простые циклы длиной ≤ maxLen; каждый цикл один раз (старт = минимальный gid в цикле).
func findCycles(g *graph.Graph, maxLen int) []Cycle {
	gids := append([]int64(nil), g.Gids...)
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })
	var cycles []Cycle
	path := make([]int64, 0, maxLen)
	onPath := map[int64]bool{}
	var dfs func(start, u int64, minSum float64)
	dfs = func(start, u int64, minSum float64) {
		for _, e := range g.Out[u] {
			v := e.Dst
			s := minSum
			if e.SumKZT < s {
				s = e.SumKZT
			}
			if v == start {
				cycles = append(cycles, Cycle{Nodes: append([]int64(nil), path...), SumKZT: s})
				continue
			}
			if v < start || onPath[v] || len(path) >= maxLen {
				continue
			}
			onPath[v] = true
			path = append(path, v)
			dfs(start, v, s)
			path = path[:len(path)-1]
			onPath[v] = false
		}
	}
	for _, s := range gids {
		if len(g.Out[s]) == 0 || len(g.In[s]) == 0 {
			continue
		}
		path = append(path[:0], s)
		onPath[s] = true
		dfs(s, s, 1e18)
		onPath[s] = false
	}
	sort.Slice(cycles, func(i, j int) bool { return cycles[i].SumKZT > cycles[j].SumKZT })
	return cycles
}

func findRoutes(g *graph.Graph) []Route {
	var routes []Route
	for _, b := range g.Gids {
		for _, in := range g.In[b] {
			if in.NTx < repeatRouteMin {
				continue
			}
			for _, out := range g.Out[b] {
				if out.NTx < repeatRouteMin || out.Dst == in.Src {
					continue
				}
				ntx := min(in.NTx, out.NTx)
				routes = append(routes, Route{A: in.Src, B: b, C: out.Dst, NTx: ntx, SumKZT: min(in.SumKZT, out.SumKZT)})
			}
		}
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].NTx != routes[j].NTx {
			return routes[i].NTx > routes[j].NTx
		}
		return routes[i].SumKZT > routes[j].SumKZT
	})
	return routes
}

// Robustness — изъять узлы и посмотреть, что осталось от сети (по рёбрам).
func Robustness(res *Result, removed map[int64]bool) RobustnessStep {
	seeds := map[int64]bool{}
	for _, n := range res.Nodes {
		if n.Features.IsSeed {
			seeds[n.Gid] = true
		}
	}
	before := componentsOf(res.Edges, nil)
	after := componentsOf(res.Edges, removed)
	var lost, total float64
	hasLivePayer := map[int64]bool{}
	orphanCand := map[int64]bool{}
	seedAlive := map[int64]bool{}
	for _, e := range res.Edges {
		total += e.SumKZT
		if removed[e.Src] || removed[e.Dst] {
			lost += e.SumKZT
			if removed[e.Src] && !removed[e.Dst] {
				orphanCand[e.Dst] = true
			}
			continue
		}
		hasLivePayer[e.Dst] = true
		if seeds[e.Src] {
			seedAlive[e.Src] = true
		}
	}
	lostAll := 0
	for gid := range orphanCand {
		if !hasLivePayer[gid] {
			lostAll++
		}
	}
	seedsCut := 0
	for s := range seeds {
		if removed[s] {
			continue
		}
		hadOut := false
		for _, e := range res.Edges {
			if e.Src == s {
				hadOut = true
				break
			}
		}
		if hadOut && !seedAlive[s] {
			seedsCut++
		}
	}
	step := RobustnessStep{
		Removed: len(removed), ComponentsBefore: before.count, LargestBefore: before.largest,
		ComponentsAfter: after.count, LargestComponent: after.largest, NodesLostAllPayers: lostAll, SeedsDisconnected: seedsCut,
	}
	if total > 0 {
		step.LostTurnoverShare = lost / total
	}
	for gid := range removed {
		step.RemovedGids = append(step.RemovedGids, gid)
	}
	sort.Slice(step.RemovedGids, func(i, j int) bool { return step.RemovedGids[i] < step.RemovedGids[j] })
	for _, gid := range step.RemovedGids {
		step.RemovedGidsAsString = append(step.RemovedGidsAsString, gidStr(gid))
	}
	return step
}

type compStats struct{ count, largest int }

func componentsOf(edges []parquet.Edge, removed map[int64]bool) compStats {
	adj := map[int64][]int64{}
	for _, e := range edges {
		if removed[e.Src] || removed[e.Dst] {
			continue
		}
		adj[e.Src] = append(adj[e.Src], e.Dst)
		adj[e.Dst] = append(adj[e.Dst], e.Src)
	}
	seen := map[int64]bool{}
	st := compStats{}
	for u := range adj {
		if seen[u] {
			continue
		}
		st.count++
		size := 0
		stack := []int64{u}
		seen[u] = true
		for len(stack) > 0 {
			x := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			size++
			for _, y := range adj[x] {
				if !seen[y] {
					seen[y] = true
					stack = append(stack, y)
				}
			}
		}
		if size > st.largest {
			st.largest = size
		}
	}
	return st
}

// robustnessSteps — изъятие top-5/10/20 по приоритету (seed не изымаем: они уже известны).
func robustnessSteps(res *Result) []RobustnessStep {
	var steps []RobustnessStep
	order := sortedByPriority(res)
	for _, n := range []int{5, 10, 20} {
		removed := map[int64]bool{}
		for _, i := range order {
			if res.Nodes[i].Features.IsSeed {
				continue
			}
			removed[res.Nodes[i].Gid] = true
			if len(removed) >= n {
				break
			}
		}
		steps = append(steps, Robustness(res, removed))
	}
	return steps
}
