package assistant

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"hackaton/pkg/llm"

	"go.uber.org/zap"
)

func (s *service) Ask(ctx context.Context, question string, contextGID int64, route []int64) (Answer, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return Answer{}, ErrEmptyQuestion
	}
	prompt := s.withContext(question, contextGID, route)
	cacheKey := llm.Key("ask", s.llm.Model(), prompt)
	if text, ok := s.cache.Get(cacheKey); ok {
		return Answer{Text: text, GIDs: extractGIDs(text), Cached: true}, nil
	}
	if !s.llm.Enabled() {
		return Answer{}, ErrLLMDisabled
	}
	reply, err := s.llm.RunTools(ctx, systemPrompt, prompt, toolDefinitions(), s.executeTool, maxToolSteps)
	if err != nil {
		s.logger.Error("failed to answer question", zap.Error(err))
		return Answer{}, err
	}
	s.cache.Put(cacheKey, reply.Text)
	if err := s.cache.Save(); err != nil {
		s.logger.Warn("failed to save llm cache", zap.Error(err))
	}
	s.logger.Info("question answered", zap.Int("steps", reply.Steps),
		zap.Int("input_tokens", reply.Usage.InputTokens), zap.Int("output_tokens", reply.Usage.OutputTokens))
	return Answer{Text: reply.Text, GIDs: extractGIDs(reply.Text), Steps: reply.Steps}, nil
}

// withContext — вопрос с открытым узлом и путём просмотра. Путь — это лишь то, что аналитик открывал,
// не утверждение о связях и не источник истины.
func (s *service) withContext(question string, gid int64, route []int64) string {
	var parts []string
	if node, ok := s.result.Node(gid); ok && gid != 0 {
		parts = append(parts, fmt.Sprintf("Контекст: у аналитика открыт узел gid %d (роль %s, приоритет %.3f). Слова «этот узел», «он», «данный клиент» относятся к нему. Начни с get_node для этого gid.",
			gid, node.Role, node.PriorityScore))
	}
	if len(route) > 0 {
		gids := make([]string, 0, len(route))
		for _, r := range route {
			if _, ok := s.result.Node(r); ok {
				gids = append(gids, strconv.FormatInt(r, 10))
			}
		}
		if len(gids) > 0 {
			parts = append(parts, "Путь просмотра аналитика (в порядке открытия): "+strings.Join(gids, " → ")+
				". Это просто узлы, которые пользователь по очереди открывал и смотрел. Используй как подсказку о его интересе, но это не источник истины: "+
				"не считай путь доказанной цепочкой переводов и не строй на нём выводы без проверки инструментами.")
		}
	}
	if len(parts) == 0 {
		return question
	}
	return strings.Join(parts, "\n\n") + "\n\nВопрос: " + question
}
