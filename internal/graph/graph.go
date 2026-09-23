// Package graph — направленный взвешенный граф переводов поверх датасета. Node ID = gid.
package graph

import (
	"sort"

	"hackaton/internal/data/parquet"

	"gonum.org/v1/gonum/graph/simple"
)

type Graph struct {
	G     *simple.WeightedDirectedGraph // вес = sum_kzt
	Nodes map[int64]parquet.Node
	Gids  []int64 // все gid в порядке nodes.parquet
	Out   map[int64][]parquet.Edge
	In    map[int64][]parquet.Edge
	TxOut map[int64][]parquet.Tx // отсортированы по дате
	TxIn  map[int64][]parquet.Tx
	Seeds []int64
}

func Build(ds parquet.Dataset) *Graph {
	g := &Graph{
		G:     simple.NewWeightedDirectedGraph(0, 0),
		Nodes: make(map[int64]parquet.Node, len(ds.Nodes)),
		Gids:  make([]int64, 0, len(ds.Nodes)),
		Out:   map[int64][]parquet.Edge{},
		In:    map[int64][]parquet.Edge{},
		TxOut: map[int64][]parquet.Tx{},
		TxIn:  map[int64][]parquet.Tx{},
	}
	for _, n := range ds.Nodes {
		g.Nodes[n.Gid] = n
		g.Gids = append(g.Gids, n.Gid)
		g.G.AddNode(simple.Node(n.Gid))
		if n.IsSeed {
			g.Seeds = append(g.Seeds, n.Gid)
		}
	}
	for _, e := range ds.Edges {
		if e.Src == e.Dst {
			continue
		}
		for _, id := range []int64{e.Src, e.Dst} {
			if _, ok := g.Nodes[id]; !ok {
				g.Nodes[id] = parquet.Node{Gid: id, Depth: -1}
				g.Gids = append(g.Gids, id)
				g.G.AddNode(simple.Node(id))
			}
		}
		g.G.SetWeightedEdge(simple.WeightedEdge{F: simple.Node(e.Src), T: simple.Node(e.Dst), W: e.SumKZT})
		g.Out[e.Src] = append(g.Out[e.Src], e)
		g.In[e.Dst] = append(g.In[e.Dst], e)
	}
	for _, t := range ds.Tx {
		g.TxOut[t.Src] = append(g.TxOut[t.Src], t)
		g.TxIn[t.Dst] = append(g.TxIn[t.Dst], t)
	}
	byDate := func(m map[int64][]parquet.Tx) {
		for _, txs := range m {
			sort.Slice(txs, func(i, j int) bool { return txs[i].Date.Before(txs[j].Date) })
		}
	}
	byDate(g.TxOut)
	byDate(g.TxIn)
	return g
}

func (g *Graph) IsSeed(gid int64) bool { return g.Nodes[gid].IsSeed }
func (g *Graph) Depth(gid int64) int   { return int(g.Nodes[gid].Depth) }
