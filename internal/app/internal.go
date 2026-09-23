package app

import (
	"context"
	"path/filepath"

	"hackaton/internal/config"
	"hackaton/internal/data/models"
	"hackaton/internal/repo/dataset"
	"hackaton/internal/services/analysis"
	"hackaton/internal/services/assistant"
	"hackaton/internal/services/export"
	"hackaton/internal/services/hypotheses"
	"hackaton/internal/services/pipeline"
	"hackaton/pkg/llm"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func ModuleRepositories() fx.Option {
	return fx.Provide(
		dataset.NewLoader,
	)
}

func ModuleServices() fx.Option {
	return fx.Provide(
		newLLMClient,
		newLLMCache,
		fx.Annotate(analysis.NewService, fx.As(new(analysis.Service))),
		fx.Annotate(export.NewService, fx.As(new(export.Service))),
		fx.Annotate(hypotheses.NewService, fx.As(new(hypotheses.Service))),
		fx.Annotate(pipeline.NewService, fx.As(new(pipeline.Service))),
		fx.Annotate(assistant.NewService, fx.As(new(assistant.Service))),
		newAnalysisResult,
	)
}

func newLLMClient(cfg *config.Config) *llm.Client {
	return llm.New(llm.Config{APIKey: cfg.LLM.APIKey, Model: cfg.LLM.Model, BaseURL: cfg.LLM.BaseURL})
}

func newLLMCache(cfg *config.Config) *llm.Cache {
	return llm.OpenCache(filepath.Join(cfg.App.OutDir, llm.CacheFileName))
}

// newAnalysisResult — один расчёт на процесс; от него зависят сервис графа и ассистент.
func newAnalysisResult(cfg *config.Config, log *zap.Logger, runner pipeline.Service) (*models.AnalysisResult, error) {
	report, err := runner.Run(context.Background(), pipeline.RunParams{DataDir: cfg.App.DataDir, OutDir: cfg.App.OutDir})
	if err != nil {
		return nil, err
	}
	log.Info("analysis ready", zap.Int("nodes", len(report.Result.Nodes)), zap.Int("clusters", len(report.Result.Clusters)),
		zap.Int("hypotheses_from_cache", report.HypothesesEnriched), zap.Duration("elapsed", report.Elapsed))
	return report.Result, nil
}
