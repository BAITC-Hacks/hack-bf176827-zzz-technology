package analysis

import (
	"fmt"
	"math/rand/v2"
	"sort"

	"hackaton/internal/data/graph"
	"hackaton/internal/data/models"

	gonumgraph "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/community"
)

const (
	louvainResolution = 1.0
	louvainSeed       = 42
	clusterTopSize    = 5
)

// assignClusters — Louvain на неориентированной проекции (вес = сумма обоих направлений), фиксированный seed.
// Изолированные узлы получают свои cluster_id. Нумерация по убыванию размера.
func assignClusters(txGraph *graph.Graph, result *models.AnalysisResult) {
	undirected := gonumgraph.UndirectWeighted{
		G:     txGraph.Weighted,
		Merge: func(x, y float64, _, _ gonumgraph.Edge) float64 { return x + y },
	}
	communities := community.Modularize(undirected, louvainResolution, rand.New(rand.NewPCG(louvainSeed, 0))).Communities()
	sort.SliceStable(communities, func(i, j int) bool {
		if len(communities[i]) != len(communities[j]) {
			return len(communities[i]) > len(communities[j])
		}
		return minNodeID(communities[i]) < minNodeID(communities[j])
	})
	clusterOf := map[int64]int{}
	for id, members := range communities {
		for _, node := range members {
			clusterOf[node.ID()] = id
		}
	}
	nextID := len(communities)
	for i := range result.Nodes {
		node := &result.Nodes[i]
		id, ok := clusterOf[node.GID]
		if !ok {
			id = nextID
			nextID++
		}
		node.ClusterID = id
	}
	for i := range result.Nodes {
		node := &result.Nodes[i]
		payerClusters := map[int]bool{}
		for _, edge := range txGraph.Incoming[node.GID] {
			if payer, ok := result.Node(edge.Payer); ok {
				payerClusters[payer.ClusterID] = true
			}
		}
		node.Features.ClustersIn = len(payerClusters)
	}
}

// describeClusters — статистика и шаблонная гипотеза по кластеру (после ролей, приоритетов и паттернов).
func describeClusters(result *models.AnalysisResult) {
	byID := map[int]*models.ClusterResult{}
	for _, node := range result.Nodes {
		cluster := byID[node.ClusterID]
		if cluster == nil {
			cluster = &models.ClusterResult{ClusterID: node.ClusterID, ComponentID: node.Features.ComponentID, RoleCounts: map[models.Role]int{}}
			byID[node.ClusterID] = cluster
		}
		cluster.NodeCount++
		if node.Features.IsSeed {
			cluster.SeedCount++
		}
		cluster.RoleCounts[node.Role]++
	}
	for _, edge := range result.Edges {
		payer, okPayer := result.Node(edge.Payer)
		payee, okPayee := result.Node(edge.Payee)
		if !okPayer || !okPayee {
			continue
		}
		if payer.ClusterID == payee.ClusterID {
			byID[payer.ClusterID].SumKZTInternal += edge.SumKZT
		} else {
			byID[payer.ClusterID].SumKZTOut += edge.SumKZT
			byID[payee.ClusterID].SumKZTIn += edge.SumKZT
		}
	}
	for _, cycle := range result.Cycles {
		if node, ok := result.Node(cycle.GIDs[0]); ok {
			byID[node.ClusterID].CycleCount++
		}
	}
	for _, i := range sortedByPriority(result) {
		node := result.Nodes[i]
		if cluster := byID[node.ClusterID]; len(cluster.TopGIDs) < clusterTopSize {
			cluster.TopGIDs = append(cluster.TopGIDs, node.GID)
		}
	}
	result.Clusters = make([]models.ClusterResult, 0, len(byID))
	for _, cluster := range byID {
		cluster.Hypothesis = templateHypothesis(cluster)
		result.Clusters = append(result.Clusters, *cluster)
	}
	sort.Slice(result.Clusters, func(i, j int) bool { return result.Clusters[i].ClusterID < result.Clusters[j].ClusterID })
}

// templateHypothesis — гипотеза по составу ролей; LLM-версия (если есть) перезаписывает её поверх.
func templateHypothesis(cluster *models.ClusterResult) string {
	if cluster.NodeCount == 1 {
		if cluster.SeedCount == 1 {
			return "изолированный seed: нет переводов ≥5000 KZT в июле, вне сети"
		}
		return "изолированный узел"
	}
	roles := cluster.RoleCounts
	text := fmt.Sprintf("%d узлов, %d seed, внутри %s KZT (вход %s, выход %s). ",
		cluster.NodeCount, cluster.SeedCount, formatKZT(cluster.SumKZTInternal), formatKZT(cluster.SumKZTIn), formatKZT(cluster.SumKZTOut))
	switch {
	case cluster.SeedCount >= 2 && roles[models.RoleConsolidator] > 0:
		text += fmt.Sprintf("Признаки сбора с %d seed через точки консолидации (%d)", cluster.SeedCount, roles[models.RoleConsolidator])
		if roles[models.RoleDistributor] > 0 {
			text += fmt.Sprintf(" и веерного вывода (распределителей: %d)", roles[models.RoleDistributor])
		}
		text += " — гипотеза: ячейка с общим сборщиком."
	case roles[models.RoleDistributor] > 0 && roles[models.RoleTerminal] >= 5:
		text += fmt.Sprintf("Распределителей %d, конечных получателей %d — гипотеза: выплаты/раздача вниз по сети.", roles[models.RoleDistributor], roles[models.RoleTerminal])
	case roles[models.RoleTransit] >= 3:
		text += fmt.Sprintf("Транзитных узлов %d — гипотеза: цепочка прогона средств.", roles[models.RoleTransit])
	case cluster.SeedCount >= 1 && roles[models.RoleConsolidator] == 0 && roles[models.RoleCoordinator] == 0:
		text += "Seed без выраженного сборщика — гипотеза: периферийная группа, деньги рассеиваются."
	default:
		text += "Выраженной структуры не выявлено — требуется ручная проверка топ-узлов."
	}
	return text
}
