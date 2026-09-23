package analysis

import (
	"reflect"
	"testing"
	"unicode/utf8"

	"hackaton/internal/data/models"
	"hackaton/internal/repo/dataset"
	"hackaton/internal/services"
	analysisservice "hackaton/internal/services/analysis"

	"go.uber.org/zap"
)

const expectedNodes = 2248

func analyze(t *testing.T) *models.AnalysisResult {
	t.Helper()
	ds, err := dataset.NewLoader().Load("../../../data")
	if err != nil {
		t.Skip("нет данных:", err)
	}
	if _, err := dataset.Validate(ds); err != nil {
		t.Fatal(err)
	}
	svc := analysisservice.NewService(analysisservice.ServiceParams{FxBaseParams: services.FxBaseParams{Logger: zap.NewNop()}})
	result, err := svc.Analyze(t.Context(), ds, analysisservice.Params{})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// Требования ТЗ к nodes_roles / clusters / top_nodes.
func TestResultMeetsSpec(t *testing.T) {
	result := analyze(t)
	if len(result.Nodes) != expectedNodes {
		t.Fatalf("узлов %d, ожидалось %d", len(result.Nodes), expectedNodes)
	}
	clusters := map[int]bool{}
	for _, cluster := range result.Clusters {
		clusters[cluster.ClusterID] = true
	}
	for _, node := range result.Nodes {
		if !models.IsValidRole(string(node.Role)) {
			t.Fatalf("%d: роль %q вне словаря", node.GID, node.Role)
		}
		if node.RoleScore < 0 || node.RoleScore > 1 || node.PriorityScore < 0 || node.PriorityScore > 1 {
			t.Fatalf("%d: скоры вне [0,1]", node.GID)
		}
		if node.Evidence == "" || utf8.RuneCountInString(node.Evidence) > 200 {
			t.Fatalf("%d: evidence пустой или длиннее 200: %q", node.GID, node.Evidence)
		}
		if !clusters[node.ClusterID] {
			t.Fatalf("%d: cluster_id %d нет в clusters", node.GID, node.ClusterID)
		}
		if node.Features.Truncated && node.Role == models.RoleTerminal {
			t.Fatalf("%d: обрезанный обходом узел помечен terminal", node.GID)
		}
		if node.Features.IsSeed && node.Role == models.RoleTransit {
			t.Fatalf("%d: seed помечен transit", node.GID)
		}
	}
	if len(result.Top) < 20 {
		t.Fatalf("top %d < 20", len(result.Top))
	}
	for i := 1; i < len(result.Top); i++ {
		if result.Top[i].PriorityScore > result.Top[i-1].PriorityScore || result.Top[i].Rank != i+1 {
			t.Fatalf("top не отсортирован на позиции %d", i)
		}
	}
}

func TestAnalysisIsDeterministic(t *testing.T) {
	first, second := analyze(t), analyze(t)
	if !reflect.DeepEqual(first.Nodes, second.Nodes) || !reflect.DeepEqual(first.Clusters, second.Clusters) || !reflect.DeepEqual(first.Top, second.Top) {
		t.Fatal("два прогона дали разный результат")
	}
}
