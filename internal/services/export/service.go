// Package export — выгрузки по схеме ТЗ: nodes_roles.csv, clusters.csv, top_nodes.csv и graph.json для UI.
package export

import (
	"os"

	"hackaton/internal/data/models"
	"hackaton/internal/services"

	"go.uber.org/zap"
)

type Service interface {
	WriteAll(result *models.AnalysisResult, dir string) error
	WriteCSV(result *models.AnalysisResult, dir string) error
	WriteGraphJSON(result *models.AnalysisResult, dir string) error
}

type service struct {
	logger *zap.Logger
}

type ServiceParams struct {
	services.FxBaseParams
}

func NewService(params ServiceParams) Service {
	return &service{logger: params.Logger.Named("export_service")}
}

func (s *service) WriteAll(result *models.AnalysisResult, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := s.WriteCSV(result, dir); err != nil {
		return err
	}
	return s.WriteGraphJSON(result, dir)
}
