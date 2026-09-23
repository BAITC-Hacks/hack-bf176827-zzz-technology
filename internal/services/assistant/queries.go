package assistant

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	graphdto "hackaton/internal/data/dto/graph"
	"hackaton/internal/data/models"
	"hackaton/internal/services/analysis"
)

const (
	counterpartyLimit = 10
	defaultListLimit  = 20
	maxListLimit      = 50
	defaultDepth      = 2
	maxDepth          = 3
	defaultPathLen    = 4
	maxPathLen        = 6
)

func parseGID(raw string) (int64, error) {
	gid, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidGID, raw)
	}
	return gid, nil
}

func (s *service) findNode(raw string) (*models.NodeResult, error) {
	gid, err := parseGID(raw)
	if err != nil {
		return nil, err
	}
	node, ok := s.result.Node(gid)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, raw)
	}
	return node, nil
}

func (s *service) queryNode(raw string) (nodeCard, error) {
	node, err := s.findNode(raw)
	if err != nil {
		return nodeCard{}, err
	}
	return s.buildNodeCard(node), nil
}

func (s *service) buildNodeCard(node *models.NodeResult) nodeCard {
	return nodeCard{
		Node:      newNodeBrief(node),
		RoleScore: node.RoleScore,
		Features:  graphdto.NewFeaturesResponse(node.Features),
		Payers:    s.counterparties(s.index.Incoming[node.GID], func(e models.Edge) int64 { return e.Payer }),
		Receivers: s.counterparties(s.index.Outgoing[node.GID], func(e models.Edge) int64 { return e.Payee }),
	}
}

func (s *service) counterparties(edges []models.Edge, other func(models.Edge) int64) []counterparty {
	list := make([]counterparty, 0, min(len(edges), counterpartyLimit))
	for i, edge := range edges {
		if i >= counterpartyLimit {
			break
		}
		gid := other(edge)
		item := counterparty{GID: graphdto.GID(gid), SumKZT: edge.SumKZT, TxN: edge.TxCount}
		if node, ok := s.result.Node(gid); ok {
			item.Role, item.IsSeed = string(node.Role), node.Features.IsSeed
		}
		list = append(list, item)
	}
	return list
}

func (s *service) queryFindNodes(args toolArgs) map[string]any {
	limit := clampLimit(args.Limit, defaultListLimit, maxListLimit)
	var found []nodeBrief
	for _, i := range sortedByPriority(s.result) {
		node := &s.result.Nodes[i]
		if args.Role != "" && string(node.Role) != args.Role {
			continue
		}
		if args.ClusterID != nil && node.ClusterID != *args.ClusterID {
			continue
		}
		if args.IsSeed != nil && node.Features.IsSeed != *args.IsSeed {
			continue
		}
		if node.PriorityScore < args.MinPriority {
			continue
		}
		found = append(found, newNodeBrief(node))
		if len(found) >= limit {
			break
		}
	}
	return map[string]any{"count": len(found), "nodes": found}
}

func (s *service) queryReceiversOf(rawGIDs []string, depth int) (map[string]any, error) {
	depth = clampLimit(depth, defaultDepth, maxDepth)
	sources, err := parseGIDs(rawGIDs)
	if err != nil {
		return nil, err
	}
	reached := map[int64]*reachedNode{}
	for _, source := range sources {
		for gid, distance := range s.index.Downstream([]int64{source}, depth) {
			if gid == source {
				continue
			}
			node, ok := s.result.Node(gid)
			if !ok {
				continue
			}
			item := reached[gid]
			if item == nil {
				item = &reachedNode{nodeBrief: newNodeBrief(node), Distance: distance}
				reached[gid] = item
			}
			item.FromCount++
			item.From = append(item.From, graphdto.GID(source))
			item.Distance = min(item.Distance, distance)
			for _, edge := range s.index.Outgoing[source] {
				if edge.Payee == gid {
					if item.DirectSums == nil {
						item.DirectSums = map[string]float64{}
					}
					item.DirectSums[graphdto.GID(source)] = edge.SumKZT
				}
			}
		}
	}
	receivers := make([]reachedNode, 0, len(reached))
	for _, item := range reached {
		receivers = append(receivers, *item)
	}
	sort.Slice(receivers, func(i, j int) bool {
		if receivers[i].FromCount != receivers[j].FromCount {
			return receivers[i].FromCount > receivers[j].FromCount
		}
		return receivers[i].Priority > receivers[j].Priority
	})
	if len(receivers) > 25 {
		receivers = receivers[:25]
	}
	return map[string]any{"sources": len(sources), "depth": depth, "receivers": receivers}, nil
}

