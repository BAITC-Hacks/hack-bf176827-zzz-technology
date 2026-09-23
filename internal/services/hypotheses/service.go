// Package hypotheses — LLM-формулировка гипотез кластеров по посчитанным фактам, с кэшем.
// Без ключа берёт только кэш; без кэша остаётся шаблон из analysis.
package hypotheses

import (
	"context"
	"encoding/json"
	"fmt"

	graphdto "hackaton/internal/data/dto/graph"
	"hackaton/internal/data/models"
	"hackaton/internal/services"
	"hackaton/pkg/llm"

	"go.uber.org/zap"
)

const (
	minClusterSize = 5  // кластеры меньше остаются с шаблоном
	batchSize      = 10 // кластеров в одном запросе
)

type Service interface {
	// Enrich переписывает Hypothesis кластеров; allowRemote=false — только кэш. Возвращает число обновлённых.
	Enrich(ctx context.Context, result *models.AnalysisResult, allowRemote bool) (int, error)
}

type service struct {
	logger *zap.Logger
	llm    *llm.Client
	cache  *llm.Cache
}

type ServiceParams struct {
	services.FxBaseParams
	LLM   *llm.Client
	Cache *llm.Cache
}

func NewService(params ServiceParams) Service {
	return &service{logger: params.Logger.Named("hypotheses_service"), llm: params.LLM, cache: params.Cache}
}

type pendingCluster struct {
	index    int
	cacheKey string
	facts    map[string]any
}

func (s *service) Enrich(ctx context.Context, result *models.AnalysisResult, allowRemote bool) (int, error) {
	var pending []pendingCluster
	updated := 0
	for i := range result.Clusters {
		cluster := &result.Clusters[i]
		if cluster.NodeCount < minClusterSize {
			continue
		}
		facts := clusterFacts(result, cluster)
		encoded, _ := json.Marshal(facts)
		cacheKey := llm.Key("hyp", s.llm.Model(), string(encoded))
		if text, ok := s.cache.Get(cacheKey); ok {
			cluster.Hypothesis = text
			updated++
			continue
		}
		pending = append(pending, pendingCluster{index: i, cacheKey: cacheKey, facts: facts})
	}
	if len(pending) == 0 || !allowRemote || !s.llm.Enabled() {
		return updated, nil
	}

	for start := 0; start < len(pending); start += batchSize {
		end := min(start+batchSize, len(pending))
		count, err := s.requestBatch(ctx, result, pending[start:end])
		updated += count
		if err != nil {
			return updated, err
		}
	}
	if err := s.cache.Save(); err != nil {
		return updated, fmt.Errorf("сохранение кэша: %w", err)
	}
	return updated, nil
}

func (s *service) requestBatch(ctx context.Context, result *models.AnalysisResult, batch []pendingCluster) (int, error) {
	payload := make([]map[string]any, 0, len(batch))
	for _, item := range batch {
		payload = append(payload, item.facts)
	}
	encoded, _ := json.Marshal(payload)
	raw, err := s.llm.Complete(ctx, systemPrompt, hypothesisPrompt+"\n\n"+string(encoded), "hypotheses", hypothesisSchema)
	if err != nil {
		return 0, fmt.Errorf("гипотезы кластеров: %w", err)
	}
	var parsed hypothesesReply
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return 0, fmt.Errorf("гипотезы: некорректный json от модели: %w", err)
	}
	byCluster := make(map[int]string, len(parsed.Items))
	for _, item := range parsed.Items {
		byCluster[item.ClusterID] = item.Hypothesis
	}
	updated := 0
	for _, item := range batch {
		cluster := &result.Clusters[item.index]
		text, ok := byCluster[cluster.ClusterID]
		if !ok || text == "" {
			s.logger.Warn("модель не вернула гипотезу", zap.Int("cluster_id", cluster.ClusterID))
			continue
		}
		cluster.Hypothesis = text
		s.cache.Put(item.cacheKey, text)
		updated++
	}
	return updated, nil
}

func clusterFacts(result *models.AnalysisResult, cluster *models.ClusterResult) map[string]any {
	top := make([]map[string]any, 0, len(cluster.TopGIDs))
	for _, gid := range cluster.TopGIDs {
		if node, ok := result.Node(gid); ok {
			top = append(top, map[string]any{"gid": graphdto.GID(gid), "role": node.Role, "priority": node.PriorityScore, "evidence": node.Evidence})
		}
	}
	return map[string]any{
		"cluster_id": cluster.ClusterID, "n_nodes": cluster.NodeCount, "n_seed": cluster.SeedCount,
		"sum_kzt_internal": cluster.SumKZTInternal, "sum_kzt_in": cluster.SumKZTIn, "sum_kzt_out": cluster.SumKZTOut,
		"roles": cluster.RoleCounts, "top_nodes": top, "template_hypothesis": cluster.Hypothesis,
	}
}
