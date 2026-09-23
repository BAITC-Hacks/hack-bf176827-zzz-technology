package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	graphdto "hackaton/internal/data/dto/graph"
	"hackaton/internal/data/models"
)

var (
	nodesHeader = []string{"gid", "role", "role_score", "cluster_id", "priority_score", "evidence",
		"in_deg", "out_deg", "in_kzt", "out_kzt", "in_tx", "out_tx", "pass_through", "pagerank", "betweenness",
		"n_seed_payers", "n_seed_upstream", "component_id", "depth", "is_seed", "truncated", "verified_sink",
		"in_cycle", "repeat_routes", "fast_forward_share", "active_days"}
	clustersHeader = []string{"cluster_id", "n_nodes", "n_seed", "sum_kzt_internal", "top_gids", "hypothesis",
		"component_id", "sum_kzt_in", "sum_kzt_out", "n_cycles",
		"n_consolidator", "n_transit", "n_distributor", "n_terminal", "n_coordinator", "n_peripheral"}
	topHeader = []string{"rank", "gid", "role", "priority_score", "why"}
)

func (s *service) WriteCSV(result *models.AnalysisResult, dir string) error {
	if err := writeCSV(filepath.Join(dir, "nodes_roles.csv"), nodesHeader, len(result.Nodes), func(i int) []string {
		return nodeRow(result.Nodes[i])
	}); err != nil {
		return err
	}
	if err := writeCSV(filepath.Join(dir, "clusters.csv"), clustersHeader, len(result.Clusters), func(i int) []string {
		return clusterRow(result.Clusters[i])
	}); err != nil {
		return err
	}
	return writeCSV(filepath.Join(dir, "top_nodes.csv"), topHeader, len(result.Top), func(i int) []string {
		top := result.Top[i]
		return []string{strconv.Itoa(top.Rank), graphdto.GID(top.GID), string(top.Role), formatScore(top.PriorityScore), top.Why}
	})
}

func nodeRow(node models.NodeResult) []string {
	f := node.Features
	return []string{
		graphdto.GID(node.GID), string(node.Role), formatScore(node.RoleScore), strconv.Itoa(node.ClusterID), formatScore(node.PriorityScore), node.Evidence,
		strconv.Itoa(f.InDegree), strconv.Itoa(f.OutDegree), formatKZT(f.InKZT), formatKZT(f.OutKZT), strconv.Itoa(f.InTxCount), strconv.Itoa(f.OutTxCount),
		formatScore(f.PassThrough), formatFloat(f.PageRank), formatFloat(f.Betweenness),
		strconv.Itoa(f.SeedPayers), strconv.Itoa(f.SeedUpstream), strconv.Itoa(f.ComponentID), strconv.Itoa(f.Depth),
		strconv.FormatBool(f.IsSeed), strconv.FormatBool(f.Truncated), strconv.FormatBool(f.VerifiedSink),
		strconv.FormatBool(f.InCycle), strconv.Itoa(f.RepeatRoutes), formatScore(f.FastForwardShare), strconv.Itoa(f.ActiveDays),
	}
}

func clusterRow(cluster models.ClusterResult) []string {
	roles := cluster.RoleCounts
	return []string{
		strconv.Itoa(cluster.ClusterID), strconv.Itoa(cluster.NodeCount), strconv.Itoa(cluster.SeedCount), formatKZT(cluster.SumKZTInternal),
		strings.Join(graphdto.GIDs(cluster.TopGIDs), ";"), cluster.Hypothesis,
		strconv.Itoa(cluster.ComponentID), formatKZT(cluster.SumKZTIn), formatKZT(cluster.SumKZTOut), strconv.Itoa(cluster.CycleCount),
		strconv.Itoa(roles[models.RoleConsolidator]), strconv.Itoa(roles[models.RoleTransit]), strconv.Itoa(roles[models.RoleDistributor]),
		strconv.Itoa(roles[models.RoleTerminal]), strconv.Itoa(roles[models.RoleCoordinator]), strconv.Itoa(roles[models.RolePeripheral]),
	}
}

func writeCSV(path string, header []string, rows int, row func(i int) []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	if err := writer.Write(header); err != nil {
		return err
	}
	for i := 0; i < rows; i++ {
		if err := writer.Write(row(i)); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func formatScore(value float64) string { return strconv.FormatFloat(value, 'f', 4, 64) }
func formatKZT(value float64) string   { return strconv.FormatFloat(value, 'f', 0, 64) }
func formatFloat(value float64) string { return strconv.FormatFloat(value, 'g', 6, 64) }
