package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	graphdto "hackaton/internal/data/dto/graph"
	"hackaton/internal/data/models"
	"hackaton/pkg/llm"

	"go.uber.org/zap"
)

func (s *service) Card(ctx context.Context, gid int64) (Card, error) {
	node, ok := s.result.Node(gid)
	if !ok {
		return Card{}, ErrNodeNotFound
	}
	facts, _ := json.Marshal(s.buildNodeCard(node))
	cacheKey := llm.Key("card", s.llm.Model(), string(facts))
	if text, ok := s.cache.Get(cacheKey); ok {
		return Card{Text: text, ByLLM: true}, nil
	}
	if !s.llm.Enabled() {
		return Card{Text: s.templateCard(node)}, nil
	}
	text, err := s.llm.Complete(ctx, systemPrompt, cardPrompt+"\n\n"+string(facts), "", nil)
	if err != nil {
		s.logger.Warn("llm card failed, using template", zap.Error(err), zap.Int64("gid", gid))
		return Card{Text: s.templateCard(node)}, nil
	}
	s.cache.Put(cacheKey, text)
	if err := s.cache.Save(); err != nil {
		s.logger.Warn("failed to save llm cache", zap.Error(err))
	}
	return Card{Text: text, ByLLM: true}, nil
}

// templateCard — справка без LLM: те же четыре блока по шаблону.
func (s *service) templateCard(node *models.NodeResult) string {
	f := node.Features
	var sb strings.Builder
	fmt.Fprintf(&sb, "Роль и почему\n%s (уверенность %.2f). %s\n\n", node.Role, node.RoleScore, node.Evidence)
	fmt.Fprintf(&sb, "Потоки\nВход: %d плательщиков, %d переводов, %.0f KZT. Выход: %d получателей, %d переводов, %.0f KZT. Активных дней: %d.\n\n",
		f.InDegree, f.InTxCount, f.InKZT, f.OutDegree, f.OutTxCount, f.OutKZT, f.ActiveDays)
	sb.WriteString("Связи\n")
	for i, edge := range s.index.Incoming[node.GID] {
		if i >= 5 {
			break
		}
		fmt.Fprintf(&sb, "← %s: %.0f KZT (%d перев.)\n", graphdto.GID(edge.Payer), edge.SumKZT, edge.TxCount)
	}
	for i, edge := range s.index.Outgoing[node.GID] {
		if i >= 5 {
			break
		}
		fmt.Fprintf(&sb, "→ %s: %.0f KZT (%d перев.)\n", graphdto.GID(edge.Payee), edge.SumKZT, edge.TxCount)
	}
	sb.WriteString("\nНа что обратить внимание\n")
	switch {
	case f.Truncated:
		sb.WriteString("Исходящие не видны (4-е колено): запросить переводы этого клиента, чтобы понять, сток это или транзит.")
	case f.IsSeed:
		sb.WriteString("Seed: входящие занижены выгрузкой. Запросить входящие переводы извне выборки.")
	case f.SeedUpstream >= 3:
		fmt.Fprintf(&sb, "К узлу ведут цепочки от %d seed — кандидат на углублённую проверку.", f.SeedUpstream)
	default:
		sb.WriteString("Проверить контрагентов с наибольшими суммами и их роли.")
	}
	return sb.String()
}
