package graph

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"hackaton/internal/config"
	graphservice "hackaton/internal/services/graph"
	"hackaton/pkg/httperr"
)

func TestAPI(t *testing.T) {
	var service graphservice.Service
	app := fx.New(fx.NopLogger, fx.Supply(&config.Config{App: config.AppConfig{DataDir: "../../../../../data"}}), fx.Provide(graphservice.NewService), fx.Populate(&service))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if e := app.Start(ctx); e != nil {
		t.Fatal(e)
	}
	defer app.Stop(ctx)
	h := NewHandler(service)
	api := fiber.New(fiber.Config{ErrorHandler: httperr.Handler(zap.NewNop())})
	api.Get("/graph", h.Graph)
	api.Get("/nodes/:gid", h.Node)
	api.Get("/nodes/:gid/ego", h.Ego)
	api.Get("/top", h.Top)
	api.Get("/clusters", h.Clusters)
	api.Get("/search", h.Search)
	for _, tc := range []struct {
		path   string
		status int
	}{{"/graph?top=30", 200}, {"/graph?cluster=0", 200}, {"/graph?role=nonsense", 400}, {"/graph?top=-1", 400}, {"/graph?component=abc", 400}, {"/nodes/abc", 400}, {"/nodes/99999999999999999999999", 400}, {"/nodes/1", 404}, {"/nodes/1/ego?depth=3", 400}, {"/top?n=0", 400}, {"/top", 200}, {"/clusters", 200}, {"/search?q=foo", 400}, {"/search?limit=101", 400}} {
		t.Run(tc.path, func(t *testing.T) {
			r, e := api.Test(httptest.NewRequest("GET", tc.path, nil))
			if e != nil {
				t.Fatal(e)
			}
			defer r.Body.Close()
			if r.StatusCode != tc.status {
				b, _ := io.ReadAll(r.Body)
				t.Fatalf("status %d: %s", r.StatusCode, b)
			}
		})
	}
	r, e := api.Test(httptest.NewRequest("GET", "/top", nil))
	if e != nil {
		t.Fatal(e)
	}
	defer r.Body.Close()
	var top []struct {
		Gid string `json:"gid"`
	}
	if e := json.NewDecoder(r.Body).Decode(&top); e != nil || len(top) != 30 {
		t.Fatalf("top: %v, %d", e, len(top))
	}
	for _, path := range []string{"/nodes/" + top[0].Gid, "/nodes/" + top[0].Gid + "/ego?depth=2", "/search?q=" + top[0].Gid} {
		r, e := api.Test(httptest.NewRequest("GET", path, nil))
		if e != nil {
			t.Fatal(e)
		}
		b, _ := io.ReadAll(r.Body)
		r.Body.Close()
		if r.StatusCode != 200 {
			t.Fatalf("%s: %d %s", path, r.StatusCode, b)
		}
	}
}
