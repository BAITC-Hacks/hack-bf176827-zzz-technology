package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func fixture(t *testing.T, mutate func(string, [][]string) [][]string) string {
	t.Helper()
	dir := t.TempDir()
	nodes := [][]string{{"gid", "role", "role_score", "cluster_id", "priority_score", "evidence"}}
	for i := 0; i < 2248; i++ {
		nodes = append(nodes, []string{strconv.FormatInt(100000003684369100+int64(i), 10), "peripheral", "0.5", "1", "0.5", "Пример"})
	}
	top := [][]string{{"rank", "gid", "role", "priority_score", "why"}}
	for i := 1; i <= 20; i++ {
		top = append(top, []string{strconv.Itoa(i), nodes[i][0], "peripheral", "0.5", "Пример"})
	}
	for name, rows := range map[string][][]string{"nodes_roles.csv": nodes, "clusters.csv": {{"cluster_id", "n_nodes", "n_seed", "sum_kzt_internal", "top_gids", "hypothesis"}, {"1", "2248", "0", "0", nodes[1][0], "Пример"}}, "top_nodes.csv": top} {
		if mutate != nil {
			rows = mutate(name, rows)
		}
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		w := csv.NewWriter(f)
		w.WriteAll(rows)
		if err := w.Error(); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
func TestValidate(t *testing.T) {
	tests := []struct {
		name, file string
		change     func([][]string) [][]string
		want       string
	}{
		{name: "valid"},
		{"duplicate", "nodes_roles.csv", func(r [][]string) [][]string { r[2][0] = r[1][0]; return r }, "строка 3: повтор gid"},
		{"nan", "nodes_roles.csv", func(r [][]string) [][]string { r[1][2] = "NaN"; return r }, "role_score"},
		{"unicode boundary", "nodes_roles.csv", func(r [][]string) [][]string { r[1][5] = strings.Repeat("я", 200); return r }, ""},
		{"long evidence", "nodes_roles.csv", func(r [][]string) [][]string { r[1][5] = strings.Repeat("я", 201); return r }, "evidence"},
		{"missing cluster", "clusters.csv", func(r [][]string) [][]string { r[1][0] = "2"; return r }, "отсутствует cluster_id 1"},
		{"rank gap", "top_nodes.csv", func(r [][]string) [][]string { r[2][0] = "3"; return r }, "ожидается rank 2"},
		{"unsorted", "top_nodes.csv", func(r [][]string) [][]string { r[2][3] = "0.7"; return r }, "невозрастать"},
		{"short top", "top_nodes.csv", func(r [][]string) [][]string { return r[:20] }, "минимум 20"},
		{"short nodes", "nodes_roles.csv", func(r [][]string) [][]string { return r[:2248] }, "ожидается 2248"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := fixture(t, func(name string, r [][]string) [][]string {
				if name == tc.file {
					return tc.change(r)
				}
				return r
			})
			_, err := validate(dir)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ожидалось %q, получено %v", tc.want, err)
			}
		})
	}
}
