// Package graph предоставляет доступ к неизменяемому результату анализа.
package graph

import (
	"context"
	"strconv"
	"strings"

	"go.uber.org/fx"
	"hackaton/internal/analysis"
	"hackaton/internal/config"
	"hackaton/internal/data/parquet"
	"hackaton/pkg/httperr"
)

type Filter struct {
	Role               string
	Cluster, Component *int
	TopOnly            int
}
type Subgraph struct {
	Nodes []analysis.NodeResult
	Edges []parquet.Edge
}
type NodeCard struct {
	Node               analysis.NodeResult
	Incoming, Outgoing []parquet.Edge
}
type Service interface {
	Graph(Filter) Subgraph
	Node(int64) (*NodeCard, error)
	Ego(int64, int) (Subgraph, error)
	Top(int) []analysis.TopNode
	Clusters() []analysis.ClusterResult
	Search(string, int) []analysis.NodeResult
}
type service struct {
	result  *analysis.Result
	in, out map[int64][]parquet.Edge
}

// NewService загружает данные до открытия HTTP-порта; далее результат только читается.
func NewService(lc fx.Lifecycle, cfg *config.Config) Service {
	s := &service{}
	lc.Append(fx.Hook{OnStart: func(context.Context) error {
		ds, err := parquet.Load(cfg.App.DataDir)
		if err != nil {
			return err
		}
		result, err := analysis.Run(ds, analysis.Options{TopN: len(ds.Nodes)})
		if err != nil {
			return err
		}
		s.initialize(result)
		return nil
	}})
	return s
}
func (s *service) initialize(result *analysis.Result) {
	s.result = result
	s.in = map[int64][]parquet.Edge{}
	s.out = map[int64][]parquet.Edge{}
	for _, e := range result.Edges {
		s.out[e.Src] = append(s.out[e.Src], e)
		s.in[e.Dst] = append(s.in[e.Dst], e)
	}
}
func (s *service) expand(ids map[int64]bool, depth int) map[int64]bool {
	frontier := make([]int64, 0, len(ids))
	for id := range ids {
		frontier = append(frontier, id)
	}
	for step := 0; step < depth; step++ {
		next := []int64{}
		for _, id := range frontier {
			for _, e := range s.out[id] {
				if !ids[e.Dst] {
					ids[e.Dst] = true
					next = append(next, e.Dst)
				}
			}
			for _, e := range s.in[id] {
				if !ids[e.Src] {
					ids[e.Src] = true
					next = append(next, e.Src)
				}
			}
		}
		frontier = next
	}
	return ids
}
func (s *service) subset(ids map[int64]bool) Subgraph {
	g := Subgraph{Nodes: []analysis.NodeResult{}, Edges: []parquet.Edge{}}
	for _, n := range s.result.Nodes {
		if ids[n.Gid] {
			g.Nodes = append(g.Nodes, n)
		}
	}
	for _, e := range s.result.Edges {
		if ids[e.Src] && ids[e.Dst] {
			g.Edges = append(g.Edges, e)
		}
	}
	return g
}
func (s *service) Graph(f Filter) Subgraph {
	ids := map[int64]bool{}
	if f.TopOnly > 0 {
		for _, n := range s.Top(f.TopOnly) {
			ids[n.Gid] = true
		}
		s.expand(ids, 1)
	} else {
		for _, n := range s.result.Nodes {
			ids[n.Gid] = true
		}
	}
	for _, n := range s.result.Nodes {
		if f.Role != "" && n.Role != f.Role || f.Cluster != nil && n.ClusterID != *f.Cluster || f.Component != nil && n.Features.ComponentID != *f.Component {
			delete(ids, n.Gid)
		}
	}
	return s.subset(ids)
}
func (s *service) Node(gid int64) (*NodeCard, error) {
	n, ok := s.result.ByGid[gid]
	if !ok {
		return nil, httperr.NotFound("node_not_found", "Узел не найден")
	}
	return &NodeCard{Node: *n, Incoming: s.in[gid], Outgoing: s.out[gid]}, nil
}
func (s *service) Ego(gid int64, depth int) (Subgraph, error) {
	if depth < 1 || depth > 2 {
		return Subgraph{}, httperr.BadRequest("invalid_depth", "depth должен быть 1 или 2")
	}
	if _, err := s.Node(gid); err != nil {
		return Subgraph{}, err
	}
	return s.subset(s.expand(map[int64]bool{gid: true}, depth)), nil
}
func (s *service) Top(n int) []analysis.TopNode {
	if n < 0 {
		n = 0
	}
	if n > len(s.result.Top) {
		n = len(s.result.Top)
	}
	return s.result.Top[:n]
}
func (s *service) Clusters() []analysis.ClusterResult { return s.result.Clusters }
func (s *service) Search(prefix string, limit int) []analysis.NodeResult {
	out := []analysis.NodeResult{}
	if limit <= 0 {
		return out
	}
	for _, n := range s.result.Nodes {
		if strings.HasPrefix(strconv.FormatInt(n.Gid, 10), prefix) {
			out = append(out, n)
			if len(out) == limit {
				break
			}
		}
	}
	return out
}