func (s *service) queryPayersOf(raw string, depth int) (map[string]any, error) {
	node, err := s.findNode(raw)
	if err != nil {
		return nil, err
	}
	depth = clampLimit(depth, defaultDepth, maxDepth)
	type payer struct {
		nodeBrief
		Distance int `json:"distance"`
	}
	var payers []payer
	seeds := 0
	for gid, distance := range s.index.Upstream([]int64{node.GID}, depth) {
		if gid == node.GID {
			continue
		}
		upstream, ok := s.result.Node(gid)
		if !ok {
			continue
		}
		if upstream.Features.IsSeed {
			seeds++
		}
		payers = append(payers, payer{nodeBrief: newNodeBrief(upstream), Distance: distance})
	}
	sort.Slice(payers, func(i, j int) bool {
		if payers[i].Distance != payers[j].Distance {
			return payers[i].Distance < payers[j].Distance
		}
		return payers[i].OutKZT > payers[j].OutKZT
	})
	if len(payers) > 30 {
		payers = payers[:30]
	}
	return map[string]any{"gid": raw, "depth": depth, "n_payers": len(payers), "n_seed_among": seeds, "payers": payers}, nil
}

func (s *service) queryPath(rawSrc, rawDst string, maxLen int) (map[string]any, error) {
	src, err := parseGID(rawSrc)
	if err != nil {
		return nil, err
	}
	dst, err := parseGID(rawDst)
	if err != nil {
		return nil, err
	}
	maxLen = clampLimit(maxLen, defaultPathLen, maxPathLen)
	path := s.index.Path(src, dst, maxLen)
	if path == nil {
		return map[string]any{"found": false}, nil
	}
	steps := make([]pathStep, 0, len(path)-1)
	for i := 0; i+1 < len(path); i++ {
		step := pathStep{From: graphdto.GID(path[i]), To: graphdto.GID(path[i+1])}
		for _, edge := range s.index.Outgoing[path[i]] {
			if edge.Payee == path[i+1] {
				step.SumKZT = edge.SumKZT
			}
		}
		if node, ok := s.result.Node(path[i+1]); ok {
			step.ToRole = string(node.Role)
		}
		steps = append(steps, step)
	}
	return map[string]any{"found": true, "hops": len(steps), "steps": steps}, nil
}

func (s *service) queryCluster(id *int) (map[string]any, error) {
	if id == nil {
		return nil, fmt.Errorf("%w: cluster_id обязателен", ErrClusterNotFound)
	}
	cluster, ok := s.result.Cluster(*id)
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrClusterNotFound, *id)
	}
	top := make([]nodeBrief, 0, len(cluster.TopGIDs))
	for _, gid := range cluster.TopGIDs {
		if node, ok := s.result.Node(gid); ok {
			top = append(top, newNodeBrief(node))
		}
	}
	return map[string]any{"cluster_id": cluster.ClusterID, "n_nodes": cluster.NodeCount, "n_seed": cluster.SeedCount,
		"sum_kzt_internal": cluster.SumKZTInternal, "sum_kzt_in": cluster.SumKZTIn, "sum_kzt_out": cluster.SumKZTOut,
		"roles": cluster.RoleCounts, "top": top, "hypothesis": cluster.Hypothesis}, nil
}

func (s *service) queryTop(n int, role string) []nodeBrief {
	n = clampLimit(n, 10, maxListLimit)
	var top []nodeBrief
	for _, i := range sortedByPriority(s.result) {
		node := &s.result.Nodes[i]
		if role != "" && string(node.Role) != role {
			continue
		}
		top = append(top, newNodeBrief(node))
		if len(top) >= n {
			break
		}
	}
	return top
}

func (s *service) queryRemoveNodes(rawGIDs []string) (graphdto.RobustnessResponse, error) {
	gids, err := parseGIDs(rawGIDs)
	if err != nil {
		return graphdto.RobustnessResponse{}, err
	}
	removed := make(map[int64]bool, len(gids))
	for _, gid := range gids {
		removed[gid] = true
	}
	return graphdto.NewRobustnessResponse(analysis.Robustness(s.result, removed)), nil
}

func parseGIDs(raw []string) ([]int64, error) {
	gids := make([]int64, 0, len(raw))
	for _, item := range raw {
		gid, err := parseGID(item)
		if err != nil {
			return nil, err
		}
		gids = append(gids, gid)
	}
	return gids, nil
}

func clampLimit(value, fallback, maxValue int) int {
	if value <= 0 || value > maxValue {
		return fallback
	}
	return value
}

func sortedByPriority(result *models.AnalysisResult) []int {
	order := make([]int, len(result.Nodes))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		left, right := result.Nodes[order[a]], result.Nodes[order[b]]
		if left.PriorityScore != right.PriorityScore {
			return left.PriorityScore > right.PriorityScore
		}
		return left.GID < right.GID
	})
	return order
}
