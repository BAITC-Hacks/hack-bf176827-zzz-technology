package assistant

import (
	"context"
	"fmt"
	"strings"

	"hackaton/pkg/llm"

	"go.uber.org/zap"
)

func (s *service) Ask(ctx context.Context, question string, contextGID int64) (Answer, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return Answer{}, ErrEmptyQuestion
	}
	prompt := s.withContext(question, contextGID)
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

// withContext — вопрос с выбранным узлом: «этот узел» в вопросе относится к нему.
func (s *service) withContext(question string, gid int64) string {
	node, ok := s.result.Node(gid)
	if gid == 0 || !ok {
		return question
	}
	return fmt.Sprintf("Контекст: у аналитика открыт узел gid %d (роль %s, приоритет %.3f). Слова «этот узел», «он», «данный клиент» относятся к нему. Начни с get_node для этого gid.\n\nВопрос: %s",
		gid, node.Role, node.PriorityScore, question)
}
