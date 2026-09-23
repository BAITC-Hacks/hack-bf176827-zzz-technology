package analysis

import (
	"sort"

	"hackaton/internal/data/graph"
	"hackaton/internal/data/models"
)

const (
	cycleMaxLen      = 5 // возвратные потоки: циклы длиной ≤
	repeatRouteMinTx = 2 // устойчивый маршрут A→B→C: переводов на каждом ребре ≥
)

var robustnessRemovalSizes = []int{5, 10, 20}

// findPatterns — циклы и устойчивые маршруты; помечает InCycle / RepeatRoutes у узлов.
func findPatterns(txGraph *graph.Graph, result *models.AnalysisResult) {
	result.Cycles = findCycles(txGraph, cycleMaxLen)
	inCycle := map[int64]bool{}
	for _, cycle := range result.Cycles {
		for _, gid := range cycle.GIDs {
			inCycle[gid] = true
		}
	}
	result.Routes = findRepeatRoutes(txGraph)
	routesVia := map[int64]int{}
	for _, route := range result.Routes {
		routesVia[route.Via]++
	}
	for i := range result.Nodes {
		node := &result.Nodes[i]
		node.Features.InCycle = inCycle[node.GID]
		node.Features.RepeatRoutes = routesVia[node.GID]
	}
}

// findCycles — простые циклы длиной ≤ maxLen, каждый один раз (старт = минимальный GID цикла).
func findCycles(txGraph *graph.Graph, maxLen int) []models.Cycle {
	gids := append([]int64(nil), txGraph.GIDs...)
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })

	var cycles []models.Cycle
	path := make([]int64, 0, maxLen)
	onPath := map[int64]bool{}
	var walk func(start, current int64, minSum float64)
	walk = func(start, current int64, minSum float64) {
		for _, edge := range txGraph.Outgoing[current] {
			next := edge.Payee
			sum := minSum
			if edge.SumKZT < sum {
				sum = edge.SumKZT
			}
			if next == start {
				cycles = append(cycles, models.Cycle{GIDs: append([]int64(nil), path...), SumKZT: sum})
				continue
			}
			if next < start || onPath[next] || len(path) >= maxLen {
				continue
			}
			onPath[next] = true
			path = append(path, next)
			walk(start, next, sum)
			path = path[:len(path)-1]
			onPath[next] = false
		}
	}
	for _, start := range gids {
		if len(txGraph.Outgoing[start]) == 0 || len(txGraph.Incoming[start]) == 0 {
			continue
		}
		path = append(path[:0], start)
		onPath[start] = true
		walk(start, start, 1e18)
		onPath[start] = false
	}
	sort.Slice(cycles, func(i, j int) bool { return cycles[i].SumKZT > cycles[j].SumKZT })
	return cycles
}

func findRepeatRoutes(txGraph *graph.Graph) []models.Route {
	var routes []models.Route
	for _, via := range txGraph.GIDs {
		for _, in := range txGraph.Incoming[via] {
			if in.TxCount < repeatRouteMinTx {
				continue
			}
			for _, out := range txGraph.Outgoing[via] {
				if out.TxCount < repeatRouteMinTx || out.Payee == in.Payer {
					continue
				}
				routes = append(routes, models.Route{From: in.Payer, Via: via, To: out.Payee,
					TxCount: min(in.TxCount, out.TxCount), SumKZT: min(in.SumKZT, out.SumKZT)})
			}
		}
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].TxCount != routes[j].TxCount {
			return routes[i].TxCount > routes[j].TxCount
		}
		return routes[i].SumKZT > routes[j].SumKZT
	})
	return routes
}

// Robustness — что остаётся от сети (по рёбрам) после изъятия узлов. Публичный: используется ассистентом.
func Robustness(result *models.AnalysisResult, removed map[int64]bool) models.RobustnessStep {
	seeds := map[int64]bool{}
	for _, node := range result.Nodes {
		if node.Features.IsSeed {
			seeds[node.GID] = true
		}
	}
	before := componentStats(result.Edges, nil)
	after := componentStats(result.Edges, removed)

	var lost, total float64
	hasLivePayer := map[int64]bool{}
	lostPayer := map[int64]bool{}
	seedStillPays := map[int64]bool{}
	seedHadOutgoing := map[int64]bool{}
	for _, edge := range result.Edges {
		total += edge.SumKZT
		if seeds[edge.Payer] {
			seedHadOutgoing[edge.Payer] = true
		}
		if removed[edge.Payer] || removed[edge.Payee] {
			lost += edge.SumKZT
			if removed[edge.Payer] && !removed[edge.Payee] {
				lostPayer[edge.Payee] = true
			}
			continue
		}
		hasLivePayer[edge.Payee] = true
		if seeds[edge.Payer] {
			seedStillPays[edge.Payer] = true
		}
	}
	step := models.RobustnessStep{
		ComponentsBefore: before.count, LargestBefore: before.largest,
		ComponentsAfter: after.count, LargestAfter: after.largest,
	}
	for gid := range lostPayer {
		if !hasLivePayer[gid] {
			step.NodesLostAllPayers++
		}
	}
	for seed := range seeds {
		if !removed[seed] && seedHadOutgoing[seed] && !seedStillPays[seed] {
			step.SeedsDisconnected++
		}
	}
	if total > 0 {
		step.LostTurnoverShare = lost / total
	}
	for gid := range removed {
		step.RemovedGIDs = append(step.RemovedGIDs, gid)
	}
	sort.Slice(step.RemovedGIDs, func(i, j int) bool { return step.RemovedGIDs[i] < step.RemovedGIDs[j] })
	return step
}

type componentSummary struct{ count, largest int }

func componentStats(edges []models.Edge, removed map[int64]bool) componentSummary {
	adjacency := map[int64][]int64{}
	for _, edge := range edges {
		if removed[edge.Payer] || removed[edge.Payee] {
			continue
		}
		adjacency[edge.Payer] = append(adjacency[edge.Payer], edge.Payee)
		adjacency[edge.Payee] = append(adjacency[edge.Payee], edge.Payer)
	}
	visited := map[int64]bool{}
	summary := componentSummary{}
	for start := range adjacency {
		if visited[start] {
			continue
		}
		summary.count++
		size := 0
		stack := []int64{start}
		visited[start] = true
		for len(stack) > 0 {
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			size++
			for _, neighbor := range adjacency[current] {
				if !visited[neighbor] {
					visited[neighbor] = true
					stack = append(stack, neighbor)
				}
			}
		}
		if size > summary.largest {
			summary.largest = size
		}
	}
	return summary
}

// robustnessSteps — изъятие top-5/10/20 не-seed узлов по приоритету (seed уже известны).
func robustnessSteps(result *models.AnalysisResult) []models.RobustnessStep {
	order := sortedByPriority(result)
	steps := make([]models.RobustnessStep, 0, len(robustnessRemovalSizes))
	for _, size := range robustnessRemovalSizes {
		removed := map[int64]bool{}
		for _, i := range order {
			if result.Nodes[i].Features.IsSeed {
				continue
			}
			removed[result.Nodes[i].GID] = true
			if len(removed) >= size {
				break
			}
		}
		steps = append(steps, Robustness(result, removed))
	}
	return steps
}
