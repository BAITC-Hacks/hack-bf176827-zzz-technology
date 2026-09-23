// Package graphdto — JSON-контракт графа для UI и API. GID всегда строкой: ≈1e17 не влезает в JS Number.
package graphdto

import (
	"strconv"

	"hackaton/internal/data/models"
)

type GraphResponse struct {
	Nodes      []NodeResponse       `json:"nodes"`
	Edges      []EdgeResponse       `json:"edges"`
	Clusters   []ClusterResponse    `json:"clusters"`
	Top        []TopNodeResponse    `json:"top"`
	Robustness []RobustnessResponse `json:"robustness"`
}

type NodeResponse struct {
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

type EdgeResponse struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	SumKZT float64 `json:"sum_kzt"`
	NTx    int64   `json:"n_tx"`
}

type ClusterResponse struct {
	ID             int            `json:"id"`
	Component      int            `json:"component"`
	NNodes         int            `json:"n_nodes"`
	NSeed          int            `json:"n_seed"`
	SumKZTInternal float64        `json:"sum_kzt_internal"`
	SumKZTIn       float64        `json:"sum_kzt_in"`
	SumKZTOut      float64        `json:"sum_kzt_out"`
	NCycles        int            `json:"n_cycles"`
	Roles          map[string]int `json:"roles"`
	TopGids        []string       `json:"top_gids"`
	Hypothesis     string         `json:"hypothesis"`
}

type TopNodeResponse struct {
	Rank     int     `json:"rank"`
	Gid      string  `json:"gid"`
	Role     string  `json:"role"`
	Priority float64 `json:"priority"`
	Why      string  `json:"why"`
}

type RobustnessResponse struct {
	Removed              int      `json:"removed"`
	RemovedGids          []string `json:"removed_gids"`
	LostTurnoverShare    float64  `json:"lost_turnover_share"`
	ComponentsBefore     int      `json:"components_before"`
	ComponentsAfter      int      `json:"components_after"`
	LargestBefore        int      `json:"largest_component_before"`
	LargestAfter         int      `json:"largest_component_after"`
	NodesLostAllIncoming int      `json:"nodes_lost_all_incoming"`
	SeedsDisconnected    int      `json:"seeds_disconnected"`
}

// FeaturesResponse — все метрики узла; ключи совпадают с колонками nodes_roles.csv.
type FeaturesResponse struct {
	Depth            int     `json:"depth"`
	IsSeed           bool    `json:"is_seed"`
	InDeg            int     `json:"in_deg"`
	OutDeg           int     `json:"out_deg"`
	InKZT            float64 `json:"in_kzt"`
	OutKZT           float64 `json:"out_kzt"`
	InTx             int     `json:"in_tx"`
	OutTx            int     `json:"out_tx"`
	AvgInTx          float64 `json:"avg_in_tx"`
	AvgOutTx         float64 `json:"avg_out_tx"`
	PassThrough      float64 `json:"pass_through"`
	PageRank         float64 `json:"pagerank"`
	Hub              float64 `json:"hub"`
	Authority        float64 `json:"authority"`
	Betweenness      float64 `json:"betweenness"`
	NSeedPayers      int     `json:"n_seed_payers"`
	NSeedUpstream    int     `json:"n_seed_upstream"`
	NClustersIn      int     `json:"n_clusters_in"`
	ComponentID      int     `json:"component_id"`
	ComponentSize    int     `json:"component_size"`
	Truncated        bool    `json:"truncated"`
	VerifiedSink     bool    `json:"verified_sink"`
	ActiveDays       int     `json:"active_days"`
	MaxSameDayPayers int     `json:"max_same_day_payers"`
	FastForwardShare float64 `json:"fast_forward_share"`
	Reciprocal       bool    `json:"reciprocal"`
	InCycle          bool    `json:"in_cycle"`
	RepeatRoutes     int     `json:"repeat_routes"`
}

func GID(gid int64) string { return strconv.FormatInt(gid, 10) }

func GIDs(gids []int64) []string {
	out := make([]string, len(gids))
	for i, gid := range gids {
		out[i] = GID(gid)
	}
	return out
}

