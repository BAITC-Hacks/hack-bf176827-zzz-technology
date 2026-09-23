package assistant

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"hackaton/internal/repo/dataset"
	"hackaton/internal/services"
	"hackaton/internal/services/analysis"
	"hackaton/pkg/llm"

	"go.uber.org/zap"
)

func newTestService(t *testing.T) *service {
	t.Helper()
	ds, err := dataset.NewLoader().Load("../../../data")
	if err != nil {
		t.Skip("нет данных:", err)
	}
	base := services.FxBaseParams{Logger: zap.NewNop()}
	result, err := analysis.NewService(analysis.ServiceParams{FxBaseParams: base}).Analyze(t.Context(), ds, analysis.Params{})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(ServiceParams{FxBaseParams: base, Result: result, LLM: llm.New(llm.Config{}), Cache: llm.OpenCache(t.TempDir() + "/cache.json")})
	return svc.(*service)
}

func TestToolsWithoutLLM(t *testing.T) {
	svc := newTestService(t)
	top := `"` + strings.TrimSpace(jsonGID(svc.result.Top[0].GID)) + `"`
	cases := map[string]string{
		"get_node":          `{"gid":` + top + `}`,
		"find_nodes":        `{"role":"consolidator","limit":5}`,
		"who_receives_from": `{"gids":[` + top + `],"depth":2}`,
		"who_pays_to":       `{"gid":` + top + `,"depth":2}`,
		"path":              `{"src":` + top + `,"dst":` + top + `,"max_len":3}`,
		"cluster_info":      `{"cluster_id":0}`,
		"top":               `{"n":5}`,
		"remove_nodes":      `{"gids":[` + top + `]}`,
	}
	for name, args := range cases {
		out, err := svc.executeTool(name, args)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !json.Valid([]byte(out)) || len(out) < 10 {
			t.Fatalf("%s: bad output %q", name, out)
		}
	}
	if _, err := svc.executeTool("get_node", `{"gid":"1"}`); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("ожидался ErrNodeNotFound, получено %v", err)
	}
	if _, err := svc.executeTool("nope", `{}`); !errors.Is(err, ErrUnknownTool) {
		t.Fatalf("ожидался ErrUnknownTool, получено %v", err)
	}

	card, err := svc.Card(t.Context(), svc.result.Top[0].GID)
	if err != nil || card.ByLLM || !strings.Contains(card.Text, "Роль и почему") {
		t.Fatalf("card: %v %+v", err, card)
	}
	if _, err := svc.Ask(t.Context(), "кто главный?", 0); !errors.Is(err, ErrLLMDisabled) {
		t.Fatalf("ожидался ErrLLMDisabled, получено %v", err)
	}
}

func jsonGID(gid int64) string {
	encoded, _ := json.Marshal(gid)
	return string(encoded)
}
