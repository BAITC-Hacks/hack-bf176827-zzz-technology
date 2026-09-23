package analysis

import (
	"math"
	"sort"

	"hackaton/internal/data/graph"
	"hackaton/internal/data/models"

	gonumgraph "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/network"
	"gonum.org/v1/gonum/graph/topo"
)

const (
	pageRankDamping = 0.85
	seedUpstreamMax = 4 // колен от seed, в пределах которых считаем SeedUpstream
	fastForwardDays = 2 // окно «пришло и ушло» для FastForwardShare
)

// computeFeatures — метрики, не зависящие от кластеров и паттернов.
func computeFeatures(txGraph *graph.Graph) map[int64]models.Features {
	seeds := make(map[int64]bool, len(txGraph.Seeds))
	for _, gid := range txGraph.Seeds {
		seeds[gid] = true
	}
	pageRank := weightedPageRank(txGraph, pageRankDamping, 1e-10, 200)
	hits := network.HITS(txGraph.Weighted, 1e-8)
	betweenness := network.Betweenness(txGraph.Weighted)
	seedUpstream := countSeedUpstream(txGraph, seedUpstreamMax)
	components := weakComponents(txGraph)

	features := make(map[int64]models.Features, len(txGraph.GIDs))
	for _, gid := range txGraph.GIDs {
		f := models.Features{Depth: txGraph.Depth(gid), IsSeed: txGraph.IsSeed(gid), PassThrough: -1}
		for _, edge := range txGraph.Incoming[gid] {
			f.InDegree++
			f.InKZT += edge.SumKZT
			f.InTxCount += int(edge.TxCount)
			if seeds[edge.Payer] {
				f.SeedPayers++
			}
			if _, back := txGraph.Edge(gid, edge.Payer); back {
				f.Reciprocal = true
			}
		}
		for _, edge := range txGraph.Outgoing[gid] {
			f.OutDegree++
			f.OutKZT += edge.SumKZT
			f.OutTxCount += int(edge.TxCount)
		}
		if f.InTxCount > 0 {
			f.AvgInTx = f.InKZT / float64(f.InTxCount)
		}
		if f.OutTxCount > 0 {
			f.AvgOutTx = f.OutKZT / float64(f.OutTxCount)
		}
		if f.InKZT > 0 {
			f.PassThrough = f.OutKZT / f.InKZT
		}
		f.PageRank = pageRank[gid]
		// gonum обходит map: округляем, чтобы значения совпадали между запусками
		f.Hub, f.Authority = round(hits[gid].Hub, 1e9), round(hits[gid].Authority, 1e9)
		f.Betweenness = round(betweenness[gid], 1e6)
		f.SeedUpstream = seedUpstream[gid]
		f.ComponentID, f.ComponentSize = components[gid].id, components[gid].size
		f.Truncated = f.Depth == 4 && f.OutDegree == 0
		f.VerifiedSink = f.Depth < 4 && f.OutDegree == 0 && f.InDegree > 0
		fillTemporal(txGraph, gid, &f)
		features[gid] = f
	}
	return features
}

// countSeedUpstream — сколько разных seed достигают узла по исходящим за ≤ maxHops шагов.
func countSeedUpstream(txGraph *graph.Graph, maxHops int) map[int64]int {
	count := map[int64]int{}
	for _, seed := range txGraph.Seeds {
		reached := map[int64]bool{seed: true}
		frontier := []int64{seed}
		for hop := 0; hop < maxHops && len(frontier) > 0; hop++ {
			var next []int64
			for _, gid := range frontier {
				for _, edge := range txGraph.Outgoing[gid] {
					if !reached[edge.Payee] {
						reached[edge.Payee] = true
						next = append(next, edge.Payee)
					}
				}
			}
			frontier = next
		}
		for gid := range reached {
			if gid != seed {
				count[gid]++
			}
		}
	}
	return count
}

type componentInfo struct{ id, size int }

// weakComponents — слабосвязные компоненты; id по убыванию размера, при равенстве по минимальному GID.
func weakComponents(txGraph *graph.Graph) map[int64]componentInfo {
	components := topo.ConnectedComponents(gonumgraph.Undirect{G: txGraph.Weighted})
	sort.SliceStable(components, func(i, j int) bool {
		if len(components[i]) != len(components[j]) {
			return len(components[i]) > len(components[j])
		}
		return minNodeID(components[i]) < minNodeID(components[j])
	})
	byGID := map[int64]componentInfo{}
	for id, component := range components {
		for _, node := range component {
			byGID[node.ID()] = componentInfo{id: id, size: len(component)}
		}
	}
	return byGID
}

func minNodeID(nodes []gonumgraph.Node) int64 {
	minID := nodes[0].ID()
	for _, node := range nodes[1:] {
		if node.ID() < minID {
			minID = node.ID()
		}
	}
	return minID
}

// fillTemporal — признаки по датам транзакций.
func fillTemporal(txGraph *graph.Graph, gid int64, f *models.Features) {
	activeDays := map[string]bool{}
	payersByDay := map[string]map[int64]bool{}
	var incomingDays []int64
	for _, tx := range txGraph.TxIncoming[gid] {
		day := tx.Date.Format("2006-01-02")
		activeDays[day] = true
		if payersByDay[day] == nil {
			payersByDay[day] = map[int64]bool{}
		}
		payersByDay[day][tx.Payer] = true
		incomingDays = append(incomingDays, tx.Date.Unix()/86400)
	}
	for _, payers := range payersByDay {
		if len(payers) > f.MaxSameDayPayers {
			f.MaxSameDayPayers = len(payers)
		}
	}
	var fastOut, totalOut float64
	for _, tx := range txGraph.TxOutgoing[gid] {
		activeDays[tx.Date.Format("2006-01-02")] = true
		totalOut += tx.SumKZT
		outDay := tx.Date.Unix() / 86400
		for _, inDay := range incomingDays {
			if outDay-inDay >= 0 && outDay-inDay <= fastForwardDays {
				fastOut += tx.SumKZT
				break
			}
		}
	}
	f.ActiveDays = len(activeDays)
	if totalOut > 0 && len(incomingDays) > 0 {
		f.FastForwardShare = fastOut / totalOut
	}
}

func round(value, scale float64) float64 { return math.Round(value*scale) / scale }
