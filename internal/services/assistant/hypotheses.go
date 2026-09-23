package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"hackaton/internal/analysis"
	"hackaton/internal/llm"
)

const hypothesisPrompt = `Для каждого кластера ниже (JSON) сформулируй гипотезу о его назначении в сети: 1–2 предложения, до 300 символов,
по-русски, с опорой на числа (seed, роли, обороты) и с упоминанием 1–2 ключевых gid. Это гипотеза для проверки, не вывод о виновности.
Шаблонная гипотеза дана как подсказка — улучши её, не противоречь фактам.`

var hypothesisSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"items": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id": map[string]any{"type": "integer"},
					"hypothesis": map[string]any{"type": "string"},
				},
				"required":             []string{"cluster_id", "hypothesis"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"items"},
	"additionalProperties": false,
}

// EnrichHypotheses — переписывает Hypothesis кластеров с n_nodes ≥ minNodes через LLM (батчами), с кэшем.
// Без ключа берёт только кэш; шаблон остаётся, если ничего нет. Возвращает число обновлённых кластеров.
func (s *Service) EnrichHypotheses(ctx context.Context, minNodes, batch int) (int, error) {
	type item struct {
		idx   int
		key   string
		facts map[string]any
	}
	var pending []item
	updated := 0
	for i := range s.res.Clusters {
		c := &s.res.Clusters[i]
		if c.NNodes < minNodes {
			continue
		}
		facts := s.clusterFacts(c)
		fb, _ := json.Marshal(facts)
		key := llm.Key("hyp", s.llm.Model(), string(fb))
		if s.cache != nil {
			if v, ok := s.cache.Get(key); ok {
				c.Hypothesis = v
				updated++
				continue
			}
		}
		pending = append(pending, item{idx: i, key: key, facts: facts})
	}
	if len(pending) == 0 || !s.llm.Enabled() {
		return updated, nil
	}
	if batch <= 0 {
		batch = 10
	}
	for start := 0; start < len(pending); start += batch {
		end := min(start+batch, len(pending))
		var payload []map[string]any
		for _, it := range pending[start:end] {
			payload = append(payload, it.facts)
		}
		pb, _ := json.Marshal(payload)
		out, err := s.llm.Complete(ctx, systemPrompt, hypothesisPrompt+"\n\n"+string(pb), "hypotheses", hypothesisSchema)
		if err != nil {
			return updated, fmt.Errorf("гипотезы кластеров: %w", err)
		}
		var parsed struct {
			Items []struct {
				ClusterID  int    `json:"cluster_id"`
				Hypothesis string `json:"hypothesis"`
			} `json:"items"`
		}
		if err := json.Unmarshal([]byte(out), &parsed); err != nil {
			return updated, fmt.Errorf("гипотезы: bad json: %w", err)
		}
		byID := map[int]string{}
		for _, p := range parsed.Items {
			byID[p.ClusterID] = p.Hypothesis
		}
		for _, it := range pending[start:end] {
			c := &s.res.Clusters[it.idx]
			h, ok := byID[c.ClusterID]
			if !ok || h == "" {
				continue
			}
			c.Hypothesis = h
			if s.cache != nil {
				s.cache.Put(it.key, h)
			}
			updated++
		}
	}
	if s.cache != nil {
		if err := s.cache.Save(); err != nil {
			return updated, err
		}
	}
	return updated, nil
}

func (s *Service) clusterFacts(c *analysis.ClusterResult) map[string]any {
	var top []map[string]any
	for _, g := range c.TopGids {
		if n := s.res.ByGid[g]; n != nil {
			top = append(top, map[string]any{"gid": strconv.FormatInt(g, 10), "role": n.Role, "priority": n.PriorityScore, "evidence": n.Evidence})
		}
	}
	return map[string]any{
		"cluster_id": c.ClusterID, "n_nodes": c.NNodes, "n_seed": c.NSeed,
		"sum_kzt_internal": c.SumKZTInternal, "sum_kzt_in": c.SumKZTIn, "sum_kzt_out": c.SumKZTOut,
		"roles": c.RoleCounts, "top_nodes": top, "template_hypothesis": c.Hypothesis,
	}
}
