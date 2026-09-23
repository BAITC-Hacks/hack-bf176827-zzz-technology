// Package pipeline — оркестрация: загрузить parquet → проверить → посчитать → обогатить гипотезы → записать выгрузки.
// Используется CLI (cmd/pipeline, cmd/ask) и веб-сервером на старте.
package pipeline

import (
	"context"
	"fmt"
	"time"

	"hackaton/internal/data/models"
	"hackaton/internal/repo/dataset"
	"hackaton/internal/services"
	"hackaton/internal/services/analysis"
	"hackaton/internal/services/export"
	"hackaton/internal/services/hypotheses"

	"go.uber.org/zap"
)

type Service interface {
	Run(ctx context.Context, params RunParams) (*Report, error)
}

type RunParams struct {
	DataDir      string
	OutDir       string
	TopN         int
	UseLLM       bool // дозапрашивать у LLM недостающие гипотезы (нужен ключ); без флага — только кэш
	WriteOutputs bool // писать CSV и graph.json в OutDir
}

// Report — результат прогона и сводка для лога.
type Report struct {
	Result             *models.AnalysisResult
	Stats              models.DatasetStats
	HypothesesEnriched int
	Elapsed            time.Duration
}

type service struct {
	logger     *zap.Logger
	loader     dataset.Loader
	analysis   analysis.Service
	hypotheses hypotheses.Service
	export     export.Service
}

type ServiceParams struct {
	services.FxBaseParams
	Loader     dataset.Loader
	Analysis   analysis.Service
	Hypotheses hypotheses.Service
	Export     export.Service
}

func NewService(params ServiceParams) Service {
	return &service{
		logger:     params.Logger.Named("pipeline_service"),
		loader:     params.Loader,
		analysis:   params.Analysis,
		hypotheses: params.Hypotheses,
		export:     params.Export,
	}
}

func (s *service) Run(ctx context.Context, params RunParams) (*Report, error) {
	started := time.Now()
	ds, err := s.loader.Load(params.DataDir)
	if err != nil {
		return nil, fmt.Errorf("загрузка данных: %w", err)
	}
	stats, err := dataset.Validate(ds)
	if err != nil {
		return nil, fmt.Errorf("проверка данных: %w", err)
	}
	result, err := s.analysis.Analyze(ctx, ds, analysis.Params{TopN: params.TopN})
	if err != nil {
		return nil, fmt.Errorf("анализ: %w", err)
	}
	enriched, err := s.hypotheses.Enrich(ctx, result, params.UseLLM)
	if err != nil {
		// гипотезы — необязательный слой: шаблоны уже на месте
		s.logger.Warn("hypotheses not enriched", zap.Error(err))
	}
	if params.WriteOutputs {
		if err := s.export.WriteAll(result, params.OutDir); err != nil {
			return nil, fmt.Errorf("выгрузки: %w", err)
		}
	}
	return &Report{Result: result, Stats: stats, HypothesesEnriched: enriched, Elapsed: time.Since(started)}, nil
}
