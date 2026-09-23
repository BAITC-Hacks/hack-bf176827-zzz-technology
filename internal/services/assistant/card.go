package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"hackaton/internal/analysis"
	"hackaton/internal/llm"
)

const cardPrompt = `Составь краткую справку по клиенту для AML-аналитика по фактам ниже (JSON). Четыре блока с заголовками:
"Роль и почему", "Потоки", "Связи", "На что обратить внимание". До 900 символов. Только факты из JSON, все gid полностью.
Формулировки — гипотезы ("признаки ..."), без утверждений о виновности. В последнем блоке — какой запрос аналитику стоит сделать дальше.`

// Card — справка по узлу: LLM по фактам, при отсутствии ключа — шаблон.
func (s *Service) Card(ctx context.Context, gid string) (text string, generated bool, err error) {
	g, err := parseGid(gid)
	if err != nil {
		return "", false, err
	}
	n, ok := s.res.ByGid[g]
	if !ok {
		return "", false, fmt.Errorf("узел %s не найден", gid)
	}
	facts, _ := s.nodeCard(gid)
	fb, _ := json.Marshal(facts)
	key := llm.Key("card", s.llm.Model(), string(fb))
	if s.cache != nil {
		if v, ok := s.cache.Get(key); ok {
			return v, true, nil
		}
	}
	if !s.llm.Enabled() {
		return s.templateCard(n), false, nil
	}
	out, err := s.llm.Complete(ctx, systemPrompt, cardPrompt+"\n\n"+string(fb), "", nil)
	if err != nil {
		return s.templateCard(n), false, err // шаблон + ошибка: вызывающий решает, показывать ли её
	}
	if s.cache != nil {
		s.cache.Put(key, out)
		_ = s.cache.Save()
	}
	return out, true, nil
}

func (s *Service) templateCard(n *analysis.NodeResult) string {
	f := n.Features
	var sb strings.Builder
	fmt.Fprintf(&sb, "Роль и почему\n%s (уверенность %.2f). %s\n\n", n.Role, n.RoleScore, n.Evidence)
	fmt.Fprintf(&sb, "Потоки\nВход: %d плательщиков, %d переводов, %.0f KZT. Выход: %d получателей, %d переводов, %.0f KZT. Активных дней: %d.\n\n",
		f.InDeg, f.InTx, f.InKZT, f.OutDeg, f.OutTx, f.OutKZT, f.ActiveDays)
	sb.WriteString("Связи\n")
	for i, e := range s.idx.In[n.Gid] {
		if i >= 5 {
			break
		}
		fmt.Fprintf(&sb, "← %s: %.0f KZT (%d перев.)\n", strconv.FormatInt(e.Src, 10), e.SumKZT, e.NTx)
	}
	for i, e := range s.idx.Out[n.Gid] {
		if i >= 5 {
			break
		}
		fmt.Fprintf(&sb, "→ %s: %.0f KZT (%d перев.)\n", strconv.FormatInt(e.Dst, 10), e.SumKZT, e.NTx)
	}
	sb.WriteString("\nНа что обратить внимание\n")
	switch {
	case f.Truncated:
		sb.WriteString("Исходящие не видны (4-е колено): запросить переводы этого клиента, чтобы понять, сток это или транзит.")
	case f.IsSeed:
		sb.WriteString("Seed: входящие занижены выгрузкой. Запросить входящие переводы извне выборки.")
	case f.NSeedUpstream >= 3:
		fmt.Fprintf(&sb, "К узлу ведут цепочки от %d seed — кандидат на углублённую проверку.", f.NSeedUpstream)
	default:
		sb.WriteString("Проверить контрагентов с наибольшими суммами и их роли.")
	}
	return sb.String()
}
