package export

import (
	"encoding/json"
	"os"
	"path/filepath"

	graphdto "hackaton/internal/data/dto/graph"
	"hackaton/internal/data/models"
)

func (s *service) WriteGraphJSON(result *models.AnalysisResult, dir string) error {
	payload, err := json.Marshal(graphdto.NewGraphResponse(result))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "graph.json"), payload, 0o644)
}
