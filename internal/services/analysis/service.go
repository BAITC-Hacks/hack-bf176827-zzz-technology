// Package analysis — метрики, роли, кластеры, приоритеты и паттерны по графу переводов.
// Формулы и пороги — docs/methodology.md.
package analysis

import (
	"context"
	"sort"

	"hackaton/internal/data/graph"
	"hackaton/internal/data/models"
	"hackaton/internal/services"

	"go.uber.org/zap"
)

type Service interface {
	Analyze(ctx context.Context, dataset models.Dataset, params Params) (*models.AnalysisResult, error)
}

// Params — настройки расчёта.
type Params struct {
	TopN int // размер топ-листа, по умолчанию 30
}

type service struct {
	logger *zap.Logger
}

type ServiceParams struct {
	services.FxBaseParams
}

func NewService(params ServiceParams) Service {
	return &service{logger: params.Logger.Named("analysis_service")}
}

// Analyze — полный детерминированный расчёт: метрики → кластеры → паттерны → роли → приоритет → кластерная сводка.
func (s *service) Analyze(_ context.Context, dataset models.Dataset, params Params) (*models.AnalysisResult, error) {
	if params.TopN <= 0 {
		params.TopN = 30
	}
	txGraph := graph.Build(dataset)
	features := computeFeatures(txGraph)

	result := &models.AnalysisResult{Edges: dataset.Edges, Nodes: make([]models.NodeResult, 0, len(txGraph.GIDs))}
	for _, gid := range txGraph.GIDs {
		result.Nodes = append(result.Nodes, models.NodeResult{GID: gid, Features: features[gid]})
	}
	result.Index()

	assignClusters(txGraph, result)
	findPatterns(txGraph, result)
	assignRoles(result)
	assignPriority(result, params.TopN)
	result.Robustness = robustnessSteps(result)
	describeClusters(result)

	s.logger.Debug("analysis done",
		zap.Int("nodes", len(result.Nodes)), zap.Int("clusters", len(result.Clusters)),
		zap.Int("cycles", len(result.Cycles)), zap.Int("routes", len(result.Routes)))
	return result, nil
}

// sortedByPriority — индексы узлов по убыванию приоритета, при равенстве по GID.
func sortedByPriority(result *models.AnalysisResult) []int {
	order := make([]int, len(result.Nodes))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		left, right := result.Nodes[order[a]], result.Nodes[order[b]]
		if left.PriorityScore != right.PriorityScore {
			return left.PriorityScore > right.PriorityScore
		}
		return left.GID < right.GID
	})
	return order
}
