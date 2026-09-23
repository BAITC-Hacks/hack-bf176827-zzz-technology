package dto

import (
	"hackaton/internal/analysis"
	"hackaton/internal/data/parquet"
	graphservice "hackaton/internal/services/graph"
	"strconv"
)

type NodeDTO struct {
	Gid       string  `json:"id"`
	Role      string  `json:"role"`
	RoleScore float64 `json:"role_score"`
	Cluster   int     `json:"cluster"`
	Priority  float64 `json:"priority"`
	Evidence  string  `json:"evidence"`
	analysis.Features
}
type EdgeDTO struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	SumKZT float64 `json:"sum_kzt"`
	NTx    int64   `json:"n_tx"`
	Depth  int8    `json:"depth"`
}
type NeighborDTO struct {
	Gid    string  `json:"gid"`
	SumKZT float64 `json:"sum_kzt"`
	NTx    int64   `json:"n_tx"`
}
type NodeCardDTO struct {
	Node     NodeDTO       `json:"node"`
	Incoming []NeighborDTO `json:"incoming"`
	Outgoing []NeighborDTO `json:"outgoing"`
}
type ClusterDTO struct {
	ID             int      `json:"id"`
	Component      int      `json:"component"`
	NNodes         int      `json:"n_nodes"`
	NSeed          int      `json:"n_seed"`
	SumKZTInternal float64  `json:"sum_kzt_internal"`
	TopGids        []string `json:"top_gids"`
	Hypothesis     string   `json:"hypothesis"`
}
type TopDTO struct {
	Rank     int     `json:"rank"`
	Gid      string  `json:"gid"`
	Role     string  `json:"role"`
	Priority float64 `json:"priority"`
	Why      string  `json:"why"`
}
type GraphResponse struct {
	Nodes []NodeDTO `json:"nodes"`
	Edges []EdgeDTO `json:"edges"`
}

func MapNode(n analysis.NodeResult) NodeDTO {
	return NodeDTO{Gid: strconv.FormatInt(n.Gid, 10), Role: n.Role, RoleScore: n.RoleScore, Cluster: n.ClusterID, Priority: n.PriorityScore, Evidence: n.Evidence, Features: n.Features}
}
func MapNodes(nodes []analysis.NodeResult) []NodeDTO {
	out := make([]NodeDTO, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, MapNode(n))
	}
	return out
}
func MapEdge(e parquet.Edge) EdgeDTO {
	return EdgeDTO{Source: strconv.FormatInt(e.Src, 10), Target: strconv.FormatInt(e.Dst, 10), SumKZT: e.SumKZT, NTx: e.NTx, Depth: e.Depth}
}
func MapGraph(g graphservice.Subgraph) GraphResponse {
	out := GraphResponse{Nodes: MapNodes(g.Nodes), Edges: make([]EdgeDTO, 0, len(g.Edges))}
	for _, e := range g.Edges {
		out.Edges = append(out.Edges, MapEdge(e))
	}
	return out
}
func MapCard(c *graphservice.NodeCard) NodeCardDTO {
	out := NodeCardDTO{Node: MapNode(c.Node), Incoming: []NeighborDTO{}, Outgoing: []NeighborDTO{}}
	for _, e := range c.Incoming {
		out.Incoming = append(out.Incoming, NeighborDTO{Gid: strconv.FormatInt(e.Src, 10), SumKZT: e.SumKZT, NTx: e.NTx})
	}
	for _, e := range c.Outgoing {
		out.Outgoing = append(out.Outgoing, NeighborDTO{Gid: strconv.FormatInt(e.Dst, 10), SumKZT: e.SumKZT, NTx: e.NTx})
	}
	return out
}
func MapClusters(clusters []analysis.ClusterResult) []ClusterDTO {
	out := make([]ClusterDTO, 0, len(clusters))
	for _, c := range clusters {
		gids := make([]string, 0, len(c.TopGids))
		for _, g := range c.TopGids {
			gids = append(gids, strconv.FormatInt(g, 10))
		}
		out = append(out, ClusterDTO{ID: c.ClusterID, Component: c.ComponentID, NNodes: c.NNodes, NSeed: c.NSeed, SumKZTInternal: c.SumKZTInternal, TopGids: gids, Hypothesis: c.Hypothesis})
	}
	return out
}
func MapTop(nodes []analysis.TopNode) []TopDTO {
	out := make([]TopDTO, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, TopDTO{Rank: n.Rank, Gid: strconv.FormatInt(n.Gid, 10), Role: n.Role, Priority: n.PriorityScore, Why: n.Why})
	}
	return out
}
