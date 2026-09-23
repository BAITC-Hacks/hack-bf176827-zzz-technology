// Package graph — направленный взвешенный граф переводов и индекс рёбер. Node ID в gonum = GID.
package graph

import (
	"sort"

	"hackaton/internal/data/models"

	"gonum.org/v1/gonum/graph/simple"
)

type Graph struct {
	Weighted   *simple.WeightedDirectedGraph // вес ребра = SumKZT
	Nodes      map[int64]models.Node
	GIDs       []int64 // в порядке nodes.parquet
	Seeds      []int64
	Outgoing   map[int64][]models.Edge
	Incoming   map[int64][]models.Edge
	TxOutgoing map[int64][]models.Transaction // отсортированы по дате
	TxIncoming map[int64][]models.Transaction
}

func Build(dataset models.Dataset) *Graph {
	txGraph := &Graph{
		Weighted:   simple.NewWeightedDirectedGraph(0, 0),
		Nodes:      make(map[int64]models.Node, len(dataset.Nodes)),
		GIDs:       make([]int64, 0, len(dataset.Nodes)),
		Outgoing:   map[int64][]models.Edge{},
		Incoming:   map[int64][]models.Edge{},
		TxOutgoing: map[int64][]models.Transaction{},
		TxIncoming: map[int64][]models.Transaction{},
	}
	for _, node := range dataset.Nodes {
		txGraph.addNode(node)
	}
	for _, edge := range dataset.Edges {
		if edge.Payer == edge.Payee {
			continue
		}
		for _, gid := range []int64{edge.Payer, edge.Payee} {
			if _, known := txGraph.Nodes[gid]; !known {
				txGraph.addNode(models.Node{GID: gid, Depth: -1})
			}
		}
		txGraph.Weighted.SetWeightedEdge(simple.WeightedEdge{F: simple.Node(edge.Payer), T: simple.Node(edge.Payee), W: edge.SumKZT})
		txGraph.Outgoing[edge.Payer] = append(txGraph.Outgoing[edge.Payer], edge)
		txGraph.Incoming[edge.Payee] = append(txGraph.Incoming[edge.Payee], edge)
	}
	for _, tx := range dataset.Transactions {
		txGraph.TxOutgoing[tx.Payer] = append(txGraph.TxOutgoing[tx.Payer], tx)
		txGraph.TxIncoming[tx.Payee] = append(txGraph.TxIncoming[tx.Payee], tx)
	}
	sortByDate(txGraph.TxOutgoing)
	sortByDate(txGraph.TxIncoming)
	return txGraph
}

func (g *Graph) addNode(node models.Node) {
	g.Nodes[node.GID] = node
	g.GIDs = append(g.GIDs, node.GID)
	g.Weighted.AddNode(simple.Node(node.GID))
	if node.IsSeed {
		g.Seeds = append(g.Seeds, node.GID)
	}
}

func (g *Graph) IsSeed(gid int64) bool { return g.Nodes[gid].IsSeed }
func (g *Graph) Depth(gid int64) int   { return g.Nodes[gid].Depth }

// Edge — ребро payer→payee, если есть.
func (g *Graph) Edge(payer, payee int64) (models.Edge, bool) {
	for _, edge := range g.Outgoing[payer] {
		if edge.Payee == payee {
			return edge, true
		}
	}
	return models.Edge{}, false
}

func sortByDate(byNode map[int64][]models.Transaction) {
	for _, txs := range byNode {
		sort.Slice(txs, func(i, j int) bool { return txs[i].Date.Before(txs[j].Date) })
	}
}
