package graph

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"hackaton/internal/repo/dataset"
	"hackaton/internal/services"
	analysisservice "hackaton/internal/services/analysis"
	graphservice "hackaton/internal/services/graph"
	"hackaton/pkg/httperr"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func newTestAPI(t *testing.T) *fiber.App {
	t.Helper()
	ds, err := dataset.NewLoader().Load("../../../../../data")
	if err != nil {
		t.Skip("нет данных:", err)
	}
	base := services.FxBaseParams{Logger: zap.NewNop()}
	result, err := analysisservice.NewService(analysisservice.ServiceParams{FxBaseParams: base}).Analyze(t.Context(), ds, analysisservice.Params{})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(HandlerParams{Logger: zap.NewNop(), Graph: graphservice.NewService(graphservice.ServiceParams{FxBaseParams: base, Result: result})})
	api := fiber.New(fiber.Config{ErrorHandler: httperr.Handler(zap.NewNop())})
	api.Get("/graph", h.Graph)
	api.Get("/nodes/:gid", h.Node)
	api.Get("/nodes/:gid/ego", h.Ego)
	api.Get("/top", h.Top)
	api.Get("/clusters", h.Clusters)
	api.Get("/search", h.Search)
	return api
}

func TestAPI(t *testing.T) {
	api := newTestAPI(t)
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/graph?top=30", 200}, {"/graph?cluster=0", 200}, {"/graph?role=nonsense", 400}, {"/graph?top=-1", 400}, {"/graph?component=abc", 400},
		{"/nodes/abc", 400}, {"/nodes/99999999999999999999999", 400}, {"/nodes/1", 404}, {"/nodes/1/ego?depth=3", 400},
		{"/top?n=0", 400}, {"/top", 200}, {"/clusters", 200}, {"/search?q=foo", 400}, {"/search?limit=101", 400},
	} {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := api.Test(httptest.NewRequest("GET", tc.path, nil))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.status {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("status %d: %s", resp.StatusCode, body)
			}
		})
	}

	resp, err := api.Test(httptest.NewRequest("GET", "/top", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var top []struct {
		Gid string `json:"gid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&top); err != nil || len(top) != 30 {
		t.Fatalf("top: %v, %d", err, len(top))
	}
	card, err := api.Test(httptest.NewRequest("GET", "/nodes/"+top[0].Gid, nil))
	if err != nil || card.StatusCode != 200 {
		t.Fatalf("card: %v %d", err, card.StatusCode)
	}
	var parsed struct {
		Node struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"node"`
		Incoming []struct {
			Gid string `json:"gid"`
		} `json:"incoming"`
	}
	if err := json.NewDecoder(card.Body).Decode(&parsed); err != nil || parsed.Node.ID != top[0].Gid || parsed.Node.Role == "" || len(parsed.Incoming) == 0 {
		t.Fatalf("card body: %v %+v", err, parsed)
	}
}
