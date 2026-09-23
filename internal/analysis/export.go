package analysis

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func gidStr(g int64) string { return strconv.FormatInt(g, 10) }
func f2(x float64) string   { return strconv.FormatFloat(x, 'f', 4, 64) }
func f0(x float64) string   { return strconv.FormatFloat(x, 'f', 0, 64) }

// WriteCSV — nodes_roles.csv, clusters.csv, top_nodes.csv (схема ТЗ + доп. колонки).
func WriteCSV(res *Result, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := writeCSV(filepath.Join(dir, "nodes_roles.csv"),
		[]string{"gid", "role", "role_score", "cluster_id", "priority_score", "evidence",
			"in_deg", "out_deg", "in_kzt", "out_kzt", "in_tx", "out_tx", "pass_through", "pagerank", "betweenness",
			"n_seed_payers", "n_seed_upstream", "component_id", "depth", "is_seed", "truncated", "verified_sink"},
		len(res.Nodes), func(i int) []string {
			n := res.Nodes[i]
			f := n.Features
			return []string{gidStr(n.Gid), n.Role, f2(n.RoleScore), strconv.Itoa(n.ClusterID), f2(n.PriorityScore), n.Evidence,
				strconv.Itoa(f.InDeg), strconv.Itoa(f.OutDeg), f0(f.InKZT), f0(f.OutKZT), strconv.Itoa(f.InTx), strconv.Itoa(f.OutTx),
				f2(f.PassThrough), strconv.FormatFloat(f.PageRank, 'g', 6, 64), strconv.FormatFloat(f.Betweenness, 'g', 6, 64),
				strconv.Itoa(f.NSeedPayers), strconv.Itoa(f.NSeedUpstream), strconv.Itoa(f.ComponentID), strconv.Itoa(f.Depth),
				strconv.FormatBool(f.IsSeed), strconv.FormatBool(f.Truncated), strconv.FormatBool(f.VerifiedSink)}
		}); err != nil {
		return err
	}
	if err := writeCSV(filepath.Join(dir, "clusters.csv"),
		[]string{"cluster_id", "n_nodes", "n_seed", "sum_kzt_internal", "top_gids", "hypothesis",
			"component_id", "sum_kzt_in", "sum_kzt_out", "n_consolidator", "n_transit", "n_distributor", "n_terminal", "n_coordinator", "n_peripheral"},
		len(res.Clusters), func(i int) []string {
			c := res.Clusters[i]
			gids := make([]string, len(c.TopGids))
			for j, g := range c.TopGids {
				gids[j] = gidStr(g)
			}
			return []string{strconv.Itoa(c.ClusterID), strconv.Itoa(c.NNodes), strconv.Itoa(c.NSeed), f0(c.SumKZTInternal),
				strings.Join(gids, ";"), c.Hypothesis, strconv.Itoa(c.ComponentID), f0(c.SumKZTIn), f0(c.SumKZTOut),
				strconv.Itoa(c.RoleCounts[RoleConsolidator]), strconv.Itoa(c.RoleCounts[RoleTransit]), strconv.Itoa(c.RoleCounts[RoleDistributor]),
				strconv.Itoa(c.RoleCounts[RoleTerminal]), strconv.Itoa(c.RoleCounts[RoleCoordinator]), strconv.Itoa(c.RoleCounts[RolePeripheral])}
		}); err != nil {
		return err
	}
	return writeCSV(filepath.Join(dir, "top_nodes.csv"),
		[]string{"rank", "gid", "role", "priority_score", "why"},
		len(res.Top), func(i int) []string {
			t := res.Top[i]
			return []string{strconv.Itoa(t.Rank), gidStr(t.Gid), t.Role, f2(t.PriorityScore), t.Why}
		})
}

func writeCSV(path string, header []string, n int, row func(i int) []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write(header); err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		if err := w.Write(row(i)); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// JSON для UI: gid строкой (≈1e17 не влезает в JS Number).
type GraphJSON struct {
	Nodes    []NodeJSON    `json:"nodes"`
	Edges    []EdgeJSON    `json:"edges"`
	Clusters []ClusterJSON `json:"clusters"`
	Top      []TopJSON     `json:"top"`
}

type NodeJSON struct {
	ID        string  `json:"id"`
	Role      string  `json:"role"`
	RoleScore float64 `json:"role_score"`
	Cluster   int     `json:"cluster"`
	Priority  float64 `json:"priority"`
	IsSeed    bool    `json:"is_seed"`
	Depth     int     `json:"depth"`
	Evidence  string  `json:"evidence"`
	InDeg     int     `json:"in_deg"`
	OutDeg    int     `json:"out_deg"`
	InKZT     float64 `json:"in_kzt"`
	OutKZT    float64 `json:"out_kzt"`
	Truncated bool    `json:"truncated"`
	Component int     `json:"component"`
}

type EdgeJSON struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	SumKZT float64 `json:"sum_kzt"`
	NTx    int64   `json:"n_tx"`
}

type ClusterJSON struct {
	ID             int      `json:"id"`
	NNodes         int      `json:"n_nodes"`
	NSeed          int      `json:"n_seed"`
	SumKZTInternal float64  `json:"sum_kzt_internal"`
	TopGids        []string `json:"top_gids"`
	Hypothesis     string   `json:"hypothesis"`
}

type TopJSON struct {
	Rank     int     `json:"rank"`
	Gid      string  `json:"gid"`
	Role     string  `json:"role"`
	Priority float64 `json:"priority"`
	Why      string  `json:"why"`
}

func ToGraphJSON(res *Result) GraphJSON {
	out := GraphJSON{
		Nodes:    make([]NodeJSON, 0, len(res.Nodes)),
		Edges:    make([]EdgeJSON, 0, len(res.Edges)),
		Clusters: make([]ClusterJSON, 0, len(res.Clusters)),
		Top:      make([]TopJSON, 0, len(res.Top)),
	}
	for _, n := range res.Nodes {
		f := n.Features
		out.Nodes = append(out.Nodes, NodeJSON{ID: gidStr(n.Gid), Role: n.Role, RoleScore: n.RoleScore, Cluster: n.ClusterID,
			Priority: n.PriorityScore, IsSeed: f.IsSeed, Depth: f.Depth, Evidence: n.Evidence, InDeg: f.InDeg, OutDeg: f.OutDeg,
			InKZT: f.InKZT, OutKZT: f.OutKZT, Truncated: f.Truncated, Component: f.ComponentID})
	}
	for _, e := range res.Edges {
		out.Edges = append(out.Edges, EdgeJSON{Source: gidStr(e.Src), Target: gidStr(e.Dst), SumKZT: e.SumKZT, NTx: e.NTx})
	}
	for _, c := range res.Clusters {
		gids := make([]string, len(c.TopGids))
		for j, g := range c.TopGids {
			gids[j] = gidStr(g)
		}
		out.Clusters = append(out.Clusters, ClusterJSON{ID: c.ClusterID, NNodes: c.NNodes, NSeed: c.NSeed, SumKZTInternal: c.SumKZTInternal, TopGids: gids, Hypothesis: c.Hypothesis})
	}
	for _, t := range res.Top {
		out.Top = append(out.Top, TopJSON{Rank: t.Rank, Gid: gidStr(t.Gid), Role: t.Role, Priority: t.PriorityScore, Why: t.Why})
	}
	return out
}

func WriteGraphJSON(res *Result, dir string) error {
	b, err := json.Marshal(ToGraphJSON(res))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "graph.json"), b, 0o644)
}
