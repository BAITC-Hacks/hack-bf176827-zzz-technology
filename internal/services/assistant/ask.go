package assistant

import (
	"context"
	"strings"

	"hackaton/pkg/llm"

	"go.uber.org/zap"
)

func (s *service) Ask(ctx context.Context, question string) (Answer, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return Answer{}, ErrEmptyQuestion
	}
	cacheKey := llm.Key("ask", s.llm.Model(), question)
	if text, ok := s.cache.Get(cacheKey); ok {
		return Answer{Text: text, GIDs: extractGIDs(text), Cached: true}, nil
	}
	if !s.llm.Enabled() {
		return Answer{}, ErrLLMDisabled
	}
	reply, err := s.llm.RunTools(ctx, systemPrompt, question, toolDefinitions(), s.executeTool, maxToolSteps)
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
