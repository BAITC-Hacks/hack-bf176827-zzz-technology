package analysis

import (
	"sort"

	"hackaton/internal/data/parquet"
	"hackaton/internal/graph"

	gg "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/network"
	"gonum.org/v1/gonum/graph/topo"
)

// computeFeatures — все метрики узла, кроме зависящих от кластеров (NClustersIn) и паттернов (InCycle).
func computeFeatures(g *graph.Graph) map[int64]Features {
	out := make(map[int64]Features, len(g.Gids))
	seedSet := map[int64]bool{}
	for _, s := range g.Seeds {
		seedSet[s] = true
	}

	pr := network.PageRank(g.G, 0.85, 1e-8)
	hits := network.HITS(g.G, 1e-8)
	btw := network.Betweenness(g.G)
	upstream := seedUpstream(g, 4)
	comp := components(g)

	for _, gid := range g.Gids {
		f := Features{Depth: g.Depth(gid), IsSeed: g.IsSeed(gid), PassThrough: -1}
		for _, e := range g.In[gid] {
			f.InDeg++
			f.InKZT += e.SumKZT
			f.InTx += int(e.NTx)
			if seedSet[e.Src] {
				f.NSeedPayers++
			}
			if _, back := edgeBetween(g, gid, e.Src); back {
				f.Reciprocal = true
			}
		}
		for _, e := range g.Out[gid] {
			f.OutDeg++
			f.OutKZT += e.SumKZT
			f.OutTx += int(e.NTx)
		}
		if f.InTx > 0 {
			f.AvgIn = f.InKZT / float64(f.InTx)
		}
		if f.OutTx > 0 {
			f.AvgOut = f.OutKZT / float64(f.OutTx)
		}
		if f.InKZT > 0 {
			f.PassThrough = f.OutKZT / f.InKZT
		}
		f.PageRank = pr[gid]
		f.Hub, f.Authority = hits[gid].Hub, hits[gid].Authority
		f.Betweenness = btw[gid]
		f.NSeedUpstream = upstream[gid]
		f.ComponentID, f.ComponentSize = comp[gid].id, comp[gid].size
		f.Truncated = f.Depth == 4 && f.OutDeg == 0
		f.VerifiedSink = f.Depth < 4 && f.OutDeg == 0 && f.InDeg > 0
		temporal(g, gid, &f)
		out[gid] = f
	}
	return out
}

func edgeBetween(g *graph.Graph, from, to int64) (parquet.Edge, bool) {
	for _, e := range g.Out[from] {
		if e.Dst == to {
			return e, true
		}
	}
	return parquet.Edge{}, false
}

// seedUpstream — сколько разных seed достигают узла по исходящим за ≤ maxHops шагов.
func seedUpstream(g *graph.Graph, maxHops int) map[int64]int {
	count := map[int64]int{}
	for _, s := range g.Seeds {
		seen := map[int64]bool{s: true}
		frontier := []int64{s}
		for hop := 0; hop < maxHops && len(frontier) > 0; hop++ {
			var next []int64
			for _, u := range frontier {
				for _, e := range g.Out[u] {
					if !seen[e.Dst] {
						seen[e.Dst] = true
						next = append(next, e.Dst)
					}
				}
			}
			frontier = next
		}
		for gid := range seen {
			if gid != s {
				count[gid]++
			}
		}
	}
	return count
}

type compInfo struct{ id, size int }

// components — слабосвязные компоненты; id по убыванию размера (0 = крупнейшая).
func components(g *graph.Graph) map[int64]compInfo {
	cc := topo.ConnectedComponents(gg.Undirect{G: g.G})
	sort.SliceStable(cc, func(i, j int) bool {
		if len(cc[i]) != len(cc[j]) {
			return len(cc[i]) > len(cc[j])
		}
		return cc[i][0].ID() < cc[j][0].ID()
	})
	out := map[int64]compInfo{}
	for id, c := range cc {
		for _, n := range c {
			out[n.ID()] = compInfo{id: id, size: len(c)}
		}
	}
	return out
}

// temporal — признаки по датам транзакций.
func temporal(g *graph.Graph, gid int64, f *Features) {
	days := map[string]bool{}
	payersByDay := map[string]map[int64]bool{}
	var inDates []int64
	for _, t := range g.TxIn[gid] {
		d := t.Date.Format("2006-01-02")
		days[d] = true
		if payersByDay[d] == nil {
			payersByDay[d] = map[int64]bool{}
		}
		payersByDay[d][t.Src] = true
		inDates = append(inDates, t.Date.Unix()/86400)
	}
	for _, m := range payersByDay {
		if len(m) > f.MaxSameDayPayers {
			f.MaxSameDayPayers = len(m)
		}
	}
	var fast, total float64
	for _, t := range g.TxOut[gid] {
		days[t.Date.Format("2006-01-02")] = true
		total += t.SumKZT
		d := t.Date.Unix() / 86400
		for _, in := range inDates {
			if d-in >= 0 && d-in <= 2 {
				fast += t.SumKZT
				break
			}
		}
	}
	f.ActiveDays = len(days)
	if total > 0 && len(inDates) > 0 {
		f.FastForwardShare = fast / total
	}
}
