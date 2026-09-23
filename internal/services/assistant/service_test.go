package assistant

import (
	"encoding/json"
	"strings"
	"testing"

	"hackaton/internal/analysis"
	"hackaton/internal/data/parquet"
)

func load(t *testing.T) *Service {
	t.Helper()
	ds, err := parquet.Load("../../../data")
	if err != nil {
		t.Skip("нет данных:", err)
	}
	res, err := analysis.Run(ds, analysis.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return New(res, nil, nil)
}

func TestTools(t *testing.T) {
	s := load(t)
	top := s.res.Top[0].Gid
	gid := strings.TrimSpace(strings.Repeat(" ", 1) + jsonStr(top))
	cases := map[string]string{
		"get_node":          `{"gid":` + gid + `}`,
		"find_nodes":        `{"role":"consolidator","limit":5}`,
		"who_receives_from": `{"gids":[` + gid + `],"depth":2}`,
		"who_pays_to":       `{"gid":` + gid + `,"depth":2}`,
		"cluster_info":      `{"cluster_id":0}`,
		"top":               `{"n":5}`,
		"remove_nodes":      `{"gids":[` + gid + `]}`,
	}
	for name, args := range cases {
		out, err := s.exec(name, args)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !json.Valid([]byte(out)) || len(out) < 10 {
			t.Fatalf("%s: bad output %q", name, out)
		}
	}
	if _, err := s.exec("get_node", `{"gid":"1"}`); err == nil {
		t.Fatal("ожидалась ошибка для несуществующего gid")
	}
	if text, gen, err := s.Card(t.Context(), strings.Trim(gid, `"`)); err != nil || gen || !strings.Contains(text, "Роль") {
		t.Fatalf("card: %v %v %q", err, gen, text)
	}
}

func jsonStr(g int64) string {
	b, _ := json.Marshal(g)
	return `"` + string(b) + `"`
}
