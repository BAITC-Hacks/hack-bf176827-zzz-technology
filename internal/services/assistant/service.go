// Package assistant — LLM-помощник аналитика поверх результата анализа: вопрос-ответ через инструменты по графу
// и справка по узлу. Роли и приоритеты не трогает — только формулирует текст по посчитанным фактам.
package assistant

import (
	"context"

	"hackaton/internal/data/graph"
	"hackaton/internal/data/models"
	"hackaton/internal/services"
	"hackaton/pkg/llm"

	"go.uber.org/zap"
)

const maxToolSteps = 8

type Service interface {
	Enabled() bool
	Model() string
	// Ask — вопрос на естественном языке. contextGID (0 — нет) — открытый узел, route — путь просмотра (только контекст).
	// Без ключа — ErrLLMDisabled.
	Ask(ctx context.Context, question string, contextGID int64, route []int64) (Answer, error)
	// Card — справка по узлу; без ключа или при сбое LLM возвращает шаблон и ByLLM=false.
	Card(ctx context.Context, gid int64) (Card, error)
}

type service struct {
	logger *zap.Logger
	result *models.AnalysisResult
	index  *graph.Index
	llm    *llm.Client
	cache  *llm.Cache
}

type ServiceParams struct {
	services.FxBaseParams
	Result *models.AnalysisResult
	LLM    *llm.Client
	Cache  *llm.Cache
}

func NewService(params ServiceParams) Service {
	return &service{
		logger: params.Logger.Named("assistant_service"),
		result: params.Result,
		index:  graph.NewIndex(params.Result.Edges),
		llm:    params.LLM,
		cache:  params.Cache,
	}
}

func (s *service) Enabled() bool { return s.llm.Enabled() }
func (s *service) Model() string { return s.llm.Model() }