func NewGraphResponse(result *models.AnalysisResult) GraphResponse {
	resp := GraphResponse{
		Nodes:      make([]NodeResponse, 0, len(result.Nodes)),
		Edges:      make([]EdgeResponse, 0, len(result.Edges)),
		Clusters:   make([]ClusterResponse, 0, len(result.Clusters)),
		Top:        make([]TopNodeResponse, 0, len(result.Top)),
		Robustness: make([]RobustnessResponse, 0, len(result.Robustness)),
	}
	for _, node := range result.Nodes {
		resp.Nodes = append(resp.Nodes, NewNodeResponse(node))
	}
	for _, edge := range result.Edges {
		resp.Edges = append(resp.Edges, NewEdgeResponse(edge))
	}
	for _, cluster := range result.Clusters {
		resp.Clusters = append(resp.Clusters, NewClusterResponse(cluster))
	}
	for _, top := range result.Top {
		resp.Top = append(resp.Top, NewTopNodeResponse(top))
	}
	for _, step := range result.Robustness {
		resp.Robustness = append(resp.Robustness, NewRobustnessResponse(step))
	}
	return resp
}

func NewNodeResponse(node models.NodeResult) NodeResponse {
	f := node.Features
	return NodeResponse{ID: GID(node.GID), Role: string(node.Role), RoleScore: node.RoleScore, Cluster: node.ClusterID,
		Priority: node.PriorityScore, IsSeed: f.IsSeed, Depth: f.Depth, Evidence: node.Evidence, InDeg: f.InDegree, OutDeg: f.OutDegree,
		InKZT: f.InKZT, OutKZT: f.OutKZT, Truncated: f.Truncated, Component: f.ComponentID}
}

func NewEdgeResponse(edge models.Edge) EdgeResponse {
	return EdgeResponse{Source: GID(edge.Payer), Target: GID(edge.Payee), SumKZT: edge.SumKZT, NTx: edge.TxCount}
}

func NewClusterResponse(cluster models.ClusterResult) ClusterResponse {
	roles := make(map[string]int, len(cluster.RoleCounts))
	for role, count := range cluster.RoleCounts {
		roles[string(role)] = count
	}
	return ClusterResponse{ID: cluster.ClusterID, Component: cluster.ComponentID, NNodes: cluster.NodeCount, NSeed: cluster.SeedCount,
		SumKZTInternal: cluster.SumKZTInternal, SumKZTIn: cluster.SumKZTIn, SumKZTOut: cluster.SumKZTOut, NCycles: cluster.CycleCount,
		Roles: roles, TopGids: GIDs(cluster.TopGIDs), Hypothesis: cluster.Hypothesis}
}

func NewTopNodeResponse(top models.TopNode) TopNodeResponse {
	return TopNodeResponse{Rank: top.Rank, Gid: GID(top.GID), Role: string(top.Role), Priority: top.PriorityScore, Why: top.Why}
}

func NewRobustnessResponse(step models.RobustnessStep) RobustnessResponse {
	return RobustnessResponse{Removed: len(step.RemovedGIDs), RemovedGids: GIDs(step.RemovedGIDs), LostTurnoverShare: step.LostTurnoverShare,
		ComponentsBefore: step.ComponentsBefore, ComponentsAfter: step.ComponentsAfter, LargestBefore: step.LargestBefore, LargestAfter: step.LargestAfter,
		NodesLostAllIncoming: step.NodesLostAllPayers, SeedsDisconnected: step.SeedsDisconnected}
}

func NewFeaturesResponse(f models.Features) FeaturesResponse {
	return FeaturesResponse{Depth: f.Depth, IsSeed: f.IsSeed, InDeg: f.InDegree, OutDeg: f.OutDegree, InKZT: f.InKZT, OutKZT: f.OutKZT,
		InTx: f.InTxCount, OutTx: f.OutTxCount, AvgInTx: f.AvgInTx, AvgOutTx: f.AvgOutTx, PassThrough: f.PassThrough,
		PageRank: f.PageRank, Hub: f.Hub, Authority: f.Authority, Betweenness: f.Betweenness,
		NSeedPayers: f.SeedPayers, NSeedUpstream: f.SeedUpstream, NClustersIn: f.ClustersIn, ComponentID: f.ComponentID, ComponentSize: f.ComponentSize,
		Truncated: f.Truncated, VerifiedSink: f.VerifiedSink, ActiveDays: f.ActiveDays, MaxSameDayPayers: f.MaxSameDayPayers,
		FastForwardShare: f.FastForwardShare, Reciprocal: f.Reciprocal, InCycle: f.InCycle, RepeatRoutes: f.RepeatRoutes}
}
